package consumer

import (
	"context"
	"iter"
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	"github.com/benpate/floodgate"
	"github.com/benpate/hannibal/clients"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
)

// TestDelivery_IsRegistered confirms that each delivery task reaches a handler. A name that falls
// through the switch returns Ignored, and the queue then fails the task as unrecognized.
func TestDelivery_IsRegistered(t *testing.T) {

	for _, name := range []string{delivery.TaskFanOut, delivery.TaskSend, service.TaskOutbox2Email} {

		task := queue.NewTask(name, mapof.Any{"hostname": "unknown.example"})
		result := New(unresolvableFactory{}).Run(task)

		require.NotEqual(t, queue.ResultStatusIgnored, result.Status, name)
		require.Equal(t, queue.ResultStatusError, result.Status, "%s: an unknown hostname may become known later", name)
	}
}

// TestDelivery_HostnameFromActor confirms that a send task, which carries no hostname argument,
// is routed by the host of the actor that sent the activity
func TestDelivery_HostnameFromActor(t *testing.T) {

	task := queue.NewTask(delivery.TaskSend, mapof.Any{"actor": "https://unknown.example/@user", "recipient": "https://remote.example/@alice", "body": "{}"})
	result := New(unresolvableFactory{}).Run(task)

	require.Equal(t, queue.ResultStatusError, result.Status, "the domain was looked up, and is not loaded")
}

// TestDelivery_Priorities confirms that a fan-out runs promptly after its commit, and that
// deliveries run below user-facing work
func TestDelivery_Priorities(t *testing.T) {

	// RULE: Assert the EXACT value.  A name that matches no case leaves Priority at the -1 sentinel.
	for name, expected := range map[string]int{
		delivery.TaskFanOut:      16,
		delivery.TaskSend:        256,
		service.TaskOutbox2Email: 256,
	} {
		task := queue.NewTask(name, nil)
		task.Priority = -1

		require.NoError(t, PreProcessor(&task))
		require.Equal(t, expected, task.Priority, name)
	}
}

/******************************************
 * A Domain With No Database
 ******************************************/

// domainServer is a server factory that holds one domain, which has no database. It satisfies
// both the consumer's ServerFactory and the one that service.NewFactory takes.
type domainServer struct {
	factory  *service.Factory
	carpool  *clients.Carpool
	queue    *queue.Queue
	recorded []queue.Task
}

// newDomainServer returns a server factory whose one domain, example.com, has every service wired
// and no database. A task that reaches the database panics, so these tests stop short of it.
func newDomainServer(t *testing.T) *domainServer {

	t.Helper()

	server := &domainServer{carpool: clients.NewCarpool()}
	server.queue = queue.New(queue.WithBufferSize(16), queue.WithPreProcessor(func(task *queue.Task) error {
		server.recorded = append(server.recorded, *task)
		return nil
	}))

	factory, err := service.NewFactory(server, config.Domain{Hostname: "example.com"}, "", nil, nil, nil, nil, nil, nil, afero.NewMemMapFs(), afero.NewMemMapFs(), afero.NewMemMapFs(), nil, nil)
	require.NoError(t, err)
	t.Cleanup(factory.Close)

	server.factory = factory
	return server
}

// RangeDomains implements the ServerFactory interface, walking the one domain
func (server *domainServer) RangeDomains() iter.Seq[*service.Factory] {
	return func(yield func(*service.Factory) bool) { yield(server.factory) }
}

// ByHostname implements the ServerFactory interface, resolving every hostname to the one domain
func (server *domainServer) ByHostname(string) (*service.Factory, error) {
	return server.factory, nil
}

// Queue implements the ServerFactory interface
func (server *domainServer) Queue() *queue.Queue { return server.queue }

// CommonDatabase implements the ServerFactory interface; there is no database
func (server *domainServer) CommonDatabase() *mongo.Database { return nil }

// AllowPrivateIPs implements the ServerFactory interface
func (server *domainServer) AllowPrivateIPs() bool { return false }

// Floodgate implements the ServerFactory interface, returning a Gate with no records
func (server *domainServer) Floodgate() floodgate.Gate {
	return floodgate.New(floodgate.NewMemoryStore())
}

// Email implements service.ServerFactory; this domain sends no email
func (server *domainServer) Email() *service.ServerEmail { return nil }

// ClientIP implements service.ServerFactory
func (server *domainServer) ClientIP(*http.Request) string { return "" }

// Hostname implements service.ServerFactory
func (server *domainServer) Hostname(*http.Request) string { return "example.com" }

// DigitalDome implements service.ServerFactory; this domain has no firewall
func (server *domainServer) DigitalDome() *dome.Dome { return nil }

// Carpool implements service.ServerFactory
func (server *domainServer) Carpool() *clients.Carpool { return server.carpool }

// ReviveHost implements service.ServerFactory, and revives nothing
func (server *domainServer) ReviveHost(string) {}

// TestDelivery_RunsThePipeline verifies that a delivery task reaches the domain's own pipeline,
// bound to the actor that the task names
func TestDelivery_RunsThePipeline(t *testing.T) {

	// O5: Delivery had no test past the hostname lookup. These tasks end before the
	// pipeline reads the database, because this domain has none.
	server := newDomainServer(t)

	// test runs one task and checks its result
	test := func(name string, task queue.Task, expected string) {
		t.Run(name, func(t *testing.T) {
			result := Delivery(server, task)
			require.Equal(t, expected, result.Status, result.Error)
		})
	}

	test("a fan-out for an actor this server cannot sign for ends",
		queue.NewTask(delivery.TaskFanOut, mapof.Any{
			"hostname": "example.com",
			"actor":    "https://remote.example/@mallory",
			"activity": mapof.Any{"type": "Create", "actor": "https://remote.example/@mallory", "to": "https://remote.example/@bob"},
		}),
		queue.ResultStatusFailure,
	)

	test("a send with nothing to send ends",
		queue.NewTask(delivery.TaskSend, mapof.Any{"actor": "https://example.com/@000000000000000000000001", "recipient": "https://remote.example/@bob"}),
		queue.ResultStatusFailure,
	)

	test("any other task belongs to another consumer",
		queue.NewTask("Something-Else", mapof.Any{"hostname": "example.com"}),
		queue.ResultStatusIgnored,
	)

	require.Empty(t, server.recorded, "nothing was queued")

	// NOT COVERED: the "Opening database session" branch. The domain's server is a data-mongo
	// Server, whose Session never returns an error, so the branch cannot be reached.
}

// TestOutbox2Email_UsesTheDomain verifies that an email task reaches the domain's own Outbox2
func TestOutbox2Email_UsesTheDomain(t *testing.T) {

	// An email task with no address is refused before the database is read
	server := newDomainServer(t)
	result := Outbox2Email(server.factory, mapof.Any{"actor": "https://example.com/@000000000000000000000001", "address": ""})

	require.Equal(t, queue.ResultStatusError, result.Status)
	require.True(t, derp.IsBadRequest(result.Error), "the refusal comes from the Follower lookup")
}

/******************************************
 * Outbox2-Email
 ******************************************/

// openerFunc opens a database session by calling itself
type openerFunc func(ctx context.Context) (data.Session, error)

// Session calls the function
func (opener openerFunc) Session(ctx context.Context) (data.Session, error) {
	return opener(ctx)
}

// recordingSender records each email it is asked to send, and fails when it is told to
type recordingSender struct {
	calls   []mapof.Any
	failure error
}

// SendEmail records its arguments, and returns the configured failure
func (sender *recordingSender) SendEmail(session data.Session, actorURL string, address string, activity mapof.Any) error {
	sender.calls = append(sender.calls, mapof.Any{"session": session, "actor": actorURL, "address": address, "activity": activity})
	return sender.failure
}

// TestOutbox2Email verifies that an email task sends one email on a plain session, and is retried
// when the session or the send fails
func TestOutbox2Email(t *testing.T) {

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	opener := openerFunc(func(context.Context) (data.Session, error) { return session, nil })
	args := mapof.Any{
		"actor":      "https://example.com/@author",
		"address":    "carol@remote.example",
		"activityId": "https://example.com/@author/pub/outbox/1",
		"activity":   mapof.Any{"type": "Create"},
	}

	t.Run("a send that succeeds", func(t *testing.T) {
		sender := &recordingSender{}

		result := outbox2Email(opener, sender, args)
		require.Equal(t, queue.ResultStatusSuccess, result.Status)

		require.Len(t, sender.calls, 1)
		require.Equal(t, session, sender.calls[0]["session"])
		require.Equal(t, "https://example.com/@author", sender.calls[0]["actor"])
		require.Equal(t, "carol@remote.example", sender.calls[0]["address"])
		require.Equal(t, mapof.Any{"type": "Create"}, sender.calls[0]["activity"])
	})

	t.Run("a send that fails is retried", func(t *testing.T) {
		sender := &recordingSender{failure: derp.Internal("test", "SMTP is down")}

		result := outbox2Email(opener, sender, args)
		require.Equal(t, queue.ResultStatusError, result.Status)
		require.Contains(t, derp.Message(result.Error), "Sending email")
	})

	t.Run("a database that cannot be reached is retried, and nothing is sent", func(t *testing.T) {
		sender := &recordingSender{}
		unreachable := openerFunc(func(context.Context) (data.Session, error) {
			return nil, derp.Internal("test", "Database is down")
		})

		result := outbox2Email(unreachable, sender, args)
		require.Equal(t, queue.ResultStatusError, result.Status)
		require.Empty(t, sender.calls)
	})
}
