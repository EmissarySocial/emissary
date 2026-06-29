# build

This package contains the *builders* that Emissary passes to HTML templates when rendering pages. A builder wraps a single model object and exposes a safe, template-friendly view of it — guarding against direct access to protected fields and adding convenience queries for related records (for example, the Stream builder offers `Ancestors`, `Parent`, `Siblings`, and `Children`). This package also implements every action step available to template designers in their pipelines.

See the [project README](../README.md) for the big picture, and [model/step](../model/step/) for the data side of each pipeline step.
