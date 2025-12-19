# Emissary AI Coding Agent Instructions

## Project Overview

Emissary is a Social Web Toolkit - a Fediverse server built in Go that enables developers to build custom social applications using declarative, low-code HTML templates and JSON configs. It bridges multiple federated protocols (ActivityPub, RSS+WebSub, IndieWeb) through the [sherlock library](https://github.com/benpate/sherlock).

## Architecture & Data Flow

### Service Factory Pattern

Emissary uses a centralized `service.Factory` ([service/factory.go](service/factory.go)) that manages all service instances for a domain. Each domain gets its own Factory instance with:
- **Per-domain services**: Stream, User, Follower, Following, etc.
- **Shared server services**: Template, Theme, JWT, Queue
- **Database sessions**: MongoDB via `github.com/benpate/data-mongo`

Services follow constructor patterns like `NewStreamService(factory *Factory, session data.Session)` and are accessed via factory methods like `factory.Stream()`.

### Builder Pipeline System

The core request handling uses a **Builder pattern** ([build/builder_.go](build/builder_.go)) combined with **action pipelines**:

1. **Builders** wrap model objects (Stream, User, Follower, etc.) and handle HTTP requests
2. **Actions** define what operations can be performed (view, edit, delete, publish)
3. **Steps** are composable pipeline operations ([model/step/](model/step/)) like:
   - `edit`: Show a form
   - `save`: Persist changes
   - `set-state`: Transition state machines
   - `view-html`: Render templates
   - `send-email`: Send notifications

Example action pipeline in a Template JSON:
```json
{
  "steps": [
    {"do": "edit"},
    {"do": "save"},
    {"do": "set-state", "state": "published"},
    {"do": "view-html"}
  ]
}
```

### Template & State Machine System

Templates ([model/template.go](model/template.go)) define:
- **States**: Lifecycle stages (draft, published, deleted)
- **Roles**: Access control (owner, author, editor, subscriber)
- **Actions**: What users can do in each state
- **Schema**: JSON Schema for custom data fields
- **HTML Templates**: Go templates for rendering

Templates support inheritance - child templates can extend parent templates. See [service/template.go](service/template.go) for loading and validation logic.

### Model Layer

Models ([model/](model/)) represent database entities:
- **Stream**: Core content objects (posts, pages, etc.)
- **User**: Local user accounts
- **Follower/Following**: Federated relationships
- **Inbox/Outbox**: ActivityPub message queues
- **Domain**: Multi-tenant configuration

All models embed `journal.Journal` for create/update timestamps and use MongoDB ObjectIDs.

## Key Conventions

### Error Handling

Use the `github.com/benpate/derp` package for structured error handling:
```go
if err != nil {
    return derp.Wrap(err, "location.Package.Function", "Context message", debugData)
}
```

### Database Queries

Use the expression builder pattern from `github.com/benpate/exp`:
```go
criteria := exp.Equal("stateId", "published").
    AndEqual("deleteDate", 0)
```

### Service Methods

Follow this pattern for service CRUD operations:
- `New()` - Create empty model
- `Load(id) (Model, error)` - Retrieve by ID
- `Save(model, comment) error` - Persist with journal entry
- `Delete(model, comment) error` - Soft delete

### File Naming

- `model_*.go` - Model definitions
- `service_*.go` - Service implementations  
- `handler_*.go` - HTTP handlers
- `builder_*.go` - Request builders
- `step_*.go` - Pipeline steps (capitalized: `step_SetState.go`)

### Testing

Tests use `github.com/stretchr/testify`:
```go
func TestMyFunction(t *testing.T) {
    result := MyFunction()
    require.Equal(t, expected, result)
}
```

Mock implementations use `github.com/benpate/data-mock` for in-memory database testing.

## Development Workflow

### Running Locally

```bash
# Start MongoDB
docker-compose up -d

# Build and run (creates emissary binary)
go build && ./emissary

# First-time setup mode
./emissary --setup
```

### Building Templates

Templates live in `_embed/templates/` and are embedded at compile time. Each template has:
- `template.hjson` - Configuration (HJSON format)
- `*.html` - Go HTML templates
- `resources/` - Static assets (CSS, JS)

After modifying templates, rebuild the binary to see changes.

### Federation Testing

See [FEDERATION.md](FEDERATION.md) for supported ActivityPub activities and protocols.

Key federation services:
- `service/activityStream.go` - ActivityPub client
- `service/streamArchive.go` - ActivityPub inbox/outbox
- `protocols/` - WebSub, WebMention, WebFinger implementations

## Integration Points

### Custom Libraries (benpate/*)

Emissary heavily uses custom libraries:
- **hannibal**: ActivityPub implementation
- **sherlock**: Multi-protocol federation client
- **rosetta**: Schema validation & data mapping
- **steranko**: Authentication
- **html**: HTML builder utilities
- **form**: Form generation

These are versioned together - check [go.mod](go.mod) for current versions.

### Echo Web Framework

Uses `github.com/labstack/echo/v4` for HTTP routing. See [server.go](server.go) for route setup.

### HTMX Integration

Frontend uses HTMX for dynamic updates. Builders return:
- Full pages for normal requests
- Fragments for HTMX requests (checked via `IsPartialRequest()`)

## Important Gotchas

1. **Template Validation**: Templates validate at server startup. Invalid templates prevent boot. Check logs for validation errors.

2. **State Transitions**: Actions must validate state requirements. An action with `states: ["published"]` only works on published streams.

3. **Access Control**: Builders check permissions via `UserCan(actionID)`. Don't bypass this in custom code.

4. **MongoDB ObjectIDs**: Always use `primitive.ObjectID` from `go.mongodb.org/mongo-driver/bson/primitive`.

5. **Time Handling**: Use `int64` Unix timestamps, not `time.Time`, for database fields. Use `github.com/EmissarySocial/emissary/tools/datetime` for conversions.

## Where to Start

- **Adding a new model**: Start with [model/stream.go](model/stream.go) as reference
- **Adding a service**: See [service/stream.go](service/stream.go) for full CRUD example
- **Adding a template**: Study [_embed/templates/](/_embed/templates/) examples
- **Adding a pipeline step**: Check [model/step/step.go](model/step/step.go) and implement in `build/step_*.go`
