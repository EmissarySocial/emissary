package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

// Simple command line tool to act like a dumb SMTP server that simply
// logs the commands and data it receives.  It does not actually send
// any email, and it does not do much in the way of error handling or
// validation. This is useful in a few contexts, including local development
// and testing, and as a simple way to log the contents of emails that are
// sent from a system.
func main() {
	listener, err := net.Listen("tcp", ":8025")
	if err != nil {
		log.Fatalf("Failed to set up TCP listener: %v", err)
	}
	defer listener.Close()
	log.Println("SMTP Logger started on port 8025")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("Connection from %s", conn.RemoteAddr().String())

	fmt.Fprintf(conn, "220 smtpLogger Ready\r\n")

	scanner := bufio.NewScanner(conn)
	dataMode := false

	// string that will be used to accumulate data lines
	var data string;

	for scanner.Scan() {
		line := scanner.Text()
		if dataMode {
			// technically we need to wait for <CR><LF>.<CR><LF> to end the data
			// but we'll just look for a line with a single period for now.
			if line == "." {
				// end of email data
				log.Println("\n\n" + data)
				dataMode = false
				log.Println(">>>> EMAIL END <<<<")
				fmt.Fprintf(conn, "250 2.0.0 Ok: queued\r\n")
			} else {
				// accumulate line if it ends with =, remove the = and append the next line
				// until the line does not end with =
				if strings.HasSuffix(line, "=") {
					data += strings.TrimSuffix(line, "=")
				} else {
					data += line + "\n"
				}
			}
		} else {
			// log.Println(line)
			upperLine := strings.ToUpper(line)

			// Respond to SMTP commands
			switch {
			case strings.HasPrefix(upperLine, "EHLO") || strings.HasPrefix(upperLine, "HELO"):
				fmt.Fprintf(conn, "250-Hello\r\n")
				fmt.Fprintf(conn, "250-SIZE 52428800\r\n")
				fmt.Fprintf(conn, "250-8BITMIME\r\n")
				fmt.Fprintf(conn, "250 SMTPUTF8\r\n")
			case strings.HasPrefix(upperLine, "MAIL FROM:"):
				fmt.Fprintf(conn, "250 2.1.0 Ok\r\n")
			case strings.HasPrefix(upperLine, "RCPT TO:"):
				fmt.Fprintf(conn, "250 2.1.5 Ok\r\n")
			case strings.HasPrefix(upperLine, "DATA"):
				dataMode = true
				log.Println(">>>> EMAIL START <<<<")
				fmt.Fprintf(conn, "354 End data with <CR><LF>.<CR><LF>\r\n")
			case strings.HasPrefix(upperLine, "QUIT"):
				fmt.Fprintf(conn, "221 2.0.0 Bye\r\n")
				return
			default:
				fmt.Fprintf(conn, "500 5.5.1 Command unrecognized: \"%s\"\r\n", line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from connection: %v", err)
	}
}
