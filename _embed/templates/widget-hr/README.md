# Horizontal Line

This widget (widgetId `hr`) renders a plain horizontal rule, giving editors a way to insert a visual divider between other widgets in a stream layout. As its description says, it's just a horizontal line — that's all.

[widget.hjson](widget.hjson) contains only the id, label, and description (no schema or form, so it has no settings), and [widget.html](widget.html) is a single `<hr class="margin-top margin-bottom">` element whose spacing comes from the theme's utility classes. It doubles as the minimal reference example of a widget: a folder with an hjson descriptor and a `widget.html` entry template is all the widget service needs to load and render it.
