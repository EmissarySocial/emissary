# service/content — Notes for AI Agents

See [doc.go](doc.go) for what this package is, and the project plan in `emissary-specs/projects/GIT-MARKDOWN-TO-STREAM-CONTENT.md` for why. These are the rules that are not visible in the code.

## The media type is read, never assumed, and never sniffed

A forge's file *page* and its raw file differ by one path segment, and the page answers `text/html`: `github.com/golang/go/blob/master/README.md` returns a 200 and a full HTML document. An author pasting the URL from their browser is the expected mistake, not the exotic one, and storing that page as a Stream's Markdown body fails silently and looks almost right.

So `contentFormat` decides from the declared `Content-Type` and refuses everything else. Do NOT add byte sniffing as a fallback: an HTML page begins with plain text, so `http.DetectContentType` would wave through the exact case this guard exists to catch. A source that declares nothing is refused for the same reason.

The explicit empty-header branch looks redundant — `mime.ParseMediaType("")` errors on its own — and it is kept for the message alone, because "did not declare a Content-Type" and "declared an unreadable Content-Type" send an author to different places. `TestHTTPS_Fetch_NoContentTypeSaysSo` pins the message rather than the refusal, because a test that only pins the refusal passes with the branch deleted.

## Every response is capped before it is read, and the cap matches the Stream schema

`maxContentBytes` is 1 MiB because that is the `maxLength` of `content.raw` in the article Templates, so an adapter can never return a body the Stream schema would then reject. `Fetch` reads through an `io.LimitReader` set one byte past the cap, so a body exactly at the limit is kept and anything larger is refused — `TestHTTPS_Fetch_SizeLimit` pins both sides, and an off-by-one here reads as a working size check right up until someone's 1 MiB page stops syncing.

## Configuration mistakes are client errors, so they stay out of the error log

A missing file, a private URL, a refused media type, an oversize body, and an invalid address all return a 4xx `derp` code. Only a network failure or a 5xx from the origin is a 500. The caller files 5xx errors as defects and shows 4xx errors to the Stream author as a status message, so a new failure path that forgets to classify itself files every author's typo into the production error log. `checkStatus` does this for a response; a `401` and a `403` are deliberately 4xx, because both mean the same thing to the author — this is not a public URL.

## A refusal must not echo the address back

`parseSourceURL` replaces `url.Parse`'s own error rather than wrapping it, because that error quotes the whole address and an address can carry a password. The same reasoning keeps the address out of the credentials refusal. `TestParseSourceURL_PasswordNeverEchoed` pins it.

## `Version` exists for one thing: the stored ETag that makes the next request conditional

`StreamSource.Version` holds the `ETag` from the last successful sync, and `Version()` sends it back as `If-None-Match`. An unchanged source then answers `304` with no body, which is the one case this method exists to win. Its only readers are `StreamSource.Sync` and this adapter; `GetPointer` exposes the field to the schema, but no feature depends on it.

`Version()` is **not** a cheap probe. It issues a full `GET`, not a `HEAD`, so on any response other than `304` it downloads the file, `closeBody` throws it away, and `Fetch` downloads the same file again. Measured second-sync cost against an httptest origin:

| Origin | Requests | Body downloaded |
| --- | --- | --- |
| sends ETag, nothing changed | 1 | none — this is the win |
| sends ETag, content changed | 2 | twice |
| no ETag, nothing changed | 2 | twice |
| no ETag, content changed | 2 | twice |

So the split pays off in one case of four and doubles the download in the other three. That is accepted deliberately: a documentation file is a few kilobytes, the no-change case is the common one on a repository that changes a few times a week, and the alternative costs an interface change. **Do not repeat the old rationale that this abstraction keeps the change check "protocol-independent"** — it was written for a Git adapter that returned a commit SHA, where a cheap check really was orders of magnitude cheaper than the fetch. D12 deleted Git, and that argument went with it.

If this ever needs to be cheaper, the fix is **not** to have `Version` keep the body it downloaded — that re-introduces a cache this package deliberately does not have. It is to move `If-None-Match` into `Fetch` and drop `Version` from the `Adapter` interface, which is one request in all four rows above. The cost is that `Fetch` then needs a way to say "not modified" (a third return value, or a flag on `Item`), since a `304` is not a `derp` error.

## An empty Version is a source with no validator, not a failure

Five of the seven forges surveyed answer `304` to `If-None-Match`; cgit ignores it and SourceHut sends no `ETag` at all. `Version` returns `""` for those rather than erroring, which means every ping fetches and `ContentHash` decides whether anything actually changed. Do not "fix" the empty string into an error — it would take two working forges offline.

`ContentHash`, not `Version`, is what guards the expensive operation. `Stream.Save` federates, notifies, and rewrites the whole document (C4), and the hash is what stops it running for unchanged bytes — on every forge, including the two that offer no validator. `Version` only ever saves bandwidth.

## Tests bind to 127.0.0.1, which the SSRF guard refuses

`remote.NewHTTPClient(false)` will not connect to a private address, and `httptest` serves from one. Every test therefore builds the adapter with `allowPrivateIPs` TRUE. A test written with FALSE fails to connect and passes for the wrong reason, proving nothing about the code under test.

## Emissary does not speak Git any more

An earlier build read repositories over the Git smart-HTTP protocol: upload-pack sessions, a packfile parser, a declared-size memory budget, and a shared snapshot cache — 2,711 lines to download a 47 MiB repository in order to read a 4 KB file. It was deleted in favour of one HTTPS GET. If a future adapter needs Git — a private repository is the only real candidate — recover the reasoning from D12 in the project plan first, and note that an HTTP GET with a token header is the easier path even there.

The warning that rule replaced is still true and now lives only here: **never read an author-supplied address through `hairyhenderson/go-fsimpl/gitfs` or go-git's `Clone`/`Remote.List`.** gitfs authenticates with `AutoAuthenticator`, which reads `GIT_HTTP_TOKEN` and `GIT_SSH_KEY` from the process environment, so a server holding a token for private Template packages would send it to whatever host the author typed. Both also reach the network through go-git's process-wide protocol table, which has no SSRF guard. Template packages still load that way, and that is fine — their addresses come from the operator, not from a Stream author.
