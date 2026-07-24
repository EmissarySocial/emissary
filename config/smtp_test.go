package config

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestSMTPSchema(t *testing.T) {

	d := NewSMTPConnection()
	s := schema.New(SMTPConnectionSchema())

	table := []tableTestItem{
		{"hostname", "SMTP_HOSTNAME", nil},
		{"username", "SMTP_USERNAME", nil},
		{"password", "SMTP_PASSWORD", nil},
		{"port", "443", 443},
		{"tls", "false", false},
	}

	tableTest_Schema(t, &s, &d, table)
}

// TestSMTPValidate guards against a regression where Validate() passed the
// connection by value. The schema resolves properties through GetPointer (a
// pointer-receiver method), so a value copy fails the PointerGetter interface
// and every populated connection was rejected -- silently disabling all email.
func TestSMTPValidate(t *testing.T) {

	// A fully-populated connection must validate cleanly.
	smtp := SMTPConnection{
		Hostname: "mailhog",
		Username: "u",
		Password: "p",
		Port:     1025,
	}

	require.Nil(t, smtp.Validate())

	// Server() gates on Validate(), so it must now return a usable client.
	server, ok := smtp.Server()
	require.True(t, ok)
	require.NotNil(t, server)

	// An empty connection is caught by IsNil() (not Validate), so it too passes
	// the schema -- callers short-circuit on IsNil before ever calling Server().
	empty := NewSMTPConnection()
	require.Nil(t, empty.Validate())
	require.True(t, empty.IsNil())
}

// TestSMTPTestConnection_Empty confirms that an unconfigured (empty) SMTP block is treated as a
// no-op success -- email is optional, so "not configured" must not be reported as "broken".
func TestSMTPTestConnection_Empty(t *testing.T) {
	empty := NewSMTPConnection()
	require.Nil(t, empty.TestConnection(time.Second))
}

// TestSMTPTestConnection_Unreachable confirms that a populated-but-unreachable server fails (rather
// than saving clean).  Port 1 refuses immediately, so this exercises the failure path without waiting
// on the timeout.
func TestSMTPTestConnection_Unreachable(t *testing.T) {
	smtp := SMTPConnection{Hostname: "127.0.0.1", Port: 1}
	require.NotNil(t, smtp.TestConnection(2*time.Second))
}

// TestSMTPTestConnection_Reachable confirms that a reachable server that completes the SMTP handshake
// passes the test.  A minimal in-process listener stands in for a real mail server.
func TestSMTPTestConnection_Reachable(t *testing.T) {

	host, port, stop := startMockSMTP(t)
	defer stop()

	// No username/password and TLS off, so Connect performs a plain handshake with no AUTH.
	smtp := SMTPConnection{Hostname: host, Port: port}
	require.Nil(t, smtp.TestConnection(5*time.Second))
}

// startMockSMTP starts a minimal SMTP server on a random loopback port that speaks just enough of the
// protocol for go-simple-mail's Connect()/Quit() to succeed: a 220 greeting, 250 to EHLO/HELO, and 221
// to QUIT.  It returns the host, port, and a stop function.
func startMockSMTP(t *testing.T) (string, int, func()) {

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}
			go serveMockSMTP(conn)
		}
	}()

	address := listener.Addr().(*net.TCPAddr)
	return "127.0.0.1", address.Port, func() { _ = listener.Close() }
}

// serveMockSMTP handles one SMTP conversation, answering EHLO/HELO with 250 and QUIT with 221.
func serveMockSMTP(conn net.Conn) {

	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	// Greeting.
	_, _ = writer.WriteString("220 mock.smtp ready\r\n")
	_ = writer.Flush()

	for {
		line, err := reader.ReadString('\n')

		if err != nil {
			return
		}

		command := strings.ToUpper(strings.TrimSpace(line))

		switch {

		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			_, _ = writer.WriteString("250 mock.smtp\r\n")

		case strings.HasPrefix(command, "QUIT"):
			_, _ = writer.WriteString("221 mock.smtp closing\r\n")
			_ = writer.Flush()
			return

		default:
			_, _ = writer.WriteString("250 OK\r\n")
		}

		_ = writer.Flush()
	}
}
