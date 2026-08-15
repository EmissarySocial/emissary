# Encrypted Messaging

Admin console page controlling who on the domain may use MLS end-to-end encrypted messaging, served at `/admin/mls` under the "External > Encrypted Messaging" sub-menu. [template.hjson](template.hjson) declares `model: Domain` (built with `build.NewDomain`, owner-only) and a schema of two fields: `mlsMode` (`NONE`, `GROUPS`, or `ALL`) and `mlsGroupIds`.

There is a single `index` action whose pipeline both renders and saves: `set-data` seeds a default, `view-html` renders [index.html](index.html), then `set-data from-form` + `save` handle the POST, finishing with `inline-save-button` and `reload-page`. The HTML is a hand-built three-column radio card form (Nobody / Select Groups / Everyone) that posts back to `/admin/mls/index`; the middle column lists `.Groups` checkboxes that auto-select the GROUPS mode when clicked, so extending the page means editing this one file rather than an hjson form definition. Extends `admin-common` for the shared menubar partial.
