#!/usr/bin/env bash
# End-to-end test of Emissary's reply controls (emissary-specs REPLY-CONTROLS) and quote approvals (FEP-044f).  See README.md.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$HERE/../.." && pwd)"
STARTED=$(date +%s)

# Everything this run creates is named after its PID, so cleanup can never touch anything else
RUN_ID="$$"
DB_PREFIX="rctest_${RUN_ID}_"
DB_A="${DB_PREFIX}a"
DB_B="${DB_PREFIX}b"
DB_COMMON="${DB_PREFIX}common"
MONGO_URI="${RC_MONGO_URI:-mongodb://127.0.0.1:27017/?directConnection=true}"
export RC_MONGO_URI="$MONGO_URI"

# Domain A (the author) and domain B (the replier) are two names for this machine, served by one process
HOST_A="localhost"
HOST_B="127.0.0.1"

WORK="$(mktemp -d "${TMPDIR:-/tmp}/replycontrols.XXXXXX")"
HELPER="$WORK/rchelper"
SERVER_PID=""
PASSED=0
FAILED=0
FAILURES=()

#############################################
# Cleanup
#############################################

cleanup() {
	local status=$?
	set +e

	if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
		kill "$SERVER_PID" 2>/dev/null
		for _ in $(seq 1 50); do kill -0 "$SERVER_PID" 2>/dev/null || break; sleep 0.1; done
		kill -9 "$SERVER_PID" 2>/dev/null
	fi

	if [[ -x "$HELPER" ]]; then
		for db in "$DB_A" "$DB_B" "$DB_COMMON"; do "$HELPER" drop-db "$db" 2>/dev/null; done
	fi

	if [[ "$status" -ne 0 && -f "$WORK/server.log" ]]; then
		echo
		echo "---- last 40 lines of the test server's log ----"
		grep -v '^\.\.\.' "$WORK/server.log" | tail -40
	fi

	if [[ "${RC_KEEP:-}" == "1" ]]; then
		echo "Kept $WORK (RC_KEEP=1)"
	else
		rm -rf "$WORK"
	fi

	exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

#############################################
# Reporting
#############################################

log() { printf '  .. %s\n' "$*"; }

pass() {
	PASSED=$((PASSED + 1))
	printf 'PASS  %-34s %s\n' "$1" "$2"
}

fail() {
	FAILED=$((FAILED + 1))
	FAILURES+=("$1")
	printf 'FAIL  %-34s %s\n' "$1" "$2"
}

# check NAME EVIDENCE TEST...  runs TEST and reports PASS or FAIL with the evidence
check() {
	local name="$1" evidence="$2"
	shift 2
	if "$@"; then pass "$name" "$evidence"; else fail "$name" "$evidence"; fi
}

die() {
	echo "ERROR: $*" >&2
	exit 2
}

#############################################
# Build and start a private server
#############################################

command -v curl >/dev/null || die "curl is required"
command -v go >/dev/null || die "go is required"

echo "Building the server and helper from $REPO"
(cd "$REPO" && go build -o "$WORK/emissary" .) || die "server build failed"
(cd "$REPO" && go build -tags replycontrols -o "$HELPER" ./integration-tests/replycontrols/rchelper) || die "helper build failed"

"$HELPER" list-dbs rctest_ >/dev/null 2>&1 || die "MongoDB is not reachable at $MONGO_URI (see README.md)"

# Drop databases left by an earlier run that was killed before its trap could fire
for db in $("$HELPER" list-dbs rctest_); do
	pid="${db#rctest_}"
	pid="${pid%%_*}"
	if [[ "$pid" =~ ^[0-9]+$ ]] && ! kill -0 "$pid" 2>/dev/null; then
		log "dropping stale database $db"
		"$HELPER" drop-db "$db"
	fi
done

PORT="$("$HELPER" freeport)"
A="http://$HOST_A:$PORT"
B="http://$HOST_B:$PORT"
MASTER_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
mkdir -p "$WORK/files"

cat >"$WORK/config.json" <<EOF
{
	"domains": [
		{"domainId": "000000000000000000000a01", "label": "Reply controls A", "hostname": "$HOST_A",
		 "connectString": "$MONGO_URI", "databaseName": "$DB_A", "smtp": {},
		 "owner": {"displayName": "Alice", "username": "alice", "emailAddress": "alice@example.com"},
		 "masterKey": "$MASTER_KEY", "createOwner": true},
		{"domainId": "000000000000000000000b01", "label": "Reply controls B", "hostname": "$HOST_B",
		 "connectString": "$MONGO_URI", "databaseName": "$DB_B", "smtp": {},
		 "owner": {"displayName": "Bob", "username": "bob", "emailAddress": "bob@example.com"},
		 "masterKey": "$MASTER_KEY", "createOwner": true}
	],
	"templates": [{"adapter": "EMBED", "location": "templates"}],
	"attachmentOriginals": {"adapter": "FILE", "location": "$WORK/files/attachments"},
	"attachmentCache": {"adapter": "FILE", "location": "$WORK/files/cache"},
	"exportCache": {"adapter": "FILE", "location": "$WORK/files/exports"},
	"certificates": {"adapter": "FILE", "location": "$WORK/files/certificates"},
	"activityPubCache": {"connectString": "$MONGO_URI", "database": "$DB_COMMON"},
	"adminEmail": "",
	"httpPort": $PORT,
	"httpsPort": 0,
	"debugLevel": "Info",
	"loggers": [{"type": "console"}],
	"clientIpStrategy": "REMOTE-ADDR",
	"allowPrivateIPs": true
}
EOF

echo "Starting a private server on port $PORT (databases ${DB_PREFIX}*)"
(cd "$WORK" && exec "$WORK/emissary" --config "file://$WORK/config.json" >"$WORK/server.log" 2>&1) &
SERVER_PID=$!

#############################################
# HTTP helpers
#############################################

JAR_A="$WORK/alice.cookies"
JAR_B="$WORK/bob.cookies"

# signin BASE JAR USERNAME prints the HTTP status of a password sign-in
signin() {
	curl -s -m 10 -o /dev/null -w '%{http_code}' -c "$2" -b "$2" \
		--data-urlencode "username=$3" --data-urlencode "password=demo" "$1/signin" || true
}

# wait_for_signin BASE JAR USERNAME signs in, retrying until the domain has booted and created its owner
wait_for_signin() {
	for _ in $(seq 1 120); do
		kill -0 "$SERVER_PID" 2>/dev/null || die "the test server exited during startup"
		[[ "$(signin "$1" "$2" "$3")" == "204" ]] && return 0
		sleep 0.5
	done
	die "could not sign in as $3 at $1"
}

# new_note CONTENT creates a public outbox note as Alice and prints its Stream ID
new_note() {
	local code
	code=$(curl -s -m 30 -o /dev/null -w '%{http_code}' -b "$JAR_A" \
		--form-string "content=$1" "$A/@me/outbox-add?templateId=outbox-message")
	[[ "$code" == "200" ]] || die "creating a note returned HTTP $code"
	"$HELPER" latest-stream "$DB_A" outbox-message
}

# set_policy STREAM_ID POLICY CONTENT stores the policy, then republishes the note through its edit form
set_policy() {
	local code
	"$HELPER" set-reply-policy "$DB_A" "$1" "$2"
	code=$(curl -s -m 30 -o /dev/null -w '%{http_code}' -b "$JAR_A" --form-string "content=$3" "$A/$1/edit")
	[[ "$code" == "303" ]] || die "republishing $1 returned HTTP $code"
}

# reply_as_bob URL CONTENT replies through Bob's inbox composer, waits for delivery, and prints the reply's URL
reply_as_bob() {
	local location reply
	location=$(curl -s -m 30 -o /dev/null -D - -b "$JAR_B" --data-urlencode "content=$2" \
		"$B/@me/newsfeed/reply?templateId=outbox-reply&url=$1" | tr -d '\r' | sed -n 's/^Location: //Ip')
	reply="${location#*url=}"
	[[ "$reply" == "$B/"* ]] || die "replying to $1 did not redirect to a new reply (Location: $location)"
	"$HELPER" wait-inbox "$DB_A" "$reply" 30 || die "the reply $reply never reached Alice's inbox"
	"$HELPER" wait-queue-idle "$DB_COMMON" 30 || die "the queue did not drain after $reply"
	echo "$reply"
}

# wait_replies URL EXPECTED polls a Replies collection until it reads EXPECTED, then prints what it read
wait_replies() {
	local got=""
	for _ in $(seq 1 40); do
		got=$("$HELPER" replies "$1")
		[[ "$got" == "$2" ]] && break
		sleep 0.25
	done
	echo "$got"
}

# reply_count_label STREAM_ID prints the builder's ReplyCount as drawn on the note ("1 Reply")
reply_count_label() {
	curl -s -m 10 -H 'HX-Request: true' "$A/$1/responses-replies" | grep -oE '[0-9]+ (Reply|Replies)' | head -1 || true
}

# bob_view VIEW URL prints Bob's inbox view of a post, as htmx requests it
bob_view() {
	curl -s -m 30 -b "$JAR_B" -H 'HX-Request: true' "$B/@me/newsfeed/$1?url=$2" || true
}

# reply_button_state URL prints "shown", "hidden", or "unrendered" for the Reply button on Bob's thread view
reply_button_state() {
	local html
	html=$(bob_view browse-tree "$1")
	if ! grep -q "$NOTE_TEXT" <<<"$html"; then echo "unrendered"
	elif grep -qF "browse-reply?url=$1\"" <<<"$html"; then echo "shown"
	else echo "hidden"; fi
}

# composer_state URL prints "form", "refusal", or "neither" for Bob's reply composer
composer_state() {
	local html
	html=$(bob_view browse-reply "$1")
	if grep -q 'id="outbox-reply"' <<<"$html"; then echo "form"
	elif grep -q 'This post does not accept replies from you.' <<<"$html"; then echo "refusal"
	else echo "neither"; fi
}

# starts_with VALUE PREFIX succeeds when VALUE begins with PREFIX
starts_with() { [[ "$1" == "$2"* ]]; }

# is_2xx CODE succeeds for any HTTP success status
is_2xx() { [[ "$1" =~ ^2[0-9][0-9]$ ]]; }

# http_status URL prints the HTTP status of an ActivityPub GET
http_status() {
	curl -s -m 10 -o /dev/null -w '%{http_code}' -H 'Accept: application/activity+json' "$1" || true
}

# quote_as_bob QUOTE_ID ATTRIBUTED_TO URL delivers a QuoteRequest signed with Bob's key, and prints the HTTP status
quote_as_bob() {
	"$HELPER" quote-request "$DB_B" "$BOB_ID" "$3" "$1" "$2"
}

# alice_fragment ACTION prints one of Alice's notification fragments, as htmx requests it
alice_fragment() {
	curl -s -m 30 -b "$JAR_A" -H 'HX-Request: true' "$A/@me/notifications/$1" || true
}

# quote_modal_query prints the query string of the first quote row on Alice's Quotes list
quote_modal_query() {
	alice_fragment quotes-list | grep -oE 'quote\?notificationId=[0-9a-f]{24}' | head -1 | cut -d'?' -f2 || true
}

# reject_quote_query MODAL_QUERY prints the query string of the Reject button in a quote's modal
reject_quote_query() {
	[[ -n "$1" ]] || return 0
	alice_fragment "quote?$1" | grep -oE 'reject-quote\?collectionItemId=[0-9a-f]{24}' | head -1 | cut -d'?' -f2 || true
}

# row_state LIST prints "unread", "read", or "missing" for the quote row on one of Alice's lists
row_state() {
	local html
	html=$(alice_fragment "$1" | tr '\n' ' ' | grep -oE '<a[^>]*>' | grep 'quote?notificationId=' | head -1 || true)
	if [[ -z "$html" ]]; then echo "missing"
	elif grep -q 'unread' <<<"$html"; then echo "unread"
	else echo "read"; fi
}

#############################################
# Seed
#############################################

wait_for_signin "$A" "$JAR_A" alice
wait_for_signin "$B" "$JAR_B" bob
log "signed in as alice@$HOST_A and bob@$HOST_B"

NOTE_TEXT="Reply controls note $RUN_ID"
S1="$(new_note "<p>$NOTE_TEXT</p>")"
S1_URL="$A/$S1"
ALICE_ID="$(curl -s -m 10 -H 'Accept: application/activity+json' "$S1_URL" | sed -n 's/.*"attributedTo":"\([^"]*\)".*/\1/p')"
[[ -n "$ALICE_ID" ]] || die "could not read attributedTo from $S1_URL"
log "S1 = $S1_URL (author $ALICE_ID)"

#############################################
# 1. Publish
#############################################

echo
echo "== 1. Publish the policy"

if out=$("$HELPER" check-policy "$S1_URL" public 2>&1); then pass "1a open: canReply and canQuote are Public" "$out"
else fail "1a open: canReply and canQuote are Public" "$out"; fi

#############################################
# 3 and 4 while OPEN
#############################################

echo
echo "== Replies OPEN on S1"

R1="$(reply_as_bob "$S1_URL" "<p>Open reply $RUN_ID</p>")"
log "R1 = $R1"

got=$(wait_replies "$S1_URL/pub/replies" "1 $R1")
check "3a open: reply listed" "/replies = [$got]" test "$got" == "1 $R1"

label=$(reply_count_label "$S1")
check "3b open: reply counted" "ReplyCount drawn as '$label'" test "$label" == "1 Reply"

count=$("$HELPER" count-notifications "$DB_A" REPLY "$R1")
check "3c open: author notified" "$count REPLY notification(s) for R1" test "$count" == "1"

button=$(reply_button_state "$S1_URL")
composer=$(composer_state "$S1_URL")
BOB_ID="$(sed -n 's/.*"attributedTo":"\([^"]*\)".*/\1/p' <<<"$(curl -s -m 10 -H 'Accept: application/activity+json' "$R1")")"
diag=$("$HELPER" diagnose-cache "$DB_COMMON" "$S1_URL" "$BOB_ID" 2>&1 || true)
check "4a open: Reply button shown" "button=$button; canReply for Bob: $diag" test "$button" == "shown"
check "4b open: composer offered" "composer=$composer" test "$composer" == "form"

#############################################
# 1, 3 and 4 after CLOSING
#############################################

echo
echo "== Replies CLOSED on S1 (replyPolicy=nobody in the DB, then republished through /edit)"

set_policy "$S1" nobody "<p>$NOTE_TEXT</p>"

if out=$("$HELPER" check-policy "$S1_URL" "$ALICE_ID" 2>&1); then pass "1b closed: canReply and canQuote are the author" "$out"
else fail "1b closed: canReply and canQuote are the author" "$out"; fi

got=$(wait_replies "$S1_URL/pub/replies" "0")
check "3d closed: replies hidden" "/replies = [$got]" test "$got" == "0"

label=$(reply_count_label "$S1")
check "3e closed: count reads zero" "ReplyCount drawn as '$label'" test "$label" == "0 Replies"

stored=$("$HELPER" reply-count "$DB_A" "$S1")
check "3f closed: nothing deleted" "stored Stream.replyCount = $stored" test "$stored" == "1"

button=$(reply_button_state "$S1_URL")
composer=$(composer_state "$S1_URL")
diag=$("$HELPER" diagnose-cache "$DB_COMMON" "$S1_URL" "$BOB_ID" 2>&1 || true)
check "4c closed: Reply button hidden" "button=$button; canReply for Bob: $diag" test "$button" == "hidden"
check "4d closed: composer refuses" "composer=$composer" test "$composer" == "refusal"

#############################################
# 2. Refuse at the inbox
#############################################

echo
echo "== 2. A reply to S2, which is CLOSED from the start"

S2="$(new_note "<p>Closed note $RUN_ID</p>")"
S2_URL="$A/$S2"
set_policy "$S2" nobody "<p>Closed note $RUN_ID</p>"
log "S2 = $S2_URL"

R2="$(reply_as_bob "$S2_URL" "<p>Refused reply $RUN_ID</p>")"
log "R2 = $R2 (delivered: it is in $DB_A.Inbox)"

count=$("$HELPER" count-notifications "$DB_A" REPLY "$R2")
check "2a closed: no REPLY notification" "$count REPLY notification(s) for R2" test "$count" == "0"

stored=$("$HELPER" reply-count "$DB_A" "$S2")
check "2b closed: ReplyCount stays 0" "stored Stream.replyCount = $stored" test "$stored" == "0"

set_policy "$S2" everyone "<p>Closed note $RUN_ID</p>"
got=$("$HELPER" replies "$S2_URL/pub/replies")
check "2c closed: never recorded" "/replies after reopening S2 = [$got]" test "$got" == "0"

#############################################
# 3 and 4 after REOPENING
#############################################

echo
echo "== Replies REOPENED on S1"

set_policy "$S1" everyone "<p>$NOTE_TEXT</p>"

if out=$("$HELPER" check-policy "$S1_URL" public 2>&1); then pass "1c reopened: canReply and canQuote are Public" "$out"
else fail "1c reopened: canReply and canQuote are Public" "$out"; fi

got=$(wait_replies "$S1_URL/pub/replies" "1 $R1")
check "3g reopened: reply listed again" "/replies = [$got]" test "$got" == "1 $R1"

label=$(reply_count_label "$S1")
check "3h reopened: reply counted again" "ReplyCount drawn as '$label'" test "$label" == "1 Reply"

button=$(reply_button_state "$S1_URL")
composer=$(composer_state "$S1_URL")
diag=$("$HELPER" diagnose-cache "$DB_COMMON" "$S1_URL" "$BOB_ID" 2>&1 || true)
check "4e reopened: Reply button shown" "button=$button; canReply for Bob: $diag" test "$button" == "shown"
check "4f reopened: composer offered" "composer=$composer" test "$composer" == "form"

#############################################
# 5. Quote posts (FEP-044f)
#############################################

echo
echo "== 5. Quotes of S1 (open), each QuoteRequest signed with Bob's own key"

# Q1 is a real post on Bob's server, because a rejection looks up the quoting actor from the quote post itself
Q1="$(reply_as_bob "$S1_URL" "<p>Quote post $RUN_ID</p>")"
log "Q1 = $Q1"
code=$(quote_as_bob "$Q1" "$BOB_ID" "$S1_URL")
check "5a open: QuoteRequest received" "HTTP $code" test "$code" == "200"

"$HELPER" wait-inbox "$DB_B" "$Q1#quote_request" 30 || die "the Accept of $Q1 never reached Bob's inbox"
"$HELPER" wait-queue-idle "$DB_COMMON" 30 || die "the queue did not drain after $Q1"
STAMP=$("$HELPER" accept-result "$DB_B" "$Q1#quote_request")
check "5b open: Accept names an authorization" "result = $STAMP" starts_with "$STAMP" "$S1_URL/pub/quoteAuthorization/"

if out=$("$HELPER" check-authorization "$STAMP" "$S1_URL" "$Q1" "$ALICE_ID" 2>&1); then pass "5c open: QuoteAuthorization verifies" "$out"
else fail "5c open: QuoteAuthorization verifies" "$out"; fi

count=$("$HELPER" count-notifications "$DB_A" QUOTE "$Q1")
check "5d open: author notified" "$count QUOTE notification(s) for Q1" test "$count" == "1"

MODAL_QUERY=$(quote_modal_query)
check "5e open: quote listed under Quotes" "row opens quote?$MODAL_QUERY" test -n "$MODAL_QUERY"

REJECT_QUERY=$(reject_quote_query "$MODAL_QUERY")
check "5e open: Reject button in the modal" "query = '$REJECT_QUERY'" test -n "$REJECT_QUERY"

echo
echo "== 6. Notification tabs, and marking one row read"

state=$(row_state all-list)
check "6a All lists the quote" "quote row on all-list is $state" test "$state" == "unread"

html=$(alice_fragment all-list)
check "6b All lists the reply too" "all-list holds 'replied to you'" grep -q 'replied to you' <<<"$html"

state=$(row_state replies-list)
check "6c Messages & Replies leaves quotes out" "quote row on replies-list is $state" test "$state" == "missing"

code=$(curl -s -m 30 -o /dev/null -w '%{http_code}' -b "$JAR_A" -H 'HX-Request: true' "$A/@me/notifications/read-one?$MODAL_QUERY" || true)
state=$(row_state quotes-list)
check "6d a GET marks nothing read" "HTTP $code, row is $state" test "$state" == "unread"

code=$(curl -s -m 30 -o /dev/null -w '%{http_code}' -b "$JAR_A" -H 'HX-Request: true' -X POST "$A/@me/notifications/read-one?$MODAL_QUERY" || true)
state=$(row_state quotes-list)
check "6e a POST marks that row read" "HTTP $code, row is $state" test "$code/$state" == "200/read"

broken=""
for page in all replies quotes shares likes followers "header?type=ALL" "sidebar?type=QUOTE"; do
	code=$(curl -s -m 30 -o /dev/null -w '%{http_code}' -b "$JAR_A" -H 'HX-Request: true' "$A/@me/notifications/$page" || true)
	[[ "$code" == "200" ]] || broken="$broken $page=$code"
done
check "6g every tab page renders" "failures:${broken:- none}" test -z "$broken"

# The thread view of Bob's reply must select the reply's own card, which needs a hashed ID on a saved post (BUG-383)
html=$(curl -s -m 30 -b "$JAR_A" -H 'HX-Request: true' "$A/@me/newsfeed/browse?url=$R1" || true)
selected=$(grep -oE 'set selectedNode to #id_[a-z0-9]*' <<<"$html" | head -1 | sed 's/.*#//')
matches=$(grep -oE "id=\"$selected\"" <<<"$html" | wc -l | tr -d ' ')
check "6h a reply opens on its own card" "selected '$selected', which names $matches card(s)" test "$selected/$matches" != "id_/$matches" -a "$matches" == "1"

html=$(alice_fragment replies-list)
check "6f the other rows stay unread" "replies-list still holds an unread row" grep -q 'notification[^"]*unread' <<<"$html"

Q_FORGED="$B/quotes/$RUN_ID-forged"
code=$(quote_as_bob "$Q_FORGED" "$B/@someone-else" "$S1_URL")
count=$("$HELPER" count-notifications "$DB_A" QUOTE "$Q_FORGED")
check "5f forged: someone else's post refused" "HTTP $code, $count QUOTE notification(s)" test "$code/$count" == "422/0"

echo
echo "== 5. Quotes CLOSED on S1"

set_policy "$S1" nobody "<p>$NOTE_TEXT</p>"

status=$(http_status "$STAMP")
check "5g closed: authorization withdrawn" "GET = HTTP $status" test "$status" == "404"

Q2="$B/quotes/$RUN_ID-2"
code=$(quote_as_bob "$Q2" "$BOB_ID" "$S1_URL")
"$HELPER" wait-queue-idle "$DB_COMMON" 30 || die "the queue did not drain after $Q2"
answers=$("$HELPER" count-inbox "$DB_B" "$Q2#quote_request")
count=$("$HELPER" count-notifications "$DB_A" QUOTE "$Q2")
check "5h closed: refused silently" "HTTP $code, $answers answer(s) to Bob, $count QUOTE notification(s)" test "$code/$answers/$count" == "200/0/0"

set_policy "$S1" everyone "<p>$NOTE_TEXT</p>"

status=$(http_status "$STAMP")
check "5i reopened: authorization served again" "GET = HTTP $status" test "$status" == "200"

echo
echo "== 5. Alice rejects Q1 from its notification"

code=$(curl -s -m 30 -o /dev/null -w '%{http_code}' -b "$JAR_A" -H 'HX-Request: true' -X POST \
	"$A/@me/notifications/reject-quote?$REJECT_QUERY" || true)
check "5j rejected: action succeeds" "HTTP $code" is_2xx "$code"

if "$HELPER" wait-inbox "$DB_B" "$STAMP" 30; then pass "5k rejected: Delete reached Bob" "Bob's inbox holds a Delete of $STAMP"
else fail "5k rejected: Delete reached Bob" "no Delete of $STAMP in Bob's inbox"; fi

status=$(http_status "$STAMP")
check "5l rejected: authorization withdrawn" "GET = HTTP $status" test "$status" == "404"

count=$("$HELPER" count-notifications "$DB_A" QUOTE "$Q1")
check "5m rejected: notification removed" "$count QUOTE notification(s) for Q1" test "$count" == "0"

#############################################
# Summary
#############################################

echo
echo "$PASSED passed, $FAILED failed in $(($(date +%s) - STARTED))s"

if [[ "$FAILED" -gt 0 ]]; then
	printf '  failed: %s\n' "${FAILURES[@]}"
	exit 1
fi
