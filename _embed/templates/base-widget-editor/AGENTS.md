# base-widget-editor — Notes for AI Agents

See [README.md](README.md) for what the editor is and how a Template adopts it. This is the rule behind its form that looks removable and is not.

## Every location input is rendered holding its current widget IDs, never `""`

`sort-widgets` ([build/step_SortWidgets.go](../../../build/step_SortWidgets.go)) rebuilds a Stream's whole placement from the `LEFT`, `TOP`, `RIGHT`, and `BOTTOM` fields on every POST, and a field posted as `""` means that location is empty. The behavior in [hyperscript/widgetEditor._hs](hyperscript/widgetEditor._hs) rewrites those fields only after a drag, but every control in the `layout-controls` slot (page width, folder format, column split) saves through the same form on `change`, with no drag. A field rendered empty therefore turns a page-width change into a POST that deletes every widget on the page, and nothing errors: the marker says "Saved". So [widgets-list.html](widgets-list.html) renders each field from `.WidgetIDsByLocation`, and `TestWidgetEditor_LocationInputsCarryPlacement` in [service/template_widgetEditor_test.go](../../../service/template_widgetEditor_test.go) pins it.

The step also leaves a location the form did not post at all unchanged. The editor must never lean on that: its four fields are always present, and a present field is authoritative, so removing the last widget from a location still works.
