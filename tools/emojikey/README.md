This package holds the canonical emoji table used to *display* EmojiKeys — the human-friendly fingerprints that MLS clients (conversations-mls) compute from their KeyPackages and publish in the ActivityStreams `summary` field. `Parse` splits a stored summary back into its component emojis so templates can show each character with its name.

This package never *computes* EmojiKeys. The client is the only party that hashes a KeyPackage into emojis; the server just stores the resulting string and, with this table, annotates it for display. Server-side computation was removed deliberately — don't reintroduce it here.

The canonical table lives in the client at `conversations-mls/user-conversations/resources/app/src/service/emojikeys.ts`; [emojis.go](emojis.go) is a mirrored copy for display lookup only. Table changes land in the client first and are mirrored here. If the two drift, nothing breaks: characters missing from this table display without a name.
