# mailchimp

A thin client for the [Mailchimp Marketing API v3](https://mailchimp.com/developer/marketing/api/).

This package speaks HTTP and nothing else. It has no dependency on Emissary's services,
sessions, or model objects, so every function can be tested against an `httptest` server
with no Factory and no database. Mailchimp publishes no official Go SDK, so it is
hand-rolled over [`benpate/remote`](https://github.com/benpate/remote).

Its consumer is the per-User mailing list integration described in
[MAILING-LISTS.md](../../../emissary-specs/projects/MAILING-LISTS.md).

## Credentials are opaque

Nothing in this package parses a Mailchimp credential. `ValidateAPIKey` checks that a
value can be *stored and sent* — non-empty, bounded, printable ASCII so it survives an
`Authorization` header — and nothing else.

That is deliberate, and it is the correction of an earlier design that extracted the data
center from the key:

- **The format is Mailchimp's to change.** An API key looks like `<32 hex>-us6` today.
  Nothing contractual says it will tomorrow, and a parser that is right about a vendor's
  undocumented format is right by luck.
- **OAuth tokens do not share it.** Mailchimp accepts an API key *or* an OAuth token on
  the same endpoints, and an OAuth token carries no data center at all — you call a
  metadata endpoint for that. Any parser is wrong for half the credential types the API
  itself accepts.
- **Mailchimp's own clients do not parse it.** Their official Node and Python libraries
  take `server` as a separate config value beside `apiKey`.

Whether a credential actually works is answered by calling the API, which setup does
before saving anything.

## `BaseURL` is the only way to address the API

Mailchimp has no single API hostname. Each account lives in a *data center* addressed as
a subdomain:

```
https://us6.api.mailchimp.com/3.0
```

Since the credential is opaque, this value is its own piece of configuration: the User
supplies it during setup, reading it from the start of their own Mailchimp web address
(`us6.admin.mailchimp.com`).

That makes it user-supplied text that becomes a hostname, so it is validated before it is
composed. Interpolating it unchecked would be server-side request forgery: a value like
`evil.example.com/x#` would send Emissary's requests, carrying Emissary's credentials and
originating from Emissary's own network position, wherever the User liked. Servers
commonly sit inside a private network where that position is worth something.

`BaseURL` is the mitigation, and it is a *structural* one:

- It validates against `^[a-z0-9]{2,32}$` before composing anything — a bare DNS label,
  with no dots, slashes, colons, or at-signs.
- Nothing else in Emissary composes a Mailchimp address.

Note what the guard is *not*: a blocklist of frightening words. A data center can only
ever become one label underneath `api.mailchimp.com`, so `localhost` yields
`localhost.api.mailchimp.com` — Mailchimp's to resolve, not the loopback. Do not "harden"
the pattern against such values; it would reject a future data center and add nothing.
`TestBaseURL_ScaryButHarmless` pins this so the reasoning is not re-litigated.

A rule that developers have to remember is not a mitigation. A single constructor is.
If you find yourself writing `"https://" + something` for Mailchimp, that is the bug.

## Credentials

A Mailchimp API key carries **full account access and has no scopes**. Holding one means
being able to read, alter, or delete the account and its campaigns. Two consequences:

- Keys live in the `Vault` of a `UserConnection`, encrypted, never in its `Data` (which exports).
- Nothing in this package logs a key or quotes one in an error message, including the
  errors that reject a malformed one.

`ValidateAPIKey` is not proof that a key works -- only Mailchimp can say that, and setup
asks before saving.

## What this package covers

Four resources, which together are the whole integration:

| Call | Used by |
|---|---|
| `GetAudiences` / `GetAudience` | proving a credential, and confirming the Audience ID a User pasted |
| `GetMergeFields` / `CreateMergeField` | creating `EMISSARYID`, which links a member back to a Follower |
| `GetWebhooks` / `CreateWebhook` / `DeleteWebhook` | inbound events |
| `SetMember` / `UnsubscribeMember` | the outbound sync itself |

Two things about members are easy to get wrong and silent when you do.

**A member is addressed by `SubscriberHash` — the MD5 of the lowercased address.** Mailchimp
accepts no other addressing, and a mis-cased hash does not error: it names a member who does
not exist. The MD5 here is an identifier, not a security primitive, which is what the `#nosec`
annotations record.

**`SetMember` is a `PUT`, and that is load-bearing.** It upserts, so `Follower.Save` can call
it on every save without first asking whether the member is already there. `UnsubscribeMember`
likewise treats a 404 as success, because Emissary pushes only on confirmation — so someone
who unsubscribes before their first push was never a member, and failing there would retry
forever.

Optional fields are **omitted rather than blanked**. A Follower with no name or no signup IP
sends neither, because an empty string would overwrite whatever the User's own signup forms
had already collected.

## Errors are written for the person reading the form

A failed request comes back through `describeError`, and the split it makes is deliberate.

The four statuses a User can *act on* -- 401, 403, 404, 429 -- become **validation (422)
errors whose message is the root of the chain**. That placement is what makes them survive:
`build.inlineErrorMessage` reads the root message of a 422 and the outermost message of
anything else, and these errors travel up through a service and a pipeline step that each
wrap them. A sentence written anywhere but the root would be replaced by "Unable to connect
to Mailchimp" before anyone saw it.

Everything else -- a 500, a timeout, a DNS failure -- keeps its cause and stays non-422,
because it is not the User's input being wrong and no instruction would help them.

Those four also do **not** wrap the failed transaction. `derp` redacts credential-bearing
headers when it records one, so the leak this originally guarded against is closed at the
source; not wrapping is now belt-and-braces, and `TestGetAudiences_DoesNotCarryTheCredential`
keeps it honest.

## Member calls keep Mailchimp's status code — form calls do not

There are two error paths here, and routing a call through the wrong one fails silently.

`describeError` is for the **setup/form** path (`GetAudience`, `GetMergeFields`,
`CreateWebhook`). It rewrites the four actionable statuses into 422s so the sentence
survives to the form, as described above.

`describeMemberError` is for the **queue** path (`SetMember`, `UnsubscribeMember`), and it
keeps Mailchimp's own status code instead. Nothing on that path is shown to a User, and two
readers depend on the code:

- `requeue` tells a retry from a permanent failure by status. As a 422, a **429 rate limit**
  read as a client error and the queue gave up for good — silently dropping that subscriber
  from the mailing list. A bulk sync is exactly what provokes a 429, so this is the busiest
  path, not the rarest.
- `mailchimp_reportMemberError` flags a connection `RECONNECT` when it sees a **401/403**.
  As a 422 that check never fired, so a revoked key left the connection reading `READY`
  while every push failed in silence.

The rule generalizes: **a 422 is a message for a human, so it is only correct where a human
reads it.** Anything a queue interprets must keep its transport semantics. The member path
also leaves the failed transaction out of the chain, because it carries an `Authorization`
header and nobody on that path benefits from the detail.
