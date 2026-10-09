# tests

Regression tests that pin what every form POST in every template does to the object it edits. They cover the whole chain at once: the template definitions, Emissary's model objects, and the `form`, `schema`, `mapof`, and `sliceof` packages beneath them. A change anywhere in that chain that alters a stored value fails a test here, even when every package's own tests still pass.

## Running

```sh
go test ./tests                                  # compare against the golden files
go test ./tests -update                          # rewrite the golden files from current behavior
go test ./tests -run '^$' -fuzz FuzzFormPosts    # post arbitrary text into every pinned form
go test ./tests -run '^$' -fuzz FuzzSetData      # the same, through every set-data step
```

Run `-update` only on purpose, and review the diff of `testdata/` as part of the change that caused it.

## What is pinned

Each **template group** is one deployment: Emissary's own templates, plus one package. A package that is not checked out beside `emissary` is skipped.

| Group | Locations, in load order |
| --- | --- |
| `emissary` | `_embed/templates` |
| `atlas`, `bandwagon`, `qwertylicious`, `qwertylicious-v2` | `_embed/templates`, then `../<package>` |

Each group is loaded by the production `service.Template`, `Theme`, `Widget`, and `Registration` services, wired the way `server/factory_core.go` wires them, so inheritance and overrides match a running server.

Every step that applies POSTed values through the form or schema packages becomes a **case**:

| Step | Applies the POST with |
| --- | --- |
| `edit` | `form.SetURLValues` on the builder's object |
| `add` | `schema.Set` of every key, as a `[]string` |
| `edit-table` | `schema.Set` of each column, for `?edit=` rows 0 to 16, `-1`, and `first` |
| `read-form` | `schema.Set` into a new `mapof.Any`, then `Validate` |
| `edit-template` | `schema.Set` of a template ID that passes the step's allow-list |
| `edit-widget` | `form.SetURLValues` on each widget definition's data |
| `edit-registration` | `schema.SetURLValues` on each registration's data |

Each case runs inside the object and schema its builder would provide: the template's schema for Stream templates, the model's schema for every other builder, and each visible theme in turn for Domain templates. Steps nested in `as-modal`, `if`, `with-draft`, and the model `with-*` steps run in their sub-builder's context.

Each case posts four **scenarios**: `typical` values, `empty` strings, `hostile` input (markup, bad formats, over-long text), and `absent` (nothing posted). For each, the golden file records the error, every stored value that changed (read through BSON, as `type value`), and the Go type left at each posted path.

## Files

| File | Holds |
| --- | --- |
| `loader_test.go` | Template groups and the production loader |
| `contexts_test.go` | The object and schema each builder provides |
| `cases_test.go` | The step walker, and one reproduction per step kind |
| `values_test.go` | The four scenarios |
| `snapshot_test.go` | Results: BSON snapshots, diffs, and types |
| `formposts_test.go` | `TestFormPosts` and the golden files |
| `setdata_test.go` | `TestSetData`, which pins every `set-data` step |
| `emulation_test.go` | `TestEmulatedSources`, which detects changes to the reproduced code |
| `fuzz_test.go` | `FuzzFormPosts` and `FuzzSetData` |
| `testdata/*.golden.json` | Form POSTs, one file per group, plus the source fingerprints |
| `testdata/*.setdata.golden.json` | `set-data` steps, one file per group |

Each golden file also has an `inventory`: every POST-reading step that is not pinned, and why.

## set-data

`TestSetData` pins every `set-data` step: in template actions, registration actions, `add` defaults, and widget `saveSteps` (in the Widget builder's context). Each step runs under `GET` (its `from-url`, `values`, and `defaults`) and `POST` (adding `from-form`), in five scenarios:

| Scenario | Request | `values` templates render as |
| --- | --- | --- |
| `literal` | nothing | the template run with no data |
| `typical`, `empty`, `hostile` | as for form POSTs | the same kind of text |
| `prefilled` | nothing | the template run with no data, against an object that already holds a value at every path the step writes |

`prefilled` is what shows whether `defaults` keep an existing value.
