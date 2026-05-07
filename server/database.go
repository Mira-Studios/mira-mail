package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Database handles all database operations
type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dataDir string) (*Database, error) {
	dbPath := fmt.Sprintf("%s/mira-mail.db", dataDir)
	
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{db: db}
	
	// Initialize tables
	if err := database.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return database, nil
}

// initTables creates all necessary tables
func (d *Database) initTables() error {
	queries := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			email TEXT,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Email accounts table
		`CREATE TABLE IF NOT EXISTS email_accounts (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			imap_server TEXT NOT NULL,
			imap_port INTEGER NOT NULL,
			smtp_server TEXT NOT NULL,
			smtp_port INTEGER NOT NULL,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			use_tls BOOLEAN NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`,

		// Mailboxes table
		`CREATE TABLE IF NOT EXISTS mailboxes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id TEXT NOT NULL,
			name TEXT NOT NULL,
			message_count INTEGER DEFAULT 0,
			unread_count INTEGER DEFAULT 0,
			last_synced DATETIME,
			FOREIGN KEY (account_id) REFERENCES email_accounts (id),
			UNIQUE(account_id, name)
		)`,

		// Emails table
		`CREATE TABLE IF NOT EXISTS emails (
			id TEXT PRIMARY KEY,
			account_id TEXT NOT NULL,
			mailbox TEXT NOT NULL,
			uid INTEGER NOT NULL,
			subject TEXT,
			from_address TEXT,
			to_addresses TEXT,
			cc_addresses TEXT,
			bcc_addresses TEXT,
			body TEXT,
			body_html TEXT,
			date_sent DATETIME,
			date_received DATETIME DEFAULT CURRENT_TIMESTAMP,
			read BOOLEAN DEFAULT 0,
			starred BOOLEAN DEFAULT 0,
			trashed BOOLEAN DEFAULT 0,
			draft BOOLEAN DEFAULT 0,
			attachments TEXT,
			message_id TEXT,
			thread_id TEXT,
			FOREIGN KEY (account_id) REFERENCES email_accounts (id),
			UNIQUE(account_id, uid)
		)`,

		// Email labels/tags table
		`CREATE TABLE IF NOT EXISTS email_labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email_id TEXT NOT NULL,
			label TEXT NOT NULL,
			FOREIGN KEY (email_id) REFERENCES emails (id),
			UNIQUE(email_id, label)
		)`,

		// Internal emails table (for user-to-user messages)
		`CREATE TABLE IF NOT EXISTS internal_emails (
			id TEXT PRIMARY KEY,
			from_user TEXT NOT NULL,
			to_users TEXT NOT NULL,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			read BOOLEAN DEFAULT 0
		)`,

		// Create indexes for better performance
		`CREATE INDEX IF NOT EXISTS idx_emails_account_mailbox ON emails(account_id, mailbox)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_date_sent ON emails(date_sent DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_read ON emails(read)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_starred ON emails(starred)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_trashed ON emails(trashed)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// StoreEmail saves an email to the database
func (d *Database) StoreEmail(email *Email, accountID, mailbox string, uid uint32) error {
	query := `INSERT OR REPLACE INTO emails (
		id, account_id, mailbox, uid, subject, from_address, to_addresses, 
		cc_addresses, bcc_addresses, body, date_sent, read, starred, trashed, draft
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	dateSent, err := time.Parse(time.RFC3339, email.Date)
	if err != nil {
		dateSent = time.Now()
	}

	_, err = d.db.Exec(query,
		email.ID, accountID, mailbox, uid, email.Subject, email.From,
		joinAddresses(email.To), joinAddresses([]string{}), joinAddresses([]string{}),
		email.Body, dateSent, email.Read, email.Starred, false, false,
	)

	return err
}

// GetEmails retrieves emails from database with pagination
func (d *Database) GetEmails(accountID, mailbox string, limit, offset int) ([]Email, error) {
	query := `SELECT id, subject, from_address, to_addresses, body, date_sent, 
		read, starred, labels, attachments FROM emails WHERE account_id = ? AND mailbox = ? 
		AND trashed = 0 ORDER BY date_sent DESC LIMIT ?1 OFFSET ?2`

	rows, err := d.db.Query(query, accountID, mailbox, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []Email
	for rows.Next() {
		var email Email
		var toAddresses, labels, attachments string
		var dateSent time.Time

		err := rows.Scan(
			&email.ID, &email.Subject, &email.From, &toAddresses,
			&email.Body, &dateSent, &email.Read, &email.Starred,
			&labels, &attachments,
		)
		if err != nil {
			return nil, err
		}

		email.To = splitAddresses(toAddresses)
		email.Labels = splitAddresses(labels)
		email.Attachments = parseAttachments(attachments)
		email.Date = dateSent.Format(time.RFC3339)
		emails = append(emails, email)
	}

	return emails, nil
}

// GetEmail retrieves a single email by ID
func (d *Database) GetEmail(emailID string) (*Email, error) {
	query := `SELECT id, subject, from_address, to_addresses, cc_addresses, 
		bcc_addresses, body, date_sent, read, starred, labels, attachments FROM emails WHERE id = ?`

	row := d.db.QueryRow(query, emailID)

	var email Email
	var toAddresses, ccAddresses, bccAddresses, labels, attachments string
	var dateSent time.Time

	err := row.Scan(
		&email.ID, &email.Subject, &email.From, &toAddresses, &ccAddresses,
		&bccAddresses, &email.Body, &dateSent, &email.Read, &email.Starred,
		&labels, &attachments,
	)
	if err != nil {
		return nil, err
	}

	email.To = splitAddresses(toAddresses)
	email.Labels = splitAddresses(labels)
	email.Attachments = parseAttachments(attachments)
	email.Date = dateSent.Format(time.RFC3339)
	return &email, nil
}

// UpdateEmailStatus updates email read/starred/trashed status
func (d *Database) UpdateEmailStatus(emailID string, read, starred *bool, trashed *bool) error {
	query := `UPDATE emails SET `
	var params []interface{}
	var updates []string

	if read != nil {
		updates = append(updates, "read = ?")
		params = append(params, *read)
	}
	if starred != nil {
		updates = append(updates, "starred = ?")
		params = append(params, *starred)
	}
	if trashed != nil {
		updates = append(updates, "trashed = ?")
		params = append(params, *trashed)
	}

	if len(updates) == 0 {
		return fmt.Errorf("no updates specified")
	}

	query += joinStrings(updates, ", ") + " WHERE id = ?"
	params = append(params, emailID)

	_, err := d.db.Exec(query, params...)
	return err
}

// GetMailboxSummary returns counts for each mailbox
func (d *Database) GetMailboxSummary(accountID string) (*MailboxSummary, error) {
	summary := &MailboxSummary{}

	// Inbox count
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM emails WHERE account_id = ? AND mailbox = 'INBOX' AND trashed = 0",
		accountID,
	).Scan(&summary.Inbox)
	if err != nil {
		log.Printf("Error getting inbox count: %v", err)
	}

	// Unread count
	err = d.db.QueryRow(
		"SELECT COUNT(*) FROM emails WHERE account_id = ? AND read = 0 AND trashed = 0",
		accountID,
	).Scan(&summary.Unread)
	if err != nil {
		log.Printf("Error getting unread count: %v", err)
	}

	// Starred count
	err = d.db.QueryRow(
		"SELECT COUNT(*) FROM emails WHERE account_id = ? AND starred = 1 AND trashed = 0",
		accountID,
	).Scan(&summary.Starred)
	if err != nil {
		log.Printf("Error getting starred count: %v", err)
	}

	// Sent count
	err = d.db.QueryRow(
		"SELECT COUNT(*) FROM emails WHERE account_id = ? AND mailbox IN ('Sent', 'Sent Items') AND trashed = 0",
		accountID,
	).Scan(&summary.Sent)
	if err != nil {
		log.Printf("Error getting sent count: %v", err)
	}

	// Drafts count
	err = d.db.QueryRow(
		"SELECT COUNT(*) FROM emails WHERE account_id = ? AND draft = 1 AND trashed = 0",
		accountID,
	).Scan(&summary.Drafts)
	if err != nil {
		log.Printf("Error getting drafts count: %v", err)
	}

	// Trash count
	err = d.db.QueryRow(
		"SELECT COUNT(*) FROM emails WHERE account_id = ? AND trashed = 1",
		accountID,
	).Scan(&summary.Trash)
	if err != nil {
		log.Printf("Error getting trash count: %v", err)
	}

	return summary, nil
}

// Helper functions
func joinAddresses(addresses []string) string {
	if len(addresses) == 0 {
		return ""
	}
	result := addresses[0]
	for i := 1; i < len(addresses); i++ {
		result += "," + addresses[i]
	}
	return result
}

func splitAddresses(addresses string) []string {
	if addresses == "" {
		return []string{}
	}
	// Split on comma and trim spaces
	parts := strings.Split(addresses, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func parseAttachments(attachments string) []Attachment {
	if attachments == "" {
		return []Attachment{}
	}
	// Simple parsing for now - in production, implement proper JSON parsing
	return []Attachment{}
}

// GetInternalEmails retrieves internal emails from the database
func (d *Database) GetInternalEmails() ([]InternalEmail, error) {
	query := `SELECT id, from_user, to_users, subject, body, created_at, read 
		FROM internal_emails ORDER BY created_at DESC`
	
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []InternalEmail
	for rows.Next() {
		var email InternalEmail
		var toUsers string
		var dateStr string
		err := rows.Scan(&email.ID, &email.From, &toUsers, &email.Subject, 
			&email.Body, &dateStr, &email.Read)
		if err != nil {
			continue
		}
		email.Date = dateStr
		email.To = splitAddresses(toUsers)
		emails = append(emails, email)
	}

	return emails, nil
}

// GetInternalEmail retrieves a single internal email by ID
func (d *Database) GetInternalEmail(id string) (*InternalEmail, error) {
	query := `SELECT id, from_user, to_users, subject, body, created_at, read 
		FROM internal_emails WHERE id = ?`
	
	row := d.db.QueryRow(query, id)
	
	var email InternalEmail
	var toUsers string
	var dateStr string
	
	err := row.Scan(&email.ID, &email.From, &toUsers, &email.Subject, 
		&email.Body, &dateStr, &email.Read)
	if err != nil {
		return nil, err
	}
	
	email.Date = dateStr
	email.To = splitAddresses(toUsers)
	return &email, nil
}

// CreateUser creates a new user in the database
func (d *Database) CreateUser(username, name, email, passwordHash string) error {
	query := `INSERT INTO users (username, name, email, password_hash) VALUES (?, ?, ?, ?)`
	_, err := d.db.Exec(query, username, name, email, passwordHash)
	return err
}

// GetUserByUsername retrieves a user by username
func (d *Database) GetUserByUsername(username string) (*User, error) {
	query := `SELECT username, name, email, password_hash FROM users WHERE username = ?`
	row := d.db.QueryRow(query, username)
	
	var user User
	err := row.Scan(&user.Username, &user.Name, &user.Email, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAllUsers retrieves all users from the database
func (d *Database) GetAllUsers() ([]User, error) {
	query := `SELECT username, name, email, password_hash FROM users ORDER BY username`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.Username, &user.Name, &user.Email, &user.PasswordHash)
		if err != nil {
			continue
		}
		users = append(users, user)
	}
	return users, nil
}

// UpdateUser updates a user's information
func (d *Database) UpdateUser(username, name, email string) error {
	query := `UPDATE users SET name = ?, email = ? WHERE username = ?`
	_, err := d.db.Exec(query, name, email, username)
	return err
}

// DeleteUser deletes a user from the database
func (d *Database) DeleteUser(username string) error {
	query := `DELETE FROM users WHERE username = ?`
	_, err := d.db.Exec(query, username)
	return err
}
