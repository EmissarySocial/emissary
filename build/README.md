# build

This package contains the *builders* that Emissary passes to HTML templates when rendering pages. A builder wraps a single model object and exposes a safe, template-friendly view of it — guarding against direct access to protected fields and adding convenience queries for related records (for example, the Stream builder offers `Ancestors`, `Parent`, `Siblings`, and `Children`). This package also implements every action step available to template designers in their pipelines.

See the [project README](../README.md) for the big picture, and [model/step](../model/step/) for the data side of each pipeline step.

## Widget save pipelines

Most builders wrap a top-level record, but [Widget](builder_widget.go) scopes a pipeline to a single Widget embedded in a Stream, so that steps read and write the Widget's own data instead of its container's. Its `object()` is the StreamWidget and its `schema()` nests the Widget definition's schema beneath `data`, which is why steps address Widget properties as `data.{property}` — and why every write is still validated by the Widget's own schema.

That builder exists to run the `saveSteps` pipeline a Widget definition may declare. [executeWidgetSaveSteps](widget_save.go) runs those pipelines for every Widget in the Stream being saved, called from the `save` step before the record is written, so a Widget can derive stored values once at save time instead of recomputing them on every page view. Widgets that declare no pipeline cost nothing, and objects that are not Streams skip the pass entirely. The shipped example is [widget-markdown](../_embed/templates/widget-markdown/), which converts Markdown source into sanitized HTML with one `set-data` step.
