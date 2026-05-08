package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

// SMTPServer handles SMTP protocol for custom domain emails
type SMTPServer struct {
	host     string
	port     int
	listener net.Listener
	database *Database
	stopChan chan bool
}

// NewSMTPServer creates a new SMTP server
func NewSMTPServer(host string, port int, db *Database) *SMTPServer {
	return &SMTPServer{
		host:     host,
		port:     port,
		database: db,
		stopChan: make(chan bool),
	}
}

// Start starts the SMTP server
func (s *SMTPServer) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.host, s.port))
	if err != nil {
		return fmt.Errorf("failed to start SMTP server: %w", err)
	}

	log.Printf("SMTP server started on %s:%d", s.host, s.port)

	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.stopChan:
					return
				default:
					log.Printf("SMTP accept error: %v", err)
					continue
				}
			}

			go s.handleConnection(conn)
		}
	}()

	return nil
}

// Stop stops the SMTP server
func (s *SMTPServer) Stop() error {
	if s.listener != nil {
		close(s.stopChan)
		return s.listener.Close()
	}
	return nil
}

// handleConnection handles an incoming SMTP connection
func (s *SMTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	client := &SMTPClient{
		conn:     conn,
		server:   s,
		reader:   bufio.NewReader(conn),
		state:    "greeting",
		helo:     "",
		mailFrom: "",
		rcptTo:   []string{},
		data:     "",
	}

	// Send greeting
	client.sendResponse(220, "Mira Mail SMTP Server Ready")

	// Handle SMTP commands
	for {
		line, err := client.reader.ReadString('\n')
		if err != nil {
			log.Printf("SMTP connection error: %v", err)
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToUpper(parts[0])
		args := parts[1:]

		if !client.handleCommand(cmd, args) {
			break
		}
	}
}

// SMTPClient represents an SMTP client connection
type SMTPClient struct {
	conn     net.Conn
	server   *SMTPServer
	reader   *bufio.Reader
	state    string
	helo     string
	mailFrom string
	rcptTo   []string
	data     string
}

// sendResponse sends an SMTP response
func (c *SMTPClient) sendResponse(code int, message string) {
	response := fmt.Sprintf("%d %s\r\n", code, message)
	c.conn.Write([]byte(response))
}

// handleCommand handles an SMTP command
func (c *SMTPClient) handleCommand(cmd string, args []string) bool {
	switch cmd {
	case "EHLO", "HELO":
		return c.handleHelo(args)
	case "MAIL":
		return c.handleMail(args)
	case "RCPT":
		return c.handleRcpt(args)
	case "DATA":
		return c.handleData()
	case "RSET":
		return c.handleRset()
	case "QUIT":
		return c.handleQuit()
	case "NOOP":
		return c.handleNoop()
	default:
		c.sendResponse(502, "Command not implemented")
		return true
	}
}

// handleHelo handles EHLO/HELO command
func (c *SMTPClient) handleHelo(args []string) bool {
	if len(args) == 0 {
		c.sendResponse(501, "Syntax error in parameters")
		return true
	}

	c.helo = args[0]
	c.state = "mail"

	if strings.ToUpper(args[0]) == "EHLO" {
		c.sendResponse(250, "Mira Mail")
	} else {
		c.sendResponse(250, "OK")
	}

	return true
}

// handleMail handles MAIL FROM command
func (c *SMTPClient) handleMail(args []string) bool {
	if c.state != "mail" {
		c.sendResponse(503, "Bad sequence of commands")
		return true
	}

	if len(args) == 0 || !strings.HasPrefix(strings.ToUpper(args[0]), "FROM:") {
		c.sendResponse(501, "Syntax error in parameters")
		return true
	}

	from := strings.TrimPrefix(args[0], "FROM:")
	from = strings.Trim(from, "<>")

	// Validate the email is from a custom domain we manage
	if !c.isValidCustomDomainEmail(from) {
		c.sendResponse(550, "Sender not allowed - not a managed custom domain")
		return true
	}

	c.mailFrom = from
	c.state = "rcpt"
	c.sendResponse(250, "OK")
	return true
}

// handleRcpt handles RCPT TO command
func (c *SMTPClient) handleRcpt(args []string) bool {
	if c.state != "rcpt" && c.state != "data" {
		c.sendResponse(503, "Bad sequence of commands")
		return true
	}

	if len(args) == 0 || !strings.HasPrefix(strings.ToUpper(args[0]), "TO:") {
		c.sendResponse(501, "Syntax error in parameters")
		return true
	}

	to := strings.TrimPrefix(args[0], "TO:")
	to = strings.Trim(to, "<>")

	c.rcptTo = append(c.rcptTo, to)
	c.state = "data"
	c.sendResponse(250, "OK")
	return true
}

// handleData handles DATA command
func (c *SMTPClient) handleData() bool {
	if c.state != "data" || len(c.rcptTo) == 0 {
		c.sendResponse(503, "Bad sequence of commands")
		return true
	}

	c.sendResponse(354, "Start mail input; end with <CRLF>.<CRLF>")

	// Read email data
	var data strings.Builder
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading email data: %v", err)
			c.sendResponse(451, "Requested action aborted: error in processing")
			return true
		}

		// Check for end of data marker
		if strings.TrimSpace(line) == "." {
			break
		}

		// Handle dot-stuffing (lines starting with . need an extra .)
		if strings.HasPrefix(line, ".") {
			line = "." + line
		}

		data.WriteString(line)
	}

	c.data = data.String()

	// Store the email
	err := c.storeEmail()
	if err != nil {
		log.Printf("Failed to store email: %v", err)
		c.sendResponse(451, "Requested action aborted: error in processing")
		return true
	}

	c.sendResponse(250, "OK: Message accepted")
	c.resetState()
	return true
}

// handleRset handles RSET command
func (c *SMTPClient) handleRset() bool {
	c.resetState()
	c.sendResponse(250, "OK")
	return true
}

// handleQuit handles QUIT command
func (c *SMTPClient) handleQuit() bool {
	c.sendResponse(221, "Bye")
	return false
}

// handleNoop handles NOOP command
func (c *SMTPClient) handleNoop() bool {
	c.sendResponse(250, "OK")
	return true
}

// resetState resets the client state
func (c *SMTPClient) resetState() {
	c.state = "mail"
	c.mailFrom = ""
	c.rcptTo = []string{}
	c.data = ""
}

// isValidCustomDomainEmail checks if the email is from a custom domain we manage
func (c *SMTPClient) isValidCustomDomainEmail(email string) bool {
	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := parts[1]

	// Check if this domain exists in our database
	_, err := c.server.database.GetDomainByName(domain)
	return err == nil
}

// storeEmail stores the received email in the database
func (c *SMTPClient) storeEmail() error {
	// Parse email headers to extract subject
	subject := c.extractSubject(c.data)

	// Store email for each recipient
	for _, recipient := range c.rcptTo {
		// For now, we'll store it in a simple way
		// In a real implementation, you'd parse the email properly
		log.Printf("Storing email from %s to %s with subject: %s", c.mailFrom, recipient, subject)
	}

	return nil
}

// extractSubject extracts subject from email data
func (c *SMTPClient) extractSubject(data string) string {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToUpper(line), "SUBJECT:") {
			return strings.TrimPrefix(line, "SUBJECT:")
		}
	}
	return "(No Subject)"
}
