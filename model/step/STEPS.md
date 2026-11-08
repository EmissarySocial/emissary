# Step Reference

---

## add

Displays a modal form for a **new** model object on `GET`, then creates and saves it on `POST`. The `defaults` pipeline runs against the new object before the form is shown, which is how you preset values the user does not edit. Works with any model.

**Attributes**

| Attribute | Description |
| --- | --- |
| form | **Required.** Form definition, parsed by `benpate/form` and validated against the Template schema. A malformed `form` is a load-time error |
| defaults | Sub-pipeline applied to the new object before the form is rendered |

<br>

**Example**

```hjson
{
	do: "add"
	form: {
		type: "layout-vertical"
		children: [
			{
				type: "text"
				path: "label"
				label: "Name"
			}
		]
	}
	defaults: [
		{
			do: "set-data"
			values: {rank: "0"}
		}
	]
}
```

---

## add-event

Attaches an HX-Trigger event with the fixed value `"true"` to the response. Unlike [`trigger-event`](#trigger-event), it can fire on `GET`.

**Attributes**

| Attribute | Description |
| --- | --- |
| event | **Required.** Name of the client event to fire |
| method | `get` fires on GET only, `post` on POST only. Defaults to `post` |

<br>

**Example**

```hjson
{
	do: "add-event"
	event: "closeModal"
	method: "post"
}
```

---

## add-stream

Creates a new Stream. `style` decides how the user picks a Template: `chooser` shows a picker, `modal` opens a create dialog, `inline` embeds the widget in the page. Setting `template` skips the choice entirely, which is what makes this step usable inside a larger pipeline.

**Attributes**

| Attribute | Description |
| --- | --- |
| style | `chooser`, `modal`, or `inline`. Defaults to `chooser` |
| title | Heading on the create modal. Defaults to `+ Add a Page` |
| location | `top`, `child`, or `outbox`. Anything else falls back to `top`. The struct comment in [addStream.go](addStream.go) claims `child`, but `val.Enum` returns the first enum value |
| state | Initial state of the new Stream. Defaults to `default`, and must be defined in the Template's `states` |
| template | Exact Template to use. Skips the picker entirely |
| roles | Acceptable template roles when `template` is empty. An empty list allows every Template valid for this container |
| with&#8209;data | Map of values (each a template) preset on the new Stream |
| redirect&#8209;to | Template URL to send the user to after creation. When empty, the pipeline continues normally |

<br>

**Example**

```hjson
{
	do: "add-stream"
	style: "inline"
	location: "child"
	template: "article"
	with-data: {label: "Untitled"}
	redirect-to: "/{{.StreamID}}/edit"
}
```

The `redirect-to` property navigates exactly like the [`redirect-to`](#redirect-to) step, including its safety check and its off-site handling, but its template is evaluated against the **new** Stream so that `{{.StreamID}}` names the record that was just created.

Once the new Stream's `create` action has saved it, `add-stream` creates the Streams that its Template lists in `defaultChildren`, beneath it and in the same transaction. A `create` that halts before its `save` gets none. The children get no default children of their own.

---

## as-confirmation

Shows a confirmation dialog on `GET`. The rest of the pipeline runs only after the user confirms.

**Attributes**

| Attribute | Description |
| --- | --- |
| title | **Required.** Dialog heading |
| message | **Required.** Body text explaining what is about to happen |
| icon | Icon name displayed beside the title |
| submit | Label on the confirm button. Defaults to `Continue` |

<br>

**Example**

```hjson
{
	do: "as-confirmation"
	icon: "warning"
	title: "Publish now?"
	message: "This will notify your followers."
	submit: "Publish"
}
```

---

## as-modal

Wraps the sub-pipeline's output in a modal window. On a partial (HTMX) request — the common case — only the modal is built, and it is retargeted into the page's `<aside>`. A full-page request needs `background-url` or `background` to know what page to draw underneath; without either, opening the route directly is a `400`. The sub-pipeline's `POST` fires a `closeModal` event when it finishes.

`background-url` draws another object's page behind the modal, such as the profile a photo's gallery sits on, which `background` cannot, because it names an action of the same builder. It is a local path, `/@{user}` or `/{stream}`, each with an optional `/{action}` and a query string, and it is built with the visitor's own session, so it checks permissions as if the visitor had opened it. When it cannot be drawn (the visitor cannot see it, it redirects, or it names anything else), `background` is drawn instead. A page drawn as a background cannot open a modal of its own.

`replace-url` gives the modal a shareable address. On a partial request, the address bar shows it while the modal is open (`HX-Replace-Url`), and the global theme's `modal-url.js` puts the page's own address back when the modal closes. A modal opened directly over a `background-url` carries that URL in `data-return-url`, which closing the modal restores.

`head` steps render into the full page's `<head>`, so a shared link previews the modal and not its background. When their output holds a `<title>`, it replaces the page's own.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline rendered inside the modal |
| options | Modal options passed to the modal wrapper (size, dismissal behavior, …), each evaluated as a template |
| background | Name of the action to render behind the modal on a full-page request |
| background-url | Local URL of a page to render behind the modal on a full-page request, evaluated as a template. Tried before `background` |
| replace-url | Local URL to show in the address bar while the modal is open, evaluated as a template |
| head | Sub-pipeline rendered into the full page's `<head>` |

<br>

**Example**

```hjson
{
	do: "as-modal"
	background: "view"
	options: ["class:large"]
	steps: [
		{
			do: "view-html"
			file: "settings"
		}
	]
}
```

A photo whose own URL opens over its gallery, on its owner's profile:

```hjson
{
	do: "as-modal"
	options: ["class:large"]
	background-url: "/@{{(.Parent `view`).ParentID}}?tab={{.ParentID}}"
	background: "page"
	replace-url: "/{{.StreamID}}"
	head: [{do:"view-html", file:"head"}]
	steps: [{do:"view-html", file:"modal"}]
}
```

---

## as-tooltip

The same wrapper as [`as-modal`](#as-modal), rendered as a tooltip instead.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline rendered inside the tooltip |

<br>

**Example**

```hjson
{
	do: "as-tooltip"
	steps: [
		{
			do: "view-html"
			file: "preview"
		}
	]
}
```

---

## cache-url

Adds `ETag` and `Cache-Control` headers to a `GET` response, and short-circuits with `304 Not Modified` when the browser's `If-None-Match` matches the object's ETag. Does nothing on `POST`.

This step does nothing at all for a signed-in User or a guest Identity — no `304`, and no headers. The ETag tracks the object's revision, not the viewer, so it cannot distinguish a public page from the privileged rendering those callers were served, and `public` must never label such a response. They fall through to the `Cache-Control` that [`view-html`](#view-html) sets, which defaults to `private, no-cache`.

**Attributes**

| Attribute | Description |
| --- | --- |
| private | Emit `private` instead of `public` in the `Cache-Control` header. Defaults to `false` |
| max&#8209;age | Cache lifetime in seconds. Defaults to `3600` |

<br>

**Example**

```hjson
{
	do: "cache-url"
	max-age: 86400
}
```

---

## delete

Shows a confirmation dialog on `GET` and deletes the object on `POST`.

**Attributes**

| Attribute | Description |
| --- | --- |
| title | Template heading. Defaults to `Delete '{{.Label}}'?`. Max 128 characters, enforced at load time |
| message | Template body text. Defaults to `Are you sure you want to delete {{.Label}}? There is NO UNDO.` Max 512 characters |
| submit | Label on the delete button. Defaults to `Delete`. Max 32 characters |
| cancel | Label on the cancel button. Defaults to `Cancel` |
| method | `get`, `post`, or `both`. Defaults to `both` |

<br>

**Example**

```hjson
{
	do: "delete"
	title: "Delete this post?"
	message: "This cannot be undone."
}
```

---

## delete-archive

Removes a cached Stream archive on `POST`. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| token | Names the archive variant to delete. An empty token addresses the default archive |

<br>

**Example**

```hjson
{
	do: "delete-archive"
	token: "full"
}
```

---

## delete-attachments

Removes attachments from the current object. The `attachmentId` query parameter, when present, narrows the selection to that one attachment, and a value that is not an AttachmentID halts with a 400 rather than widening the delete. With none of the three selecting attributes set, the step does nothing.

**Attributes**

| Attribute | Description |
| --- | --- |
| all | Delete every attachment on this object |
| field | Delete only the attachment named by this property, which holds its AttachmentID or its URL (as `set-thumbnail` writes into `iconUrl`), and clear the property |
| category | Delete every attachment in this category |
| minimum | Fewest attachments the category may be left holding. A delete that would leave fewer halts with a 400 and deletes nothing. Ignored with `field`. Defaults to `0` |

<br>

**Example**

```hjson
{
	do: "delete-attachments"
	category: "cover"
}
```

---

## dump

Writes the object — or the evaluated `value` — to the server console with `spew.Dump`, then continues. Runs in both phases. Debugging only.

**Attributes**

| Attribute | Description |
| --- | --- |
| value | Template expression to dump. When empty, dumps the whole object |

<br>

**Example**

```hjson
{do: "dump"}
```

---

## edit

Modal form for an **existing** object: renders on `GET`, saves on `POST`. Unlike [`add`](#add), an omitted `form` is allowed — the object's schema then drives the whole editor.

**Attributes**

| Attribute | Description |
| --- | --- |
| form | Form definition, validated against the Template schema at load time |
| options | Modal options, each evaluated as a template |

<br>

**Example**

```hjson
{
	do: "edit"
	form: {
		type: "layout-vertical"
		children: [
			{
				type: "text"
				path: "label"
				label: "Label"
			}
			{
				type: "textarea"
				path: "summary"
				label: "Summary"
			}
		]
	}
}
```

---

## edit-attachments

Draws the multi-file zone that an `html-remote` form element loads, and uploads, deletes, and sorts the files in one category in place. The element is a plain loader: it draws an empty shell that fills itself from the action this step runs on, which its `url` names (`/{{.ID}}/photos` on an edit form), so the step holds the category's whole configuration, including the field name the files post under. A `create` action runs the same action inline with `include`, on the POST alone (`method: "post"`) because the element's own request draws the zone, so the files a new object's form posts are stored under its ID.

On GET it writes the zone for a saved object, the grid the theme's behavior and styles find: the tile row with the limits as `data-` attributes and the multipart encoding its uploads need, then the plus that adds one file, the hidden single-file input named by `fieldname`, and the status line. With `?readonly=true`, which the widget's read-only view sends, it writes the tiles alone with no controls and the row marked `data-readonly`, so the browser behavior leaves it be. On POST it applies one change and writes the row alone again, so the plus and the input stay in place while the row swaps. `?attachmentId=` deletes that one file, and is refused with a 400 when the ID is not one of the category's files on this object, or with an inline error when the category would fall below `minimum`. Otherwise the files posted under `fieldname` are sniffed against `accept-type`, and any `{fieldname}.order` value naming a file the signed-in User staged through this step is adopted into the category, with the stored type checked against `accept-type`. The posted and staged files are stored and adopted after the category's highest rank, and then the order values re-rank the category's files in the order posted. This is how a create form's files reach the new object: the widget stages each one as it is dropped, and the Save posts their IDs. The step never deletes more than one file, and reads the files and the order from the body alone.

The limits apply differently to the two kinds of POST. On a new object, the POST is the create form's Save: the files past `maximum` are dropped, the posted ones kept before the staged ones and each in the order posted, and a dropped staged file stays staged until the purge removes it; the Save is then refused inline when the category would hold fewer than `minimum`. On a saved object, the POST is one change from the widget: an upload is refused as a batch when it would pass `maximum`, while a sort, or an upload that leaves the category short of `minimum`, goes through, so a category outside its limits can still be sorted and filled.

**Before the object exists.** A new object has no route of its own, so its create form's element reaches this step through the parent's `add`, with the action named in the query: `url: "/{{.ParentID.Hex}}/add?action=photos"`, plus `&templateId={{.TemplateID}}` when the add's step names no template. `add-stream` mints the child as it does for the form, runs the named action on it, and answers with that action's result alone. The action must be one the child's `create` action runs through `include`, at any depth, or `add-stream` refuses the request with a 400, because the Save finishes that action's work and any other action could save a child that `create` never shaped; the action's own roles apply, with the signed-in User as the child's author. The step tells that request from the form's Save by where it sits: when the current action holds the step, the request is the widget's own, and when another action reached it through `include`, the request is the Save. On a new object the widget's GET draws an empty row wired to post each file back through the same route, then the plus, the file input, and the status line, and the Save's GET draws nothing. The widget's POST stores the one file posted under `fieldname` in the `upload-staging` category of the signed-in User, sniffed against `accept-type` so the form hears a wrong type at once, and answers with its tile alone, which the widget appends to its row; the tile's `{fieldname}.order` input carries the staged file's ID, so the Save posts IDs and no file. `?attachmentId=` deletes that one staged file and writes nothing, so the tile removes itself; an ID that is not one of the User's own staged files is refused with a 400, a body with no file or more than one with a 400, and a request with no signed-in User with a 401. Each of the widget's answers on a new object halts the action, so a `save` or any other step after this one in the widget's action runs only for a saved object; the Save through `include` continues as usual. A staged file that no Save adopts within a day is deleted by the `PurgeStagedUploads` task.

**Attributes**

| Attribute | Description |
| --- | --- |
| category | **Required.** Category of the files this step manages |
| fieldname | Form field holding the files, and the prefix of the `.order` field holding their order. Defaults to the category |
| accept&#8209;type | MIME type filter, e.g. `image/*`, enforced by sniffing each file |
| minimum | Fewest files the category may hold. A delete that would leave fewer, or a create form's Save that would, is refused. Defaults to `0`, and cannot exceed `maximum` |
| maximum | Most files the category may hold. An upload to a saved object that would pass it is refused whole; a create form's Save keeps the files that fit. Defaults to `1`, and can never be lower |
| rules.height | Fixed height for all downloads |
| rules.width | Fixed width for all downloads |
| rules.types | Allowed extensions. The first value is the default |

<br>

**Example**

```hjson
{
	do: "edit-attachments"
	category: "photo"
	fieldname: "photos"
	accept-type: "image/*"
	minimum: 1
	maximum: 64
	rules: {
		types: ["webp"]
	}
}
```

---

## edit-connection

Admin modal that configures a connection to an external provider, identified by the `providerId` query parameter. Requires the `Domain` model, a Domain builder, and a provider that implements `ManualProvider`.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "edit-connection"}
```

---

## edit-content

The long-form content editor. `max-length` is expressed in **kilobytes** for ergonomics and converted to runes internally; it defaults to 64 KB and is clamped to the 1 MB storage ceiling, so an over-large value is capped and logged rather than rejected.

**Attributes**

| Attribute | Description |
| --- | --- |
| format | **Required.** `EDITORJS`, `HTML`, `MARKDOWN`, or `TEXT`. Enforced at load time by schema validation, which makes the `editorjs` fallback in [editContent.go](editContent.go) unreachable |
| file | Template file used to render the editor. Defaults to the action ID |
| field | Schema path to edit. Defaults to `content` |
| max&#8209;length | Maximum content size in KB. Defaults to `64`, clamped to `1024` |
| require&#8209;content | Halt with an error when the submission is empty. Defaults to `false` |

<br>

**Example**

```hjson
{
	do: "edit-content"
	format: "EDITORJS"
	max-length: 128
	require-content: true
}
```

---

## edit-registration

Admin modal that picks the domain's new-user signup method. Requires the `Domain` model and a Domain builder.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "edit-registration"}
```

---

## edit-table

Renders an editable table for an array inside the object's schema. Form fields are relative to `path`, not to the schema root.

**Attributes**

| Attribute | Description |
| --- | --- |
| form | **Required.** Form definition for one row. A malformed `form` is a load-time error |
| path | **Required.** Schema path of the array being edited |

<br>

**Example**

```hjson
{
	do: "edit-table"
	path: "links"
	form: {
		type: "layout-vertical"
		children: [
			{
				type: "text"
				path: "name"
				label: "Name"
			}
			{
				type: "text"
				path: "url"
				label: "URL"
			}
		]
	}
}
```

---

## edit-template

Lets a user re-assign the object's Template. Only the three named boolean properties are accepted; any other property is a load-time error.

**Attributes**

| Attribute | Description |
| --- | --- |
| title | Modal heading |
| templateId | Offer the primary Template picker |
| inboxTemplate | Offer the inbox Template picker |
| outboxTemplate | Offer the outbox Template picker |

<br>

**Example**

```hjson
{
	do: "edit-template"
	title: "Choose a Theme"
	templateId: true
}
```

---

## edit-widget

Opens the settings form for the Widget named by the request, using that Widget definition's own schema and form. Renders the form on `GET`, saves the Widget's configuration data on `POST`.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "edit-widget"}
```

---

## forward-to

Sends the visitor somewhere else because they are finished with this page — after a save, a delete, or a checkout. Inside an htmx request this is the `Hx-Redirect` header, which moves the whole browser and closes any open modal; outside one it is an HTTP `303 See Other`, because `Hx-Redirect` means nothing to a browser following a plain link. Either way the visitor's whole document navigates.

Use [`redirect-to`](#redirect-to) instead when the *content* lives at another URL and an htmx caller should swap the new fragment into the page rather than reload it.

The target is validated with `uri.IsSafeRedirectURL`, so a `javascript:` or protocol-relative URL built from remote data is rejected rather than followed.

**Attributes**

| Attribute | Description |
| --- | --- |
| url | **Required.** Template URL to forward to |
| method | `get`, `post`, or `both`. Defaults to `post`; an unrecognized value fails at Template load. Think before widening this one — a pipeline that renders a form on `GET` keeps running past the form, so a `forward-to` set to `both` navigates away before the visitor ever sees it. |

<br>

**Example**

```hjson
{
	do: "forward-to"
	url: "/{{.StreamID}}"
}
```

---

## get-archive

Downloads a Stream archive on `GET`, queueing a `MakeStreamArchive` task first if the file is not already cached. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| token | Names this archive variant, so one Stream can carry several archives with different settings |
| depth | How many levels of child Streams to include |
| json | Include JSON-LD alongside the HTML |
| attachments | Include attachment files |
| metadata | List of `translate` pipelines that generate archive metadata. Parsed and validated at load time |

<br>

**Example**

```hjson
{
	do: "get-archive"
	token: "full"
	depth: 10
	json: true
	attachments: true
}
```

---

## halt

Stops the pipeline immediately, in both phases.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "halt"}
```

---

## if

Evaluates `condition` as a template and runs either `then` or `else`. Both branches' state and role requirements are rolled into this step's own, so the Template must define everything either branch needs.

**Attributes**

| Attribute | Description |
| --- | --- |
| condition | **Required.** Template expression. An empty condition evaluates false and always takes the `else` branch |
| then | Sub-pipeline run when the condition is true |
| else | Sub-pipeline run when the condition is false |

<br>

**Example**

```hjson
{
	do: "if"
	condition: "{{.IsAuthenticated}}"
	then: [
		{
			do: "view-html"
			file: "detail"
		}
	]
	else: [
		{
			do: "view-html"
			file: "teaser"
		}
	]
}
```

---

## include

Looks up another action on the current Template and runs its pipeline inline, in the current phase. The included steps run on the caller's builder, so `.Action` in any file they render still names the caller. A `view-html` or `view-css` in the included action renders its own action's file unless it names another.

**Attributes**

| Attribute | Description |
| --- | --- |
| action | **Required.** Name of the action to run. An unknown name halts with a `400` at request time |
| method | `get`, `post`, or `both`. Defaults to `both`. A form's GET runs on past the form, so an include that only saves needs `post`, or it runs again when the form opens |

<br>

**Example**

```hjson
{
	do: "include"
	action: "view"
}
```

---

## inline-error

Writes an error message into a form without navigating away. Typically used inside a [`save`](#save) step's `on-error` pipeline.

**Attributes**

| Attribute | Description |
| --- | --- |
| message | **Required.** Template message |

<br>

**Example**

```hjson
{
	do: "inline-error"
	message: "That name is already taken."
}
```

---

## inline-save-button

Re-renders a form's save button, which is how a form signals its own state — saving, saved, disabled — without a page reload.

**Attributes**

| Attribute | Description |
| --- | --- |
| id | Template element ID. Defaults to `inline-save-button` |
| class | CSS class on the button. Defaults to `primary` |
| label | Template button label. Defaults to `Save Changes` |

<br>

**Example**

```hjson
{
	do: "inline-save-button"
	label: "Publish"
	class: "primary"
}
```

---

## inline-success

Writes a success message into a form without navigating away.

**Attributes**

| Attribute | Description |
| --- | --- |
| message | **Required.** Template message |
| href | Template URL that turns the message into a link |

<br>

**Example**

```hjson
{
	do: "inline-success"
	message: "Saved."
	href: "/{{.StreamID}}"
}
```

---

## make-archive

Queues a `MakeStreamArchive` background task on `POST`. Requires the `Stream` model. Use [`get-archive`](#get-archive) to serve the result.

**Attributes**

| Attribute | Description |
| --- | --- |
| token | Names this archive variant |
| depth | How many levels of child Streams to include |
| json | Include JSON-LD alongside the HTML |
| attachments | Include attachment files |
| metadata | List of `translate` pipelines that generate archive metadata. Parsed and validated at load time |

<br>

**Example**

```hjson
{
	do: "make-archive"
	token: "full"
	depth: 10
	json: true
	attachments: true
}
```

---

## mark-folder-read

Marks every unread item in the folder named by the `folderId` query parameter as read, on `POST`. Requires an authenticated user. A missing or invalid `folderId` — the all-folders "News Feed" view — is a no-op, not an error.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "mark-folder-read"}
```

---

## mark-notifications-read

Marks all of the current User's notifications as read.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "mark-notifications-read"}
```

---

## process-content

Reformats a Stream's content: converts between formats, optionally strips HTML, and optionally linkifies bare URLs. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| format | `MARKDOWN`, `EDITORJS`, or `HTML`. Any other value is a load-time error; an empty value leaves the format alone |
| remove&#8209;html | Strip HTML from the source content |
| add&#8209;links | Convert bare URLs into links |
| add&#8209;tags | **Deprecated and ignored.** Logs a warning when the Template loads |
| tag&#8209;path | **Deprecated and ignored.** Logs a warning when the Template loads |

<br>

**Example**

```hjson
{
	do: "process-content"
	format: "MARKDOWN"
	add-links: true
}
```

---

## process-tags

**Deprecated.** Hashtags are extracted automatically when a Stream or User is saved, driven by the Template's `tagPaths`, so this step does nothing and logs a warning when the Template loads. It is retained only so older Templates keep loading — remove it.

**Attributes**

| Attribute | Description |
| --- | --- |
| paths | Comma-separated list of schema paths. Parsed, then ignored |

---

## promote-draft

Copies a StreamDraft's content over its live Stream and moves the Stream into `state`. Requires the `Stream` model.

Eleven properties are copied: `url`, `token`, `label`, `summary`, `content`, `iconUrl`, `icon`, `widgets`, `data`, `attributedTo`, and `inReplyTo`. Everything else on the Stream — sharing, tree position, publish dates, response counts, journal — is left alone, because a draft's copy of those goes stale the moment anything changes them outside the draft.

`omit` names properties to leave alone as well. It exists for a Stream whose value is written by something other than the author: a draft is a snapshot taken when it was created, so promoting one reverts every change made since. The case this was built for is [article-remote](../../_embed/templates/stream-article-remote/), whose body belongs to a `StreamSource` — and where the revert is permanent, because the source's `ContentHash` still matches what it last wrote, so the next synchronization stops before fetching and **Sync Now** does nothing.

Only the eleven names above are accepted, and nested paths (`data.tags`) are not. Anything else fails the Template at load, because a name that is merely ignored would leave the property copied — the exact mistake `omit` is there to prevent, and one that reports nothing when it happens.

**Attributes**

| Attribute | Description |
| --- | --- |
| state | State to move into. Defaults to `published`, and must be defined in the Template's `states` |
| omit | Names of properties NOT to copy from the draft, leaving the live Stream's values in place |

<br>

**Example**

```hjson
{
	do: "promote-draft"
	state: "published"
}
```

Promote everything except the body, which some other process owns:

```hjson
{
	do: "promote-draft"
	omit: ["content"]
}
```

---

## read-form

Reads named fields from a form POST into the Builder's temporary data scope, where later steps read them with `.GetString`. Visitor input never reaches the object being built — for a Stream, that is the page record itself.

Values come from the request **body** only. Unlike most steps, `read-form` ignores the URL query string, so a crafted link cannot supply or append to a field.

Does nothing on `GET`.

**Attributes**

| Attribute | Description |
| --- | --- |
| schema | **Required.** A JSON-Schema object describing every field this step accepts |

<br>

The schema is an allowlist, not a suggestion: a field the template did not declare is never read, and a declared field that fails validation halts the pipeline. A value longer than its `maxLength` is **rejected**, not shortened — the same rule [`edit-content`](#edit-content) uses, and for the same reason. `maxLength` counts characters, not bytes.

**Example**

```hjson
{
	do: "read-form"
	schema: {
		type: "object"
		properties: {
			name: {type:"string", maxLength:128, required:true}
			email: {type:"string", format:"email", maxLength:255, required:true}
			message: {type:"string", maxLength:4096, required:true}
		}
	}
}
```

---

## redirect-to

Sends the visitor to a new URL because the content they asked for lives there. This is a real HTTP redirect, which means an htmx caller follows it inside its own request and swaps the result into the page — a same-site `redirect-to` does not reload the browser, and that is usually what you want for a `view` action that normalizes its own URL.

An **off-site** target is the exception, and it is handled automatically: htmx cannot read a cross-origin response, so the step emits `Hx-Redirect` instead and lets htmx navigate the whole browser. A Redirect stream therefore behaves the same whether a visitor opens it through a plain `<a href>` or through an `hx-get`, with no branch in the Template.

Use [`forward-to`](#forward-to) instead when the visitor is finished with this page rather than being sent to the rest of it.

The target is validated with `uri.IsSafeRedirectURL`, so a `javascript:` or protocol-relative URL built from remote data is rejected rather than followed.

**Attributes**

| Attribute | Description |
| --- | --- |
| url | **Required.** Template URL to redirect to |
| method | `get`, `post`, or `both`. Defaults to `both`; an unrecognized value fails at Template load |
| status | HTTP status code. Defaults to `307`. Ignored on the off-site path, where htmx acts on the header instead |

<br>

**Example**

```hjson
{
	do: "redirect-to"
	url: "/signin"
	status: 302
}
```

---

## refresh-page

Fires the `closeModal` and `refreshPage` client events on `POST`. Use this after a modal save, when the underlying page needs to pick up the change.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "refresh-page"}
```

---

## reload-page

Forces a full browser reload with `HX-Refresh: true` on `POST`. Heavier than [`refresh-page`](#refresh-page); use it when the page chrome itself has changed.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "reload-page"}
```

---

## remove-event

Strips an HX-Trigger event that an earlier step added. Runs in both phases.

**Attributes**

| Attribute | Description |
| --- | --- |
| event | **Required.** Name of the event to remove |

<br>

**Example**

```hjson
{
	do: "remove-event"
	event: "refreshPage"
}
```

---

## require

Evaluates `condition` as a template, and halts the pipeline with a `400 Bad Request` error unless it renders exactly `true`. Runs in both phases. It reads the condition the same way [`if`](#if) does, so `{do: "require", condition: "X"}` refuses exactly the requests that `{do: "if", condition: "X", else: [{do: "halt"}]}` would stop, with an error in place of a silent halt. A condition that fails to render halts with a server error instead.

**Attributes**

| Attribute | Description |
| --- | --- |
| condition | **Required.** Template expression that must render `true`. An empty condition is a load-time error |

<br>

**Example**

```hjson
{
	do: "require"
	condition: "{{eq (.Data `layout`) `notes`}}"
}
```

---

## require-password

Interposes a password-confirmation dialog before a sensitive action. The `title` and `message` defaults are inherited from the delete dialog, so set both explicitly.

**Attributes**

| Attribute | Description |
| --- | --- |
| title | Template heading. Defaults to `Delete '{{.Label}}'?`. Max 128 characters, enforced at load time |
| message | Template body text. Defaults to `Are you sure you want to delete {{.Label}}? There is NO UNDO.` Max 256 characters |
| submit | Label on the confirm button. Defaults to `Confirm`. Max 32 characters |
| submitClass | CSS class on the confirm button. Defaults to `warning` |
| cancel | Label on the cancel button. Defaults to `Cancel`. Max 32 characters |

<br>

**Example**

```hjson
{
	do: "require-password"
	title: "Confirm Your Identity"
	message: "Enter your password to continue."
	submit: "Verify"
}
```

---

## save

Saves the current object to the database. The optional `on-error` pipeline runs when the save fails, which is how a form reports a validation failure inline instead of blowing up the request.

**Attributes**

| Attribute | Description |
| --- | --- |
| comment | Template audit comment stored with the save. Max 1024 characters, enforced at load time |
| method | `get`, `post`, or `both`. Defaults to `post` |
| on&#8209;error | Sub-pipeline run when the save fails |

<br>

**Example**

```hjson
{
	do: "save"
	comment: "Updated by {{.UserName}}"
	on-error: [
		{
			do: "inline-error"
			message: "Could not save your changes."
		}
	]
}
```

---

## save-and-publish

Saves the object, stamps its publish date, moves it into `state`, and optionally fans it out to the User's outbox and syndication targets.

**Attributes**

| Attribute | Description |
| --- | --- |
| state | State to move into. Defaults to `published`, and must be defined in the Template's `states` |
| outbox | Also send updates to this User's outbox |
| republish | Republish this Stream to its syndication targets |

<br>

**Example**

```hjson
{
	do: "save-and-publish"
	state: "published"
	outbox: true
}
```

---

## schedule-delete

Sets a future delete date on a Stream. All four values are templates, so the delay can be computed from the object. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| days | Template number of days to wait |
| hours | Template number of hours to wait |
| minutes | Template number of minutes to wait |
| seconds | Template number of seconds to wait |

At least one of the four is needed for the step to schedule anything.

<br>

**Example**

```hjson
{
	do: "schedule-delete"
	days: "30"
}
```

---

## search-index

Synchronizes the object's search record on `POST`. When `if` evaluates false the record is marked deleted rather than updated, which is how unpublished or private content drops out of search.

**Attributes**

| Attribute | Description |
| --- | --- |
| if | Template condition. Defaults to `true` |

<br>

**Example**

```hjson
{
	do: "search-index"
	if: "{{.IsPublished}}"
}
```

---

## send-email

Sends one of the domain's named emails. The email definition names its own recipient, subject, headers, and the model object its data describes; this step only names the email and supplies the values it interpolates.

`welcome` and `password-reset` are special: each mints a password-reset credential before it sends, so they route through the User service and take no `values`.

**Attributes**

| Attribute | Description |
| --- | --- |
| email | **Required.** Name of the email definition to send |
| values | Key/value pairs passed into the email's data. Each value is compiled as a template. |

<br>

Every key that the email's `to` and `headers` templates interpolate must appear in `values` — those templates reject a missing key outright, and the check runs when Templates are loaded, not when the email is sent.

**Example**

```hjson
{
	do: "send-email"
	email: "stream-contact-form"
	values: {
		To: "{{.Data `emailAddress`}}"
		ReplyEmail: "{{.GetString `email`}}"
	}
}
```

---

## set-args

Sets values into the **Builder's** render data rather than the object, which is how state passes between steps and into templates. Runs in both phases.

**Attributes**

Every property except `do` is treated as a key, and every value is compiled as a template.

| Attribute | Description |
| --- | --- |
| *arbitrary* | Each key/value pair is written into the Builder's render data |

<br>

**Example**

```hjson
{
	do: "set-args"
	tab: "settings"
	returnUrl: "{{.Permalink}}"
}
```

---

## set-circle-sharing

The richer sharing dialog, sharing with named Circles and purchased Products. Requires the `Stream` model.

It offers three choices, in this order: Everyone, Not Shared, and These Circles Only. Not Shared grants the role to no Group, so only the author keeps access, and a Stream that is neither public nor shared with Circles opens with it checked. A form saved with nothing checked is shared with Everyone.

**Attributes**

| Attribute | Description |
| --- | --- |
| role | **Required.** Role granted to the audience, enforced at load time. Must be defined in the Template's `roles` |
| method | `get`, `post`, or `both`. Defaults to `both` |
| title | Dialog heading. Defaults to `Sharing Settings` |
| message | Body text. Defaults to `Public Settings` |
| button | Label on the submit button. Defaults to `Save Changes` |

A menu that names the current sharing should read the Stream builder's `CircleSharingStatus`, which uses these terms (Everyone, a Circle's name, a count of Circles, or "Not Shared"), rather than `SharingStatus`, which mirrors `set-simple-sharing`.

<br>

**Example**

```hjson
{
	do: "set-circle-sharing"
	role: "viewer"
	title: "Share With…"
	button: "Update Sharing"
}
```

---

## set-data

Writes values into the current object. The four sources are applied independently, so a single step can pull IDs off the URL, accept a whitelist of form fields, force some values from templates, and backfill others only when empty.

**Attributes**

| Attribute | Description |
| --- | --- |
| from&#8209;url | Schema paths to populate from query-string parameters |
| from&#8209;form | Schema paths to populate from the form POST |
| values | Map of schema path to template, written unconditionally |
| defaults | Map of schema path to value, written only when the target is currently empty |

<br>

**Example**

```hjson
{
	do: "set-data"
	from-url: ["parentId"]
	from-form: ["label", "summary"]
	values: {
		token: "{{.StreamID}}"
	}
	defaults: {
		rank: 0
	}
}
```

---

## set-header

Sets a raw HTTP response header.

**Attributes**

| Attribute | Description |
| --- | --- |
| name | **Required.** Header name |
| value | **Required.** Template header value |
| method | `get`, `post`, or `both`. Defaults to `both` |

<br>

**Example**

```hjson
{
	do: "set-header"
	name: "X-Robots-Tag"
	value: "noindex"
}
```

---

## set-password

Reads `new_password` — and optionally `confirm_password` — from the form POST and updates the User's password. An empty `new_password` is a silent no-op; a mismatched confirmation halts with an inline validation error.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "set-password"}
```

---

## set-privileges

Product and circle privilege editor for a Stream, driven by the author's merchant accounts. Displays an empty-state prompt when the author has no merchant account configured. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| title | Dialog heading. Defaults to `Product Settings` |

<br>

**Example**

```hjson
{
	do: "set-privileges"
	title: "Paid Access"
}
```

---

## set-query-param

Rewrites the **request's** query string in place, so later steps and templates see the new values. Runs in both phases.

**Attributes**

Every property except `do` is treated as a key, and every value is compiled as a template.

| Attribute | Description |
| --- | --- |
| *arbitrary* | Each key/value pair is set on the request's query string |

<br>

**Example**

```hjson
{
	do: "set-query-param"
	folderId: "{{.FolderID}}"
}
```

---

## set-response

Creates, updates, or removes the authenticated User's Response — like, dislike, announce — to the current object, based on the posted transaction. Requires an authenticated user.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "set-response"}
```

---

## set-sharing

Forces the Stream's sharing settings for a role to a fixed magic Group on `POST`, replacing whatever was there — no dialog, no user input. Does nothing on `GET`. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| role | **Required.** Role granted to the audience, enforced at load time. Must be defined in the Template's `roles` |
| group | **Required.** Magic Group to share with: `anonymous`, `authenticated`, or `owner`. Enforced at load time |

<br>

**Example**

```hjson
{
	do: "set-sharing"
	role: "viewer"
	group: "anonymous"
}
```

---

## set-simple-sharing

The plain public/private sharing dialog. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| role | **Required.** Role granted to the audience, enforced at load time. Must be defined in the Template's `roles` |
| title | Dialog heading. Defaults to `Sharing Settings` |
| message | Body text. Empty by default |

<br>

**Example**

```hjson
{
	do: "set-simple-sharing"
	role: "viewer"
	title: "Who can see this?"
}
```

---

## set-state

Moves the object into a new state.

**Attributes**

| Attribute | Description |
| --- | --- |
| state | **Required.** State to move into, enforced at load time. Must be defined in the Template's `states` |

<br>

**Example**

```hjson
{
	do: "set-state"
	state: "published"
}
```

---

## set-thumbnail

Scans the object's attachments in rank order and writes the first image into `path` — as a bare AttachmentID for `User` objects, and as a permalink URL for everything else. When no image is found, `path` is cleared.

**Attributes**

| Attribute | Description |
| --- | --- |
| path | **Required.** Schema path that receives the thumbnail. An invalid path halts the pipeline |
| category | Consider only attachments in this category. Defaults to every category |

<br>

**Example**

```hjson
{
	do: "set-thumbnail"
	path: "imageUrl"
}
```

---

## setup-complete

Ends the startup wizard, moving the Domain out of its `STARTUP` state and into production, on `POST`. Requires the `Domain` model **and** a Template whose `templateRole` is `admin` — ending setup opens the whole Domain to the public, so `Domain` alone would still admit a public-facing Template. A Domain that is already live is left alone, so a double submit does not fail the surrounding action.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "setup-complete"}
```

---

## share-reply-policy

Copies the current Stream's `replyPolicy` onto every Stream in its User's outbox that uses `template`, on `POST`. It is how a collection that does not hold its items passes its policy on, such as a Notes collection passing it to the User's notes. A collection's own children get its policy through the Template's `shareWithChildren` instead. Requires the `Stream` model. It does nothing for a Stream that is not in an outbox, and it halts when the signed-in User does not own that outbox. Only the Streams whose policy differs are saved, and nothing is republished.

**Attributes**

| Attribute | Description |
| --- | --- |
| template | **Required.** The TemplateID of the outbox Streams to update |

<br>

**Example**

```hjson
{do: "share-reply-policy", template: "outbox-message"}
```

---

## sleep

Pauses the pipeline in both phases. Debugging only.

**Attributes**

| Attribute | Description |
| --- | --- |
| duration | Time to sleep, in **milliseconds**. Defaults to `0` |

<br>

**Example**

```hjson
{
	do: "sleep"
	duration: 2000
}
```

---

## sort

Re-ranks a set of records from a form POST containing a `keys` array of ObjectIDs, in the order posted, and saves only the records whose rank changed. Responds with `HX-Reswap: none` so the page does not reload.

With `direction: "asc"`, the posted records are numbered 1, 2, 3, and so on. With `direction: "desc"`, they are numbered down from the highest rank among them, so the first record keeps the top rank. That lets a list shown highest-rank-first post only its loaded pages, as an infinite scroll does, without moving the records below them.

**Attributes**

| Attribute | Description |
| --- | --- |
| model | Named model service to sort. Defaults to the builder's own service |
| keys | Property matched against each posted key. Defaults to `_id` |
| values | Property that receives the new rank. Defaults to `rank` |
| direction | `asc` (the default) or `desc`, matching the order the list is shown in |
| message | Audit comment recorded with each save |

<br>

**Example**

```hjson
{
	do: "sort"
	model: "Folder"
	message: "Reordered folders"
}
```

---

## sort-widgets

Re-ranks a Stream's widgets and re-assigns them to the Template's widget locations from a form POST. Stream builders only.

**Attributes**

*No attributes.*

<br>

**Example**

```hjson
{do: "sort-widgets"}
```

---

## startup-create-streams

Seeds an empty Domain with the Streams its Theme declares in `startupStreams`, on `POST`. Requires the `Domain` model **and** a Template whose `templateRole` is `admin`. Every precondition failure is a silent no-op, so a Domain that is already live does not fail the surrounding action.

The form POST chooses *which* Streams to create. Every value posted under the `tokens` field is matched against the `token` of each entry in the Theme's `startupStreams`, and an entry whose token was not posted is skipped. The Theme stays the authority on what can be created — a token it does not define matches nothing — so posting no tokens creates no Streams, which is a legitimate choice rather than an error.

**Attributes**

*No attributes.* The selection arrives with the form POST:

| Form Field | Description |
| --- | --- |
| tokens | Repeated field naming the `startupStreams` entries to create. Anything the Theme does not define is ignored |

<br>

**Example**

```html
<input name="tokens" value="{{.token}}" type="checkbox" checked>
```

```hjson
{do: "startup-create-streams"}
```

---

## startup-save-task

Records one completed startup-wizard task on the Domain, so the wizard can tell which of its Theme's tasks the owner has already worked through. Writes on `POST` only, and every guard is a silent no-op.

**Attributes**

| Attribute | Description |
| --- | --- |
| value | **Required.** Name of the completed task. 1–32 characters, enforced at load time to match the `startupTasks` property in `model.DomainSchema()` |

<br>

**Example**

```hjson
{
	do: "startup-save-task"
	value: "profile"
}
```

---

## switch

Renders `expression` as a template, then runs the case whose key equals the result, in the current phase. When no key matches, it runs `default`, or nothing if there is no `default`. Matching is exact and case-sensitive, after trimming whitespace from both ends of the rendered value. A case that matches but holds an empty list runs nothing; it does not fall through to `default`. The state and role requirements of every case and the default are rolled into this step's own, as with [`if`](#if).

**Attributes**

| Attribute | Description |
| --- | --- |
| expression | **Required.** Template expression whose rendered value picks the case. An expression that fails to render halts the pipeline with an error |
| cases | **Required.** Map of values to sub-pipelines, with at least one entry. Each sub-pipeline is a single step or a list of steps |
| default | Sub-pipeline run when no case matches: a single step or a list of steps |

<br>

**Example**

```hjson
{
	do: "switch"
	expression: "{{.Data `layout`}}"
	cases: {
		podcast: [
			{do: "view-html", file: "edit-podcast"}
			{do: "save", comment: "Podcast updated"}
		]
		gallery: {do: "view-html", file: "edit-gallery"}
	}
	default: [
		{do: "view-html", file: "edit-articles"}
	]
}
```

---

## trigger-event

Fires one named HX-Trigger event with a template-evaluated value, on `POST`. Use [`add-event`](#add-event) when the event needs to fire on `GET`.

**Attributes**

| Attribute | Description |
| --- | --- |
| event | **Required.** Name of the client event to fire |
| value | Template value sent with the event |

<br>

**Example**

```hjson
{
	do: "trigger-event"
	event: "streamUpdated"
	value: "{{.StreamID}}"
}
```

---

## unpublish

Retracts a published Stream and returns it to `state`. The inverse of [`save-and-publish`](#save-and-publish). Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| state | State to return to. Defaults to `default`, and must be defined in the Template's `states` |
| outbox | Also send the retraction to this User's outbox |

<br>

**Example**

```hjson
{
	do: "unpublish"
	outbox: true
}
```

---

## upload-attachments

Accepts file uploads and attaches them to the current object. The optional `*-path` attributes write the resulting AttachmentID, download URL, and original filename back into the object's schema, which is how a single-file upload becomes an `iconId` or a `coverImage`.

Every posted file is checked against `accept-type` before any is stored, so a refused file leaves nothing behind. New files take ranks after the ones the category already holds, in the order posted, and `replace` starts them again at 1.

**Attributes**

| Attribute | Description |
| --- | --- |
| action | `append` deletes the oldest attachments in the category to make room beneath `maximum`, `replace` deletes them all, and `fill` deletes none, so it accepts only as many new files as fit. Anything else is treated as `append` |
| fieldname | Form field holding the file data. Defaults to `file` |
| attachment&#8209;path | Schema path that receives the AttachmentID |
| download&#8209;path | Schema path that receives the download URL |
| filename&#8209;path | Schema path that receives the original filename |
| accept&#8209;type | MIME type filter, e.g. `image/*` |
| category | Category applied to each Attachment |
| maximum | Maximum number of uploads. Defaults to `1`, and can never be lower |
| minimum | Fewest attachments the category may hold once the upload is done. A post that would leave fewer, including one with no file at all, halts with an inline error. Defaults to `0`, and cannot exceed `maximum` |
| json&#8209;result | Return a JSON structure instead of HTML. Forces `maximum: 1` |
| label | Literal value for `attachment.label` |
| label&#8209;fieldname | Form field that supplies the label |
| description | Literal value for `attachment.description` |
| description&#8209;fieldname | Form field that supplies the description |
| rules.height | Fixed height for all downloads |
| rules.width | Fixed width for all downloads |
| rules.types | Allowed extensions. The first value is the default |

<br>

**Example**

```hjson
{
	do: "upload-attachments"
	action: "replace"
	accept-type: "image/*"
	category: "avatar"
	download-path: "iconUrl"
	rules: {
		width: 400
		height: 400
		types: ["webp"]
	}
}
```

---

## view-attachment

Serves the attachment named by the `attachmentId` query parameter, transcoded to match the rules configured here. Stream builders only. Honors `If-None-Match` and answers `304 Not Modified`.

**Attributes**

| Attribute | Description |
| --- | --- |
| format | **Required.** Allowed output types. At least one, enforced at load time |
| category | Attachment must belong to one of these categories to be reachable |
| width | Allowed widths, for images and video |
| height | Allowed heights, for images and video |
| bitrate | Allowed bitrates, for audio and video |
| metadata | `translate` pipeline used to generate metadata. Parsed at load time |
| cache | Cache the transcoded result. Defaults to `true` |

<br>

**Example**

```hjson
{
	do: "view-attachment"
	category: ["cover"]
	format: ["webp", "jpg"]
	width: [1200, 600]
}
```

---

## view-css

Renders a template file and serves it as a stylesheet.

**Attributes**

| Attribute | Description |
| --- | --- |
| file | Name of the file in the Template directory. Defaults to the name of the action that declares the step, even when another action reaches it through `include` |

<br>

**Example**

```hjson
{
	do: "view-css"
	file: "stylesheet"
}
```

---

## view-feed

Renders the Stream's children — or a search query — as RSS, Atom, or JSONFeed. The format comes from the `format` query parameter (`json`, `atom`, `rss`), falling back to `Accept` header negotiation. Child Streams are filtered by the *viewer's* permissions as well as the publish-date window, so gated content is never leaked through a feed.

**Attributes**

| Attribute | Description |
| --- | --- |
| search&#8209;types | When present, the feed is built from search results of these types instead of child Streams |

<br>

**Example**

```hjson
{do: "view-feed"}
```

---

## view-html

Renders one of the Template's HTML files. The workhorse of nearly every `view` action. Publishes `ETag` and `Last-Modified`, but deliberately does not act on `If-None-Match` — an index page goes stale as soon as a child changes, and nothing invalidates the parent.

**Attributes**

| Attribute | Description |
| --- | --- |
| file | Name of the `.html` file in the Template directory. Defaults to the name of the action that declares the step, even when another action reaches it through `include` |
| method | `get`, `post`, or `both`. Defaults to `get` |
| cache&#8209;control | Overrides the default `Cache-Control` policy, which lives in [/build](../../build/) beside the headers it guards |
| as&#8209;full&#8209;page | Render the full page chrome, not just the fragment |

<br>

**Example**

```hjson
{
	do: "view-html"
	file: "detail"
	as-full-page: true
}
```

---

## view-json

Evaluates a template expression and writes it as JSON. The value is wrapped in `{{ … | json}}` internally, so write the bare expression without the braces.

**Attributes**

| Attribute | Description |
| --- | --- |
| value | **Required.** Template expression, without the surrounding `{{ }}`. Enforced at load time |
| jsonp | Callback name that wraps the result as JSONP |

<br>

**Example**

```hjson
{
	do: "view-json"
	value: ".JSONLD"
}
```

---

## with-annotation

Switches to the Annotation named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Annotation Builder |

<br>

**Example**

```hjson
{
	do: "with-annotation"
	steps: [
		{do: "edit"}
		{do: "save"}
	]
}
```

---

## with-attachment

Switches to the Attachment named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Attachment Builder |

<br>

**Example**

```hjson
{
	do: "with-attachment"
	steps: [
		{do: "delete"}
	]
}
```

---

## with-children

Runs `steps` against **each** child Stream in turn — the only `with-*` step that iterates. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run once per child Stream |

<br>

**Example**

```hjson
{
	do: "with-children"
	steps: [
		{
			do: "set-state"
			state: "archived"
		}
		{do: "save"}
	]
}
```

---

## with-circle

Switches to the Circle named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Circle Builder |

<br>

**Example**

```hjson
{
	do: "with-circle"
	steps: [
		{do: "edit"}
		{do: "save"}
	]
}
```

---

## with-draft

Switches to the current Stream's draft and runs `steps` against its Builder. Requires the `Stream` model. This is the only `with-*` step that propagates its sub-steps' state requirements, since a draft shares its Stream's state machine.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the StreamDraft Builder |

<br>

**Example**

```hjson
{
	do: "with-draft"
	steps: [
		{
			do: "edit-content"
			format: "EDITORJS"
		}
		{do: "save"}
	]
}
```

---

## with-folder

Switches to the Folder named by the request and runs `steps` against its Builder. Requires an authenticated user.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Folder Builder |

<br>

**Example**

```hjson
{
	do: "with-folder"
	steps: [
		{do: "edit"}
		{do: "save"}
	]
}
```

---

## with-follower

Switches to the Follower named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Follower Builder |

<br>

**Example**

```hjson
{
	do: "with-follower"
	steps: [
		{do: "delete"}
	]
}
```

---

## with-following

Switches to the Following named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Following Builder |

<br>

**Example**

```hjson
{
	do: "with-following"
	steps: [
		{do: "edit"}
		{do: "save"}
	]
}
```

---

## with-import

Switches to the Import named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Import Builder |

<br>

**Example**

```hjson
{
	do: "with-import"
	steps: [
		{
			do: "view-html"
			file: "status"
		}
	]
}
```

---

## with-keypackage

Switches to the KeyPackage named by the request and runs `steps` against its Builder. Requires the `Settings` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the KeyPackage Builder |

<br>

**Example**

```hjson
{
	do: "with-keypackage"
	steps: [
		{do: "delete"}
	]
}
```

---

## with-merchant-account

Switches to the MerchantAccount named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the MerchantAccount Builder |

<br>

**Example**

```hjson
{
	do: "with-merchant-account"
	steps: [
		{do: "edit"}
		{do: "save"}
	]
}
```

---

## with-message

Switches to the Message named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Message Builder |

<br>

**Example**

```hjson
{
	do: "with-message"
	steps: [
		{
			do: "set-data"
			values: {readDate: "0"}
		}
		{do: "save"}
	]
}
```

---

## with-next-sibling

Switches to the next sibling of the current Stream and runs `steps` against its Builder. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the sibling Stream Builder |

<br>

**Example**

```hjson
{
	do: "with-next-sibling"
	steps: [
		{
			do: "view-html"
			file: "nav-link"
		}
	]
}
```

---

## with-notification

Switches to the Notification named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Notification Builder |

<br>

**Example**

```hjson
{
	do: "with-notification"
	steps: [
		{do: "delete"}
	]
}
```

---

## with-oauth-token

Switches to the OAuth user token named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the OAuthUserToken Builder |

<br>

**Example**

```hjson
{
	do: "with-oauth-token"
	steps: [
		{do: "delete"}
	]
}
```

---

## with-parent

Switches to the parent of the current Stream and runs `steps` against its Builder. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the parent Stream Builder |

<br>

**Example**

```hjson
{
	do: "with-parent"
	steps: [
		{
			do: "forward-to"
			url: "/{{.StreamID}}"
		}
	]
}
```

---

## with-prev-sibling

Switches to the previous sibling of the current Stream and runs `steps` against its Builder. Requires the `Stream` model.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the sibling Stream Builder |

<br>

**Example**

```hjson
{
	do: "with-prev-sibling"
	steps: [
		{
			do: "view-html"
			file: "nav-link"
		}
	]
}
```

---

## with-privilege

Switches to the Privilege named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Privilege Builder |

<br>

**Example**

```hjson
{
	do: "with-privilege"
	steps: [
		{do: "delete"}
	]
}
```

---

## with-response

Switches to the Response named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Response Builder |

<br>

**Example**

```hjson
{
	do: "with-response"
	steps: [
		{do: "delete"}
	]
}
```

---

## with-rule

Switches to the Rule named by the request and runs `steps` against its Builder.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the Rule Builder |

<br>

**Example**

```hjson
{
	do: "with-rule"
	steps: [
		{do: "edit"}
		{do: "save"}
	]
}
```

---

## with-stream-source

Switches to the `StreamSource` record attached to this Stream — creating one in memory when the Stream has none yet — and runs `steps` against it. This is how a Template creates, configures, and removes a remote content source: `StreamSource` is its own collection, so `set-data` and `save` on the Stream cannot reach it. Requires the `Stream` model.

`save` also queues a synchronization with the remote file — every time, because nothing polls and a repeat costs one conditional GET that answers `304`. A **Sync Now** button is therefore just a `save` with nothing else in the pipeline.

**Attributes**

| Attribute | Description |
| --- | --- |
| steps | **Required.** Sub-pipeline run against the `StreamSource` Builder |

<br>

**Example**

```hjson
edit-source: [{do: "as-modal", steps: [
	{do: "with-stream-source", steps: [
		{do: "edit", form: {
			type: layout-vertical
			label: Remote Content Source
			children: [
				{type: text, path: url, label: "Markdown File URL"}
				{type: text, path: "config.webhookToken", label: "Webhook Token"}
			]
		}}
		{do: "save"}
	]}
]}]

// No refresh-page: `save` already sends an SSE update that the settings screen listens for,
// and a second trigger redraws the screen twice
sync-source: [{do: "with-stream-source", steps: [
	{do: "save"}
]}]

delete-source: [{do: "with-stream-source", steps: [
	{do: "delete", title: "Stop syncing this article?"}
	{do: "refresh-page"}
]}]
```
