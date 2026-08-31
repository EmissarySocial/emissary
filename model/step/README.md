# Pipeline Steps

This package holds the *data* for every pipeline step that a Template can use. Each `.go` file here defines one step: a struct that holds the parsed configuration, and a `New…()` constructor that compiles the raw HJSON map into that struct. The code that actually *executes* a step lives in [/build](../../build/), in a matching `step_*.go` file.

**[STEPS.md](STEPS.md) documents all 88 steps** — what each one does, its attributes, and a sample. This file covers the mechanics they share.

## How Steps Are Parsed

A step is a JSON/HJSON object whose `do` property names the step. [New()](step.go#L52) switches on that name and hands the whole map to the step's constructor, which pulls out the properties it recognizes and ignores the rest. Anything invalid — a malformed `text/template`, a missing required property, a sub-pipeline that will not parse — is an error at *Template load time*, not at request time.

```hjson
{do: "view-html", file: "detail"}
```

Several steps take a nested pipeline (usually under `steps`, sometimes `then`/`else`/`defaults`/`on-error`), which is parsed recursively by [NewPipeline()](step.go).

To hold a pipeline on your own type, declare the field as [Pipeline](step.go) rather than `[]Step`. `Step` is an interface whose concrete type is chosen at parse time by the `do` property, so a plain slice of them cannot be unmarshalled — `Pipeline` is a named `[]Step` that implements `UnmarshalJSON`, which lets the containing type decode declaratively instead of hand-parsing its steps. `model.Widget.SaveSteps` is the simplest example.

## GET vs. POST

Every step is executed twice over its lifetime — once for `GET` (build the page) and once for `POST` (handle the submission) — and most steps only do work in one of the two. A step like `edit-content` renders a form on `GET` and saves it on `POST`; a step like `save` does nothing on `GET`. Steps that could reasonably fire in either phase take a `method` property (`get`, `post`, or `both`) to pin down which.

Every step that takes one reads it through [parseMethod](utils.go), which lower-cases the value and rejects anything outside those three words at Template load time. That validation is not cosmetic: the build-side steps guard on the parsed value with hand-written comparisons, and the two shapes of that guard read an unknown value in opposite directions — an allow-list (`method == "post"`) runs the step for *nothing*, a deny-list (`method != "post"`) runs it for *everything*. Neither reports anything, so before the check a typo silently moved when a step fired. Rejecting the typo up front is what lets both shapes coexist.

Note also that a step's `method` does **not** stop the pipeline. `as-modal` and the `edit` steps return without halting on a `GET`, so a pipeline that renders a form keeps running through the steps that follow it; those steps skip themselves because of their own `method`, not because the pipeline stopped. Widening a `method` can therefore fire a step in a phase its author never considered.

## Template Requirements

Beyond its own configuration, each step declares what the surrounding Template must provide. The Template service checks these when it loads a Template, so a mismatch fails at startup rather than in front of a user.

- **RequiredModel** — the step only works against one kind of model object (`Stream`, `Domain`, `Settings`). A Template with a different `model` cannot use it.
- **RequiredStates** — states the step names (for example `save-and-publish`'s `state`) must be defined in the Template's `states` map.
- **RequiredRoles** — roles the step names (for example `set-simple-sharing`'s `role`) must be defined in the Template's `roles` map.
- **RequiredTemplateRoles** — a narrower restriction than the model: the Template must declare one of these `templateRole` values. Only `startup-create-streams` and `setup-complete` use it today, both requiring `admin`.

Container steps (`if`, `as-modal`, `with-*`, and friends) roll their sub-steps' requirements up into their own, except that the `with-*` steps deliberately drop *state* requirements, since the child object they switch to has its own state machine. `with-draft` is the exception, because a draft shares its Stream's states.

## Value Templates

Many properties are compiled as Go `text/template` values and evaluated against the Builder at runtime — [STEPS.md](STEPS.md) marks these as *template*. That is what makes `{{.Label}}`, `{{.StreamID}}`, and similar expressions work inside step configuration. A handful of steps ([inlineError](inlineError.go), [inlineSuccess](inlineSuccess.go), [inlineSaveButton](inlineSaveButton.go), [setData](setData.go), [triggerEvent](triggerEvent.go), [viewJSON](viewJSON.go)) also install the helper [FuncMap()](functions.go).

## Adding a New Step

1. Add `newStepName.go` here with the struct, the `New…()` constructor, and the four `Step` interface methods (`Name`, `RequiredModel`, `RequiredStates`, `RequiredRoles`).
2. Implement `GetForm() form.Element` if the step renders a form, and `RequiredTemplateRoles() []string` if it is restricted to a particular kind of Template.
3. Register the step name in the switch in [step.go](step.go#L55).
4. Add the matching `step_NewStepName.go` to [/build](../../build/) with `Get` and `Post` methods, and register it in [build/step_.go](../../build/step_.go).
5. Add a `newStepName_test.go` covering the constructor's defaults, its required properties, and its error cases.
6. Document the step in [STEPS.md](STEPS.md).
