package main

import (
	"fmt"
	"net/smtp"

	"github.com/emersion/go-sasl"
)

// SendEmail sends an email via SMTP
func (e *EmailClient) SendEmail(acc Account, to []string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", acc.SMTPServer, acc.SMTPPort)

	// Build message
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		acc.Email,
		to[0],
		subject,
		body,
	)

	auth := sasl.NewPlainClient("", acc.Username, acc.Password)

	if acc.SMTPPort == 465 {
		// SMTPS (TLS)
		return e.sendSMTPS(addr, auth, acc.Email, to, []byte(msg))
	}

	// STARTTLS
	return e.sendSTARTTLS(addr, auth, acc.Email, to, []byte(msg))
}

func (e *EmailClient) sendSMTPS(addr string, auth sasl.Client, from string, to []string, msg []byte) error {
	// TLS connection
	// TODO: Implement SMTPS
	return fmt.Errorf("SMTPS not yet implemented")
}

func (e *EmailClient) sendSTARTTLS(addr string, auth sasl.Client, from string, to []string, msg []byte) error {
	// Connect and upgrade to TLS
	// TODO: Implement STARTTLS
	return smtp.SendMail(addr, nil, from, to, msg)
}
