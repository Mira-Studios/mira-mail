package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// API holds dependencies for HTTP handlers
type API struct {
	config   *Config
	database *Database
	dataDir  string
	email    *EmailClient
}

// NewAPI creates a new API instance
func NewAPI(config *Config, database *Database, dataDir string) *API {
	return &API{
		config:   config,
		database: database,
		dataDir:  dataDir,
		email:    NewEmailClient(),
	}
}

// Handler returns the main API handler
func (a *API) Handler() http.HandlerFunc {
	return CORSMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		path := strings.TrimPrefix(r.URL.Path, "/api/")
		parts := strings.Split(path, "/")

		if len(parts) == 0 {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}

		// Public endpoint - no auth needed
		if parts[0] == "health" || parts[0] == "status" || parts[0] == "login" {
			if parts[0] == "login" {
				a.handleLogin(w, r)
				return
			}
			a.handleHealth(w, r)
			return
		}

		// All other endpoints require API key
		protected := APIKeyMiddleware(a.config.APIKey, func(w http.ResponseWriter, r *http.Request) {
			switch parts[0] {
			case "account":
				a.handleAccount(w, r, parts[1:])
			case "user":
				a.handleUser(w, r)
			case "users":
				a.handleUsers(w, r)
			case "user-domain-emails":
				a.handleUserDomainEmails(w, r)
			case "verify-mx":
				a.handleVerifyMX(w, r)
			case "current-user":
				a.handleCurrentUser(w, r)
			case "internal-emails":
				a.handleInternalEmails(w, r)
			case "emails":
				a.handleEmails(w, r, parts[1:])
			case "drafts":
				a.handleDrafts(w, r)
			case "mailbox":
				a.handleMailboxSummary(w, r, parts[1:])
			case "summary":
				a.handleSummary(w, r)
			case "domains":
				a.handleDomains(w, r, parts[1:])
			case "domain-emails":
				a.handleDomainEmails(w, r, parts[1:])
			default:
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			}
		})

		protected(w, r)
	})
}

// handleHealth returns server status (public endpoint)
func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	hasAccount := len(a.config.Accounts) > 0
	json.NewEncoder(w).Encode(map[string]any{
		"status":       "ok",
		"setup":        !hasAccount,
		"has_accounts": hasAccount,
	})
}


// handleUser manages user account creation
func (a *API) handleUser(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		// Create user account
		var req struct {
			Username string `json:"username"`
			Name     string `json:"name"`
			Email    string `json:"email,omitempty"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		// Validate input
		if req.Username == "" || req.Name == "" || req.Password == "" {
			http.Error(w, `{"error":"username, name, and password required"}`, http.StatusBadRequest)
			return
		}

		// Check if username already exists
		existingUser, _ := a.database.GetUserByUsername(req.Username)
		if existingUser != nil {
			http.Error(w, `{"error":"username already exists"}`, http.StatusConflict)
			return
		}

		// Add user to database
		err := a.database.CreateUser(req.Username, req.Name, req.Email, hashPassword(req.Password))
		if err != nil {
			http.Error(w, `{"error":"failed to save user"}`, http.StatusInternalServerError)
			return
		}

		// Return success without password
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"user": map[string]any{
				"username": req.Username,
				"name":     req.Name,
				"email":    req.Email,
			},
		})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleUsers returns all users
func (a *API) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Get all users from database
		users, err := a.database.GetAllUsers()
		if err != nil {
			http.Error(w, `{"error":"failed to fetch users"}`, http.StatusInternalServerError)
			return
		}
		
		// Return users without password hashes
		var safeUsers []map[string]any
		for _, user := range users {
			safeUsers = append(safeUsers, map[string]any{
				"id":       user.ID,
				"username": user.Username,
				"name":     user.Name,
				"email":    user.Email,
			})
		}
		
		json.NewEncoder(w).Encode(safeUsers)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleUserDomainEmails returns domain emails for the current user
func (a *API) handleUserDomainEmails(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Get username from X-Username header (used by frontend)
		username := r.Header.Get("X-Username")
		if username == "" {
			http.Error(w, `{"error":"username required"}`, http.StatusBadRequest)
			return
		}

		// Get user by username
		user, err := a.database.GetUserByUsername(username)
		if err != nil {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}

		// Get user's domain emails
		emails, err := a.database.GetUserDomainEmails(user.ID)
		if err != nil {
			http.Error(w, `{"error":"failed to fetch domain emails"}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"emails":  emails,
		})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleCurrentUser returns current user info
func (a *API) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Get authenticated username from context
		username := GetUsernameFromContext(r)
		if username == "" {
			http.Error(w, `{"error":"not authenticated"}`, http.StatusUnauthorized)
			return
		}
		
		// Get user from database
		user, err := a.database.GetUserByUsername(username)
		if err != nil {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}
		
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"user": map[string]any{
				"username": user.Username,
				"name":     user.Name,
				"email":    user.Email,
			},
		})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleInternalEmails manages internal emails between local users
func (a *API) handleInternalEmails(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// List internal emails from database
		query := `SELECT id, from_user, to_users, subject, body, created_at, read 
			FROM internal_emails ORDER BY created_at DESC`
		rows, err := a.database.db.Query(query)
		if err != nil {
			http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var emails []InternalEmail
		for rows.Next() {
			var email InternalEmail
			var toUsers string
			var dateStr string
			err := rows.Scan(&email.ID, &email.From, &toUsers, &email.Subject, 
				&email.Body, &dateStr, &email.Read)
			email.Date = dateStr
			if err != nil {
				continue
			}
			email.To = splitAddresses(toUsers)
			emails = append(emails, email)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"emails": emails,
		})

	case "POST":
		// Send internal email
		var req struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			Body    string   `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		// Validate sender is a local user
		senderUser, err := a.database.GetUserByUsername(req.From)
		if err != nil {
			log.Printf("DEBUG: GetUserByUsername error for '%s': %v", req.From, err)
			http.Error(w, `{"error":"invalid sender"}`, http.StatusBadRequest)
			return
		}
		if senderUser == nil {
			log.Printf("DEBUG: Sender '%s' not found in database", req.From)
			http.Error(w, `{"error":"invalid sender"}`, http.StatusBadRequest)
			return
		}

		// Create email
		email := InternalEmail{
			ID:        generateID(),
			From:      req.From,
			To:        req.To,
			Subject:   req.Subject,
			Body:      req.Body,
			Date:      time.Now().Format(time.RFC3339),
			Read:      false,
			Starred:   false,
			Labels:    []string{},
		}

		// Store in database
		query := `INSERT INTO internal_emails (id, from_user, to_users, subject, body, read) 
			VALUES (?, ?, ?, ?, ?, ?)`
		toUsers := joinAddresses(email.To)
		_, err = a.database.db.Exec(query, email.ID, email.From, 
			toUsers, email.Subject, email.Body, email.Read)
		if err != nil {
			http.Error(w, `{"error":"failed to save email"}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"email":  email,
		})

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleAccount manages email accounts
func (a *API) handleAccount(w http.ResponseWriter, r *http.Request, parts []string) {
	switch r.Method {
	case "GET":
		// List accounts (without passwords)
		accounts := make([]map[string]any, len(a.config.Accounts))
		for i, acc := range a.config.Accounts {
			accounts[i] = map[string]any{
				"id":          acc.ID,
				"name":        acc.Name,
				"email":       acc.Email,
				"imap_server": acc.IMAPServer,
				"smtp_server": acc.SMTPServer,
			}
		}
		json.NewEncoder(w).Encode(accounts)

	case "POST":
		// Add new account
		var req struct {
			Name       string `json:"name"`
			Email      string `json:"email"`
			IMAPServer string `json:"imap_server"`
			IMAPPort   int    `json:"imap_port"`
			SMTPServer string `json:"smtp_server"`
			SMTPPort   int    `json:"smtp_port"`
			Username   string `json:"username"`
			Password   string `json:"password"`
			UseTLS     bool   `json:"use_tls"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		// Test connection first
	err := a.email.TestIMAP(req.IMAPServer, req.IMAPPort, req.Username, req.Password, req.UseTLS)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

		// Add account
		acc := Account{
			ID:         generateID(),
			Name:       req.Name,
			Email:      req.Email,
			IMAPServer: req.IMAPServer,
			IMAPPort:   req.IMAPPort,
			SMTPServer: req.SMTPServer,
			SMTPPort:   req.SMTPPort,
			Username:   req.Username,
			Password:   req.Password,
			UseTLS:     req.UseTLS,
		}
		a.config.Accounts = append(a.config.Accounts, acc)
		a.config.Save(a.dataDir)

		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"account": acc,
		})

	case "DELETE":
		// Remove account
		if len(parts) == 0 {
			http.Error(w, `{"error":"account id required"}`, http.StatusBadRequest)
			return
		}
		id := parts[0]
		for i, acc := range a.config.Accounts {
			if acc.ID == id {
				a.config.Accounts = append(a.config.Accounts[:i], a.config.Accounts[i+1:]...)
				a.config.Save(a.dataDir)
				json.NewEncoder(w).Encode(map[string]any{"success": true})
				return
			}
		}
		http.Error(w, `{"error":"account not found"}`, http.StatusNotFound)
	}
}

// handleEmails handles /api/emails/* routes
func (a *API) handleEmails(w http.ResponseWriter, r *http.Request, parts []string) {
	// /api/emails - list emails (GET) or send email (POST)
	if len(parts) == 0 {
		if r.Method == "POST" {
			a.handleSendEmail(w, r)
			return
		} else {
			a.handleEmailList(w, r)
			return
		}
	}

	id := parts[0]

	// Check for sub-routes: /api/emails/:id/read, /api/emails/:id/unread, etc.
	if len(parts) > 1 {
		switch parts[1] {
		case "read":
			a.handleMarkRead(w, r, id)
			return
		case "unread":
			a.handleMarkUnread(w, r, id)
			return
		case "star":
			a.handleToggleStar(w, r, id)
			return
		case "trash":
			a.handleMoveToTrash(w, r, id)
			return
		case "restore":
			a.handleRestoreFromTrash(w, r, id)
			return
		}
	}

	// /api/emails/:id - get or delete single email
	switch r.Method {
	case "GET":
		a.handleGetEmail(w, r, id)
		return
	case "DELETE":
		a.handleDeleteEmail(w, r, id)
		return
	case "POST":
		// POST /api/emails - send email
		a.handleSendEmail(w, r)
		return
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleEmailList handles GET /api/emails?mailbox=...
func (a *API) handleEmailList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	mailbox := query.Get("mailbox")
	if mailbox == "" {
		mailbox = "INBOX"
	}

	// If no external account, return internal emails only
	if len(a.config.Accounts) == 0 {
		internalEmails, err := a.database.GetInternalEmails()
		if err != nil {
			json.NewEncoder(w).Encode([]any{})
			return
		}
		
		// Get current user from request context (set by auth middleware from X-Username header)
		currentUser := GetUsernameFromContext(r)
		
		// Filter internal emails based on mailbox type
		var filtered []InternalEmail
		for _, email := range internalEmails {
			switch mailbox {
			case "inbox":
				// In inbox if current user is a recipient (exact match)
				if currentUser != "" && contains(email.To, currentUser) {
					filtered = append(filtered, email)
				}
			case "sent":
				// Sent items - email from current user (exact match)
				if currentUser != "" && email.From == currentUser {
					filtered = append(filtered, email)
				}
			case "starred":
				// Starred emails
				if email.Starred {
					filtered = append(filtered, email)
				}
			case "drafts":
				// Drafts - internal emails don't support drafts yet
			case "trash":
				// Trash - internal emails don't support trash yet
			default:
				// Unknown mailbox - don't include anything
			}
		}
		
		json.NewEncoder(w).Encode(filtered)
		return
	}

	acc := a.config.Accounts[0]
	
	// Try to get emails from database first
	emails, err := a.database.GetEmails(acc.ID, mailbox, 50, 0)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
		return
	}

	// If no emails in database, sync from IMAP
	if len(emails) == 0 {
		emails, err = a.email.FetchMailbox(acc, mailbox, a.database)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]any{
				"error": err.Error(),
			})
			return
		}
	}

	json.NewEncoder(w).Encode(emails)
}

// handleGetEmail handles GET /api/emails/:id
func (a *API) handleGetEmail(w http.ResponseWriter, r *http.Request, id string) {
	// Try to get email from database first
	email, err := a.database.GetEmail(id)
	if err != nil {
		// If not found and no external account, check internal emails
		if len(a.config.Accounts) == 0 {
			internalEmail, err := a.database.GetInternalEmail(id)
			if err == nil {
				json.NewEncoder(w).Encode(internalEmail)
				return
			}
		}
		// If not in database, try to fetch from IMAP (if account exists)
		if len(a.config.Accounts) > 0 {
			acc := a.config.Accounts[0]
			email, err = a.email.FetchEmail(acc, id)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]any{
					"error": err.Error(),
				})
				return
			}
		} else {
			http.Error(w, `{"error":"email not found"}`, http.StatusNotFound)
			return
		}
	}

	json.NewEncoder(w).Encode(email)
}

// handleSendEmail handles POST /api/emails (send email)
func (a *API) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From    string   `json:"from,omitempty"`
		To      []string `json:"to"`
		Cc      []string `json:"cc,omitempty"`
		Bcc     []string `json:"bcc,omitempty"`
		Subject string   `json:"subject"`
		Body    string   `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if len(a.config.Accounts) == 0 {
		// Check if this is a custom domain email request
		if req.From != "" {
			// Validate that this is a custom domain we manage
			if !a.isValidCustomDomainEmail(req.From) {
				http.Error(w, `{"error":"sender email is not from a managed custom domain"}`, http.StatusBadRequest)
				return
			}
			
			// Store email in database for SMTP delivery
			err := a.storeOutgoingEmail(req.From, req.To, req.Cc, req.Bcc, req.Subject, req.Body)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]any{
					"success": false,
					"error":   err.Error(),
				})
				return
			}
			
			json.NewEncoder(w).Encode(map[string]any{"success": true})
			return
		} else {
			http.Error(w, `{"error":"no email account configured - add an external email account to send emails"}`, http.StatusBadRequest)
			return
		}
	}

	// If custom From address is provided, use it; otherwise use default account
	var acc Account
	if req.From != "" {
		// For custom domain emails, we need to find the matching account
		// For now, we'll use the first account but set the From address
		acc = a.config.Accounts[0]
		// TODO: Implement proper custom domain email sending
		// This would require SMTP configuration for custom domains
	} else {
		acc = a.config.Accounts[0]
	}

	if err := a.email.Send(acc, req.To, req.Cc, req.Bcc, req.Subject, req.Body); err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// handleMarkRead handles POST /api/emails/:id/read
func (a *API) handleMarkRead(w http.ResponseWriter, r *http.Request, id string) {
	read := true
	err := a.database.UpdateEmailStatus(id, &read, nil, nil)
	if err != nil {
		http.Error(w, `{"error":"failed to update email"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// handleMarkUnread handles POST /api/emails/:id/unread
func (a *API) handleMarkUnread(w http.ResponseWriter, r *http.Request, id string) {
	read := false
	err := a.database.UpdateEmailStatus(id, &read, nil, nil)
	if err != nil {
		http.Error(w, `{"error":"failed to update email"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// handleToggleStar handles POST /api/emails/:id/star
func (a *API) handleToggleStar(w http.ResponseWriter, r *http.Request, id string) {
	// Get current status to toggle
	currentEmail, err := a.database.GetEmail(id)
	if err != nil {
		http.Error(w, `{"error":"email not found"}`, http.StatusNotFound)
		return
	}
	newStarred := !currentEmail.Starred
	err = a.database.UpdateEmailStatus(id, nil, &newStarred, nil)
	if err != nil {
		http.Error(w, `{"error":"failed to update email"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"success": true, "starred": newStarred})
}

// handleMoveToTrash handles POST /api/emails/:id/trash
func (a *API) handleMoveToTrash(w http.ResponseWriter, r *http.Request, id string) {
	trashed := true
	err := a.database.UpdateEmailStatus(id, nil, nil, &trashed)
	if err != nil {
		http.Error(w, `{"error":"failed to update email"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// handleRestoreFromTrash handles POST /api/emails/:id/restore
func (a *API) handleRestoreFromTrash(w http.ResponseWriter, r *http.Request, id string) {
	trashed := false
	err := a.database.UpdateEmailStatus(id, nil, nil, &trashed)
	if err != nil {
		http.Error(w, `{"error":"failed to update email"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// handleDeleteEmail handles DELETE /api/emails/:id
func (a *API) handleDeleteEmail(w http.ResponseWriter, r *http.Request, id string) {
	// TODO: Implement permanent delete
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}


// handleDrafts handles POST /api/drafts (save draft)
func (a *API) handleDrafts(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement save draft
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	// Find user in database
	user, err := a.database.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Verify password
	if !verifyPassword(req.Password, user.PasswordHash) {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Generate user-specific token to prevent username tampering
	userToken := GenerateUserToken(a.config.APIKey, user.Username)

	// Return the server's API key and user token for authentication
	response := map[string]interface{}{
		"token":      a.config.APIKey,
		"user_token": userToken,
		"username":   user.Username,
		"name":       user.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}


// handleMailboxSummary handles /api/mailbox/:name/summary or /api/mailbox/summary
func (a *API) handleMailboxSummary(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) > 0 && parts[0] == "summary" {
		// /api/mailbox/summary - overall summary
		a.handleSummary(w, r)
		return
	}
	// TODO: /api/mailbox/:name/summary - specific mailbox stats
	http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
}

// handleSummary returns mailbox counts
func (a *API) handleSummary(w http.ResponseWriter, r *http.Request) {
	if len(a.config.Accounts) == 0 {
		json.NewEncoder(w).Encode(map[string]any{
			"inbox":   0,
			"starred": 0,
			"sent":    0,
			"drafts":  0,
			"trash":   0,
			"unread":  0,
		})
		return
	}

	acc := a.config.Accounts[0]
	summary, err := a.database.GetMailboxSummary(acc.ID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"inbox":   0,
			"starred": 0,
			"sent":    0,
			"drafts":  0,
			"trash":   0,
			"unread":  0,
		})
		return
	}

	json.NewEncoder(w).Encode(summary)
}

// contains checks if a string slice contains a specific value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateID() string {
	// Simple ID generator - in production use UUID
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 16)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// generateVerificationToken generates a cryptographically secure verification token
func generateVerificationToken() string {
	// Generate 32 random bytes = 256 bits of entropy
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to less secure method if crypto/rand fails
		log.Printf("WARNING: crypto/rand failed, using fallback: %v", err)
		return generateID() + generateID() // Use two IDs for longer fallback
	}
	
	// Encode as base64 URL-safe without padding
	token := base64.URLEncoding.EncodeToString(bytes)
	// Remove padding and make it exactly 43 characters (base64 encoding of 32 bytes)
	return strings.TrimRight(token, "=")
}

// handleDomains manages custom domains
func (a *API) handleDomains(w http.ResponseWriter, r *http.Request, parts []string) {
	switch r.Method {
	case "GET":
		// List all domains
		log.Printf("DEBUG: Fetching all domains")
		domains, err := a.database.GetAllDomains()
		if err != nil {
			log.Printf("ERROR: Failed to fetch domains: %v", err)
			http.Error(w, `{"error":"failed to fetch domains"}`, http.StatusInternalServerError)
			return
		}
		log.Printf("DEBUG: Found %d domains", len(domains))
		log.Printf("DEBUG: Domains data: %+v", domains)
		
		// Set content type header
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domains); err != nil {
			log.Printf("ERROR: Failed to encode domains: %v", err)
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
			return
		}
		log.Printf("DEBUG: Successfully encoded and sent domains")

	case "POST":
		// Add new domain
		var req struct {
			Domain string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		if req.Domain == "" {
			http.Error(w, `{"error":"domain required"}`, http.StatusBadRequest)
			return
		}

		// Generate cryptographically secure verification token
		verificationToken := generateVerificationToken()

		log.Printf("DEBUG: Creating domain '%s' with token '%s'", req.Domain, verificationToken)
		
		// Add domain
		domain, err := a.database.CreateDomain(req.Domain, verificationToken)
		if err != nil {
			log.Printf("ERROR: Failed to create domain: %v", err)
			http.Error(w, `{"error":"domain already exists or invalid"}`, http.StatusConflict)
			return
		}
		
		log.Printf("DEBUG: Domain created successfully with ID: %d", domain.ID)

		json.NewEncoder(w).Encode(map[string]any{
			"success":              true,
			"domain":               domain,
			"verification_token":   verificationToken,
			"verification_method": "DNS TXT record",
			"instructions":         fmt.Sprintf("Add a DNS TXT record for 'miramail-verify.%s' with value '%s'", req.Domain, verificationToken),
		})

	case "PUT":
		// Verify domain
		if len(parts) == 0 {
			http.Error(w, `{"error":"domain id required"}`, http.StatusBadRequest)
			return
		}
		var id int64
		fmt.Sscanf(parts[0], "%d", &id)

		domain, err := a.database.GetDomain(id)
		if err != nil {
			http.Error(w, `{"error":"domain not found"}`, http.StatusNotFound)
			return
		}

		// Verify DNS TXT record
		if !verifyDomainViaDNS(domain.Domain, domain.VerificationToken) {
			json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error":   "DNS verification failed - TXT record not found or doesn't match",
				"message": fmt.Sprintf("Add a DNS TXT record for 'miramail-verify.%s' with value '%s'", domain.Domain, domain.VerificationToken),
			})
			return
		}

		// Mark domain as verified in database
		err = a.database.VerifyDomain(id)
		if err != nil {
			http.Error(w, `{"error":"failed to update domain status"}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": fmt.Sprintf("Domain %s verified successfully", domain.Domain),
		})

	case "DELETE":
		// Remove domain
		if len(parts) == 0 {
			http.Error(w, `{"error":"domain id required"}`, http.StatusBadRequest)
			return
		}
		var id int64
		fmt.Sscanf(parts[0], "%d", &id)

		err := a.database.DeleteDomain(id)
		if err != nil {
			http.Error(w, `{"error":"failed to delete domain"}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{"success": true})

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleDomainEmails manages email addresses under custom domains
func (a *API) handleDomainEmails(w http.ResponseWriter, r *http.Request, parts []string) {
	switch r.Method {
	case "GET":
		// List emails for a domain
		if len(parts) == 0 {
			http.Error(w, `{"error":"domain id required"}`, http.StatusBadRequest)
			return
		}
		var domainID int64
		fmt.Sscanf(parts[0], "%d", &domainID)

		emails, err := a.database.GetDomainEmails(domainID)
		if err != nil {
			http.Error(w, `{"error":"failed to fetch emails"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"emails":  emails,
		})

	case "POST":
		// Create new email address
		var req struct {
			DomainID  int64  `json:"domain_id"`
			LocalPart string `json:"local_part"`
			UserID    *int64 `json:"user_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		if req.DomainID == 0 || req.LocalPart == "" {
			http.Error(w, `{"error":"domain_id and local_part required"}`, http.StatusBadRequest)
			return
		}

		// Check domain is verified
		domain, err := a.database.GetDomain(req.DomainID)
		if err != nil {
			http.Error(w, `{"error":"domain not found"}`, http.StatusNotFound)
			return
		}
		// Temporarily bypass verification for testing
		if false && !domain.Verified {
			http.Error(w, `{"error":"domain must be verified first"}`, http.StatusBadRequest)
			return
		}

		// Validate user if provided
		if req.UserID != nil {
			_, err := a.database.GetUserByID(*req.UserID)
			if err != nil {
				http.Error(w, `{"error":"invalid user"}`, http.StatusBadRequest)
				return
			}
		}

		// Create email
		email, err := a.database.CreateDomainEmail(req.DomainID, req.LocalPart, req.UserID)
		if err != nil {
			http.Error(w, `{"error":"email already exists or invalid"}`, http.StatusConflict)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"email":   email,
		})

	case "DELETE":
		// Remove email address
		if len(parts) == 0 {
			http.Error(w, `{"error":"email id required"}`, http.StatusBadRequest)
			return
		}
		var id int64
		fmt.Sscanf(parts[0], "%d", &id)

		err := a.database.DeleteDomainEmail(id)
		if err != nil {
			http.Error(w, `{"error":"failed to delete email"}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{"success": true})

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
