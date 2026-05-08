package main

import (
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Email represents a simplified email message
type Email struct {
	ID          string      `json:"id"`
	Subject     string      `json:"subject"`
	From        string      `json:"from"`
	To          []string    `json:"to"`
	Body        string      `json:"body"`
	Date        string      `json:"date"`
	Read        bool        `json:"read"`
	Starred     bool        `json:"starred"`
	Labels      []string    `json:"labels"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
}

// MailboxSummary holds counts for each mailbox
type MailboxSummary struct {
	Inbox   int `json:"inbox"`
	Starred int `json:"starred"`
	Sent    int `json:"sent"`
	Drafts  int `json:"drafts"`
	Trash   int `json:"trash"`
	Unread  int `json:"unread"`
}

// EmailClient handles IMAP/SMTP operations
type EmailClient struct{}

// NewEmailClient creates a new email client
func NewEmailClient() *EmailClient {
	return &EmailClient{}
}

// TestIMAP tests an IMAP connection
func (e *EmailClient) TestIMAP(server string, port int, username, password string, useTLS bool) error {
	addr := fmt.Sprintf("%s:%d", server, port)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		// Try non-TLS
		c, err = client.Dial(addr)
		if err != nil {
			return err
		}
	}
	defer c.Logout()

	if err := c.Login(username, password); err != nil {
		return err
	}
	return nil
}

// FetchMailbox retrieves emails from a mailbox and syncs to database
func (e *EmailClient) FetchMailbox(acc Account, mailbox string, database *Database) ([]Email, error) {
	addr := fmt.Sprintf("%s:%d", acc.IMAPServer, acc.IMAPPort)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		c, err = client.Dial(addr)
		if err != nil {
			return nil, err
		}
	}
	defer c.Logout()

	if err := c.Login(acc.Username, acc.Password); err != nil {
		return nil, err
	}

	mbox, err := c.Select(mailbox, false)
	if err != nil {
		return nil, err
	}

	if mbox.Messages == 0 {
		return []Email{}, nil
	}

	// Fetch last 50 messages
	seqset := new(imap.SeqSet)
	from := uint32(1)
	if mbox.Messages > 50 {
		from = mbox.Messages - 49
	}
	seqset.AddRange(from, mbox.Messages)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}, messages)
	}()

	var emails []Email
	for msg := range messages {
		email := Email{
			ID:      fmt.Sprintf("%s_%d", acc.ID, msg.Uid),
			Subject: msg.Envelope.Subject,
			From:    formatAddress(msg.Envelope.From),
			To:      formatAddresses(msg.Envelope.To),
			Date:    msg.Envelope.Date.Format(time.RFC3339),
			Read:    !containsFlag(msg.Flags, "\\Seen"),
			Starred: containsFlag(msg.Flags, "\\Flagged"),
		}
		emails = append(emails, email)

		// Store in database
		if database != nil {
			database.StoreEmail(&email, acc.ID, mailbox, msg.Uid)
		}
	}

	if err := <-done; err != nil {
		return nil, err
	}

	// Reverse to show newest first
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}

// FetchEmail retrieves a single email with body
func (e *EmailClient) FetchEmail(acc Account, uid string) (*Email, error) {
	addr := fmt.Sprintf("%s:%d", acc.IMAPServer, acc.IMAPPort)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		c, err = client.Dial(addr)
		if err != nil {
			return nil, err
		}
	}
	defer c.Logout()

	if err := c.Login(acc.Username, acc.Password); err != nil {
		return nil, err
	}

	// Fetch full message
	seqset := new(imap.SeqSet)
	seqset.Add(uid)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchBodyStructure, section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	msg := <-messages
	if msg == nil {
		return nil, fmt.Errorf("message not found")
	}

	// Get body
	var body string
	for _, literal := range msg.Body {
		body = parseBody(literal)
		break
	}

	if err := <-done; err != nil {
		return nil, err
	}

	// Mark as read
	c.UidStore(seqset, imap.AddFlags, []interface{}{"\\Seen"}, nil)

	email := &Email{
		ID:      uid,
		Subject: msg.Envelope.Subject,
		From:    formatAddress(msg.Envelope.From),
		To:      formatAddresses(msg.Envelope.To),
		Body:    body,
		Date:    msg.Envelope.Date.Format(time.RFC3339),
		Read:    true,
		Starred: containsFlag(msg.Flags, "\\Flagged"),
	}

	return email, nil
}

// GetSummary returns mailbox counts
func (e *EmailClient) GetSummary(acc Account) (*MailboxSummary, error) {
	addr := fmt.Sprintf("%s:%d", acc.IMAPServer, acc.IMAPPort)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		c, err = client.Dial(addr)
		if err != nil {
			return nil, err
		}
	}
	defer c.Logout()

	if err := c.Login(acc.Username, acc.Password); err != nil {
		return nil, err
	}

	summary := &MailboxSummary{}

	// Inbox
	mbox, _ := c.Select("INBOX", true)
	if mbox != nil {
		summary.Inbox = int(mbox.Messages)
	}

	// Sent
	mbox, _ = c.Select("Sent", true)
	if mbox == nil {
		mbox, _ = c.Select("Sent Items", true)
	}
	if mbox != nil {
		summary.Sent = int(mbox.Messages)
	}

	// Drafts
	mbox, _ = c.Select("Drafts", true)
	if mbox != nil {
		summary.Drafts = int(mbox.Messages)
	}

	// Trash
	mbox, _ = c.Select("Trash", true)
	if mbox == nil {
		mbox, _ = c.Select("Deleted Items", true)
	}
	if mbox != nil {
		summary.Trash = int(mbox.Messages)
	}

	return summary, nil
}

// Send sends an email via SMTP
func (e *EmailClient) Send(acc Account, to, cc, bcc []string, subject, body string) error {
	// For now, return an error since we need SMTP configuration
	return fmt.Errorf("SMTP sending not configured - use custom domain SMTP")
}

// SendCustomDomainEmail sends an email via custom domain SMTP
func (e *EmailClient) SendCustomDomainEmail(smtpConfig SMTPConfig, from string, to []string, cc, bcc []string, subject, body string) error {
	// Build the email message
	var message strings.Builder
	
	// Headers
	message.WriteString(fmt.Sprintf("From: %s\r\n", from))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	if len(cc) > 0 {
		message.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(cc, ", ")))
	}
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	message.WriteString("\r\n")
	
	// Body
	message.WriteString(body)
	
	// Connect to SMTP server
	auth := smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, smtpConfig.Host)
	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)
	
	recipients := append(to, cc...)
	if len(bcc) > 0 {
		recipients = append(recipients, bcc...)
	}
	err := smtp.SendMail(addr, auth, from, recipients, []byte(message.String()))
	if err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}
	
	return nil
}

func formatAddress(addrs []*imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	addr := addrs[0]
	email := addr.Address()
	if addr.PersonalName != "" {
		return addr.PersonalName + " <" + email + ">"
	}
	return email
}

func formatAddresses(addrs []*imap.Address) []string {
	result := make([]string, len(addrs))
	for i, a := range addrs {
		result[i] = a.Address()
	}
	return result
}

func containsFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}
	return false
}

func parseBody(r io.Reader) string {
	// Simple body parsing - in production, handle multipart properly
	data, _ := io.ReadAll(r)
	return string(data)
}
