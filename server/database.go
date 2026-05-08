package main

import (
	"database/sql"
	"encoding/json"
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
	// First create tables
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

		// Domains table - for user-owned custom domains
		`CREATE TABLE IF NOT EXISTS domains (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT UNIQUE NOT NULL,
			verified BOOLEAN NOT NULL DEFAULT 0,
			verification_token TEXT,
			mx_configured BOOLEAN NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Domain emails table - email addresses under custom domains
		`CREATE TABLE IF NOT EXISTS domain_emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain_id INTEGER NOT NULL,
			local_part TEXT NOT NULL,
			password_hash TEXT,
			user_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (domain_id) REFERENCES domains (id),
			FOREIGN KEY (user_id) REFERENCES users (id),
			UNIQUE(domain_id, local_part)
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

		// Custom domain emails table (sent and received)
		`CREATE TABLE IF NOT EXISTS custom_domain_emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_email TEXT NOT NULL,
			to_emails TEXT NOT NULL, -- JSON array
			cc_emails TEXT, -- JSON array
			bcc_emails TEXT, -- JSON array
			subject TEXT,
			body TEXT,
			direction TEXT NOT NULL CHECK (direction IN ('sent', 'received')),
			domain_id INTEGER NOT NULL,
			user_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (domain_id) REFERENCES domains (id),
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

	// Add migrations for existing tables
	if err := d.migrateTables(); err != nil {
		return fmt.Errorf("failed to migrate tables: %w", err)
	}

	// Create custom domain emails table if needed
	if err := d.migrateCustomDomainEmails(); err != nil {
		return fmt.Errorf("failed to migrate custom domain emails: %w", err)
	}

	return nil
}

// migrateTables handles database schema migrations
func (d *Database) migrateTables() error {
	// Check if user_id column exists in domain_emails table
	var hasUserIDColumn bool
	checkColumnQuery := `PRAGMA table_info(domain_emails)`
	rows, err := d.db.Query(checkColumnQuery)
	if err != nil {
		return fmt.Errorf("failed to check domain_emails table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}
		if name == "user_id" {
			hasUserIDColumn = true
			break
		}
	}

	// Add user_id column if it doesn't exist
	if !hasUserIDColumn {
		// First, make password_hash nullable and add user_id column
		migrationQueries := []string{
			`ALTER TABLE domain_emails ADD COLUMN user_id INTEGER REFERENCES users(id)`,
			// Create a new table without the NOT NULL constraint on password_hash
			`CREATE TABLE domain_emails_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				domain_id INTEGER NOT NULL,
				local_part TEXT NOT NULL,
				password_hash TEXT,
				user_id INTEGER,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (domain_id) REFERENCES domains (id),
				FOREIGN KEY (user_id) REFERENCES users (id),
				UNIQUE(domain_id, local_part)
			)`,
			// Copy data from old table
			`INSERT INTO domain_emails_new (id, domain_id, local_part, password_hash, created_at) 
			 SELECT id, domain_id, local_part, password_hash, created_at FROM domain_emails`,
			// Drop old table
			`DROP TABLE domain_emails`,
			// Rename new table
			`ALTER TABLE domain_emails_new RENAME TO domain_emails`,
		}
		
		for _, query := range migrationQueries {
			if _, err := d.db.Exec(query); err != nil {
				return fmt.Errorf("failed to execute migration query '%s': %w", query, err)
			}
		}
		log.Printf("INFO: Added user_id column and made password_hash nullable in domain_emails table")
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
	query := `SELECT id, username, name, email, password_hash FROM users WHERE username = ?`
	row := d.db.QueryRow(query, username)
	
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Name, &user.Email, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (d *Database) GetUserByID(id int64) (*User, error) {
	query := `SELECT id, username, name, email, password_hash FROM users WHERE id = ?`
	row := d.db.QueryRow(query, id)
	
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Name, &user.Email, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAllUsers retrieves all users from the database
func (d *Database) GetAllUsers() ([]User, error) {
	query := `SELECT id, username, name, email, password_hash FROM users ORDER BY username`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Name, &user.Email, &user.PasswordHash)
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

// Domain represents a custom domain owned by a user
type Domain struct {
	ID                int64  `json:"id"`
	Domain            string `json:"domain"`
	Verified          bool   `json:"verified"`
	VerificationToken string `json:"verification_token,omitempty"`
	MXConfigured      bool   `json:"mx_configured"`
}

// DomainEmail represents an email address under a custom domain
type DomainEmail struct {
	ID         int64  `json:"id"`
	DomainID   int64  `json:"domain_id"`
	LocalPart  string `json:"local_part"`
	FullEmail  string `json:"full_email"`
	UserID     *int64 `json:"user_id,omitempty"`
	Username   string `json:"username,omitempty"`
}

// CreateDomain adds a new domain
func (d *Database) CreateDomain(domain string, verificationToken string) (*Domain, error) {
	// Check if domains table exists
	var count int
	checkQuery := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='domains'`
	err := d.db.QueryRow(checkQuery).Scan(&count)
	if err != nil {
		log.Printf("ERROR: Failed to check domains table: %v", err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	log.Printf("DEBUG: Domains table exists: %v", count > 0)
	
	query := `INSERT INTO domains (domain, verification_token) VALUES (?, ?)`
	result, err := d.db.Exec(query, domain, verificationToken)
	if err != nil {
		log.Printf("ERROR: Failed to insert domain: %v", err)
		return nil, err
	}
	id, _ := result.LastInsertId()
	log.Printf("DEBUG: Created domain with ID: %d", id)
	return &Domain{ID: id, Domain: domain, Verified: false, VerificationToken: verificationToken}, nil
}

// GetDomain retrieves a domain by ID
func (d *Database) GetDomain(id int64) (*Domain, error) {
	query := `SELECT id, domain, verified, verification_token, mx_configured FROM domains WHERE id = ?`
	row := d.db.QueryRow(query, id)
	
	var domain Domain
	var verificationToken sql.NullString
	err := row.Scan(&domain.ID, &domain.Domain, &domain.Verified, &verificationToken, &domain.MXConfigured)
	if err != nil {
		return nil, err
	}
	
	if verificationToken.Valid {
		domain.VerificationToken = verificationToken.String
	}
	
	return &domain, nil
}

// GetDomainByName retrieves a domain by its name
func (d *Database) GetDomainByName(domain string) (*Domain, error) {
	query := `SELECT id, domain, verified, verification_token, mx_configured FROM domains WHERE domain = ?`
	row := d.db.QueryRow(query, domain)
	var dmn Domain
	var verificationToken sql.NullString
	err := row.Scan(&dmn.ID, &dmn.Domain, &dmn.Verified, &verificationToken, &dmn.MXConfigured)
	if err != nil {
		return nil, err
	}
	if verificationToken.Valid {
		dmn.VerificationToken = verificationToken.String
	}
	return &dmn, nil
}

// GetAllDomains retrieves all domains
func (d *Database) GetAllDomains() ([]Domain, error) {
	query := `SELECT id, domain, verified, verification_token FROM domains ORDER BY domain`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var domains []Domain
	for rows.Next() {
		var domain Domain
		var verificationToken sql.NullString
		err := rows.Scan(&domain.ID, &domain.Domain, &domain.Verified, &verificationToken)
		if err != nil {
			log.Printf("ERROR: Failed to scan domain row: %v", err)
			continue
		}
		if verificationToken.Valid {
			domain.VerificationToken = verificationToken.String
		}
		domains = append(domains, domain)
	}
	return domains, nil
}

// VerifyDomain marks a domain as verified
func (d *Database) VerifyDomain(id int64) error {
	query := `UPDATE domains SET verified = 1, verification_token = NULL WHERE id = ?`
	_, err := d.db.Exec(query, id)
	return err
}

// DeleteDomain removes a domain
func (d *Database) DeleteDomain(id int64) error {
	query := `DELETE FROM domains WHERE id = ?`
	_, err := d.db.Exec(query, id)
	return err
}

// CreateDomainEmail creates an email address under a custom domain
func (d *Database) CreateDomainEmail(domainID int64, localPart string, userID *int64) (*DomainEmail, error) {
	query := `INSERT INTO domain_emails (domain_id, local_part, user_id) VALUES (?, ?, ?)`
	result, err := d.db.Exec(query, domainID, localPart, userID)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	
	// Get domain name for full email
	var domain Domain
	domainQuery := `SELECT domain FROM domains WHERE id = ?`
	row := d.db.QueryRow(domainQuery, domainID)
	row.Scan(&domain.Domain)
	
	email := &DomainEmail{
		ID:        id,
		DomainID:  domainID,
		LocalPart: localPart,
		FullEmail: localPart + "@" + domain.Domain,
		UserID:    userID,
	}
	
	// Get username if user is assigned
	if userID != nil {
		var username string
		userQuery := `SELECT username FROM users WHERE id = ?`
		userRow := d.db.QueryRow(userQuery, *userID)
		userRow.Scan(&username)
		email.Username = username
	}
	
	return email, nil
}

// GetDomainEmails retrieves all emails for a domain
func (d *Database) GetDomainEmails(domainID int64) ([]DomainEmail, error) {
	query := `SELECT e.id, e.domain_id, e.local_part, d.domain, e.user_id, u.username 
		FROM domain_emails e 
		JOIN domains d ON e.domain_id = d.id 
		LEFT JOIN users u ON e.user_id = u.id
		WHERE e.domain_id = ? ORDER BY e.local_part`
	rows, err := d.db.Query(query, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []DomainEmail
	for rows.Next() {
		var email DomainEmail
		var domain string
		var userID sql.NullInt64
		var username sql.NullString
		err := rows.Scan(&email.ID, &email.DomainID, &email.LocalPart, &domain, &userID, &username)
		if err != nil {
			continue
		}
		email.FullEmail = email.LocalPart + "@" + domain
		if userID.Valid {
			email.UserID = &userID.Int64
		}
		if username.Valid {
			email.Username = username.String
		}
		emails = append(emails, email)
	}
	return emails, nil
}

// GetUserDomainEmails retrieves all domain emails assigned to a specific user
func (d *Database) GetUserDomainEmails(userID int64) ([]DomainEmail, error) {
	query := `SELECT e.id, e.domain_id, e.local_part, d.domain, e.user_id, u.username 
		FROM domain_emails e 
		JOIN domains d ON e.domain_id = d.id 
		LEFT JOIN users u ON e.user_id = u.id
		WHERE e.user_id = ? ORDER BY d.domain, e.local_part`
	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []DomainEmail
	for rows.Next() {
		var email DomainEmail
		var domain string
		var userID sql.NullInt64
		var username sql.NullString
		err := rows.Scan(&email.ID, &email.DomainID, &email.LocalPart, &domain, &userID, &username)
		if err != nil {
			continue
		}
		email.FullEmail = email.LocalPart + "@" + domain
		if userID.Valid {
			email.UserID = &userID.Int64
		}
		if username.Valid {
			email.Username = username.String
		}
		emails = append(emails, email)
	}
	return emails, nil
}

// GetDomainEmailByAddress retrieves a domain email by full address
func (d *Database) GetDomainEmailByAddress(localPart, domain string) (*DomainEmail, error) {
	query := `SELECT e.id, e.domain_id, e.local_part, d.domain 
		FROM domain_emails e 
		JOIN domains d ON e.domain_id = d.id 
		WHERE e.local_part = ? AND d.domain = ?`
	row := d.db.QueryRow(query, localPart, domain)
	var email DomainEmail
	err := row.Scan(&email.ID, &email.DomainID, &email.LocalPart, &email.FullEmail)
	if err != nil {
		return nil, err
	}
	email.FullEmail = email.LocalPart + "@" + email.FullEmail
	return &email, nil
}

// DeleteDomainEmail removes a domain email
func (d *Database) DeleteDomainEmail(id int64) error {
	query := `DELETE FROM domain_emails WHERE id = ?`
	_, err := d.db.Exec(query, id)
	return err
}

// migrateCustomDomainEmails creates the custom domain emails table if it doesn't exist
func (d *Database) migrateCustomDomainEmails() error {
	// Check if custom_domain_emails table exists
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='custom_domain_emails'").Scan(&count)
	if err != nil {
		return err
	}
	
	if count == 0 {
		query := `CREATE TABLE custom_domain_emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_email TEXT NOT NULL,
			to_emails TEXT NOT NULL, -- JSON array
			cc_emails TEXT, -- JSON array
			bcc_emails TEXT, -- JSON array
			subject TEXT,
			body TEXT,
			direction TEXT NOT NULL CHECK (direction IN ('sent', 'received')),
			domain_id INTEGER NOT NULL,
			user_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (domain_id) REFERENCES domains (id),
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`
		_, err = d.db.Exec(query)
		if err != nil {
			return err
		}
	}
	
	// Add mx_configured column to domains table if it doesn't exist
	var hasMXColumn bool
	checkMXColumnQuery := `PRAGMA table_info(domains)`
	rows, err := d.db.Query(checkMXColumnQuery)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}
		if name == "mx_configured" {
			hasMXColumn = true
			break
		}
	}
	
	if !hasMXColumn {
		_, err = d.db.Exec("ALTER TABLE domains ADD COLUMN mx_configured BOOLEAN NOT NULL DEFAULT 0")
		if err != nil {
			return fmt.Errorf("failed to add mx_configured column: %w", err)
		}
		log.Printf("Added mx_configured column to domains table")
	}
	
	// Add updated_at column if it doesn't exist
	var hasUpdatedColumn bool
	checkUpdatedColumnQuery := `PRAGMA table_info(domains)`
	rows2, err := d.db.Query(checkUpdatedColumnQuery)
	if err != nil {
		return err
	}
	defer rows2.Close()
	
	for rows2.Next() {
		var cid2 int
		var name2, dataType2 string
		var notNull2, pk2 int
		var defaultValue2 interface{}
		err := rows2.Scan(&cid2, &name2, &dataType2, &notNull2, &defaultValue2, &pk2)
		if err != nil {
			continue
		}
		if name2 == "updated_at" {
			hasUpdatedColumn = true
			break
		}
	}
	
	if !hasUpdatedColumn {
		// First add the column without default value for existing data
		_, err = d.db.Exec("ALTER TABLE domains ADD COLUMN updated_at DATETIME")
		if err != nil {
			return fmt.Errorf("failed to add updated_at column: %w", err)
		}
		
		// Then update existing rows to have current timestamp
		_, err = d.db.Exec("UPDATE domains SET updated_at = CURRENT_TIMESTAMP")
		if err != nil {
			return fmt.Errorf("failed to update existing rows: %w", err)
		}
		
		log.Printf("Added updated_at column to domains table")
	}
	
	return nil
}

// UpdateDomainMXStatus updates the MX configuration status for a domain
func (d *Database) UpdateDomainMXStatus(domainID int64, mxConfigured bool) error {
	query := `UPDATE domains SET mx_configured = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.Exec(query, mxConfigured, domainID)
	return err
}

// StoreCustomDomainEmail stores a custom domain email in the database
func (d *Database) StoreCustomDomainEmail(from string, to, cc, bcc []string, subject, body, direction string, domainID, userID int64) error {
	query := `INSERT INTO custom_domain_emails (from_email, to_emails, cc_emails, bcc_emails, subject, body, direction, domain_id, user_id) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	// Convert arrays to JSON
	toJSON, _ := json.Marshal(to)
	ccJSON, _ := json.Marshal(cc)
	bccJSON, _ := json.Marshal(bcc)
	
	_, err := d.db.Exec(query, from, toJSON, ccJSON, bccJSON, subject, body, direction, domainID, userID)
	return err
}
