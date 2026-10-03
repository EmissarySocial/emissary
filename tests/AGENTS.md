# tests — Notes for AI Agents

See [README.md](README.md) for what this suite pins and how to run it.

## A failing pin is a behavior change, not a broken test

Every golden file was generated from current behavior, so a failure means a stored value, an error, or a type is now different. Find the cause and decide whether the change is intended before running `-update`. Never regenerate goldens to make a failure go away; the diff of `testdata/` is the record of what a change did to visitors' data.

## The steps are reproduced, so the reproductions must track the code

A real `Builder` needs a factory and a database, so `cases_test.go` makes the same form, schema, and model calls that each step's `Post` makes, and `contexts_test.go` picks the same object and schema each builder does. `TestEmulatedSources` fingerprints every function those files reproduce, ignoring comments and formatting. When it fails, re-read the function it names, update the reproduction named in the message so it still makes the same calls, then run `-update`. A new step kind that reads the POST belongs in `newPostCase` or in `unpinnedPostReaders`, never in neither.

## Some results depend on map order

`edit-registration` (through rosetta's `Schema.SetURLValues`), `set-data` (its `values` and `defaults`), `add` (the request keys), and `read-form` (the schema's properties) range over Go maps and stop at the first error. So the field an error names, and the fields written before it, change from one request to the next. That is production behavior.

**Never detect this by repeating a run.** Go's map order inside one process follows the map's own hash seed, which is fixed when the map is created, so fifty runs in one test binary can all agree while the next binary sees a different order. That made an earlier version of this suite flaky. Instead, each such case sets `Orders` and `ApplyOrdered`, and `pinScenario` runs it once for every order in `mapOrders`: every rotation of each map's keys, forwards and backwards, so that every key comes first once. A result every order agrees on is pinned whole; one that varies keeps only the error text every order shared, marked `orderDependent`. `-update` still refuses to write a group that differs between two passes.

## set-data

**`values` render against a builder, which this suite does not have.** Each scenario supplies the rendered text instead, and the golden labels every template `(literal)` or `(renders against the builder; mocked)`. Only the literal ones are exact in the `literal` and `prefilled` scenarios; a mocked one renders as `""`, which is what `executeTemplate` returns when rendering fails.

**`defaults` read the current value through the builder, not the object.** `schema.Get(builder, name)` fails for every builder, because none implements a schema getter interface, so every default is applied every time (BUG-240). `builderStandIn` reproduces that. If a builder ever gains `GetPointer` or `GetStringOK`, `TestEmulatedSources` fails, and `builderStandIn` must change to match.

## What is not reproduced

- **`add` defaults.** They are other steps, run before the form is applied, and are pinned as `set-data` steps of their own.
- **Lookups.** `form.SetURLValues` gets a nil lookup provider, so a `::NEWVALUE::` option is never created.
- **Related Streams.** `with-children`, `with-parent`, `with-next-sibling`, and `with-prev-sibling` switch to a Stream whose template is known only at runtime. Each is listed in the inventory instead.
- **Steps that need the database.** Sharing, privileges, responses, passwords, sorting, attachments, and content conversion are listed in the inventory with their reasons.
