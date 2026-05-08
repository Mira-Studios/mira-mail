package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

// isValidCustomDomainEmail checks if the email is from a custom domain we manage
func (a *API) isValidCustomDomainEmail(email string) bool {
	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := parts[1]

	// Check if this domain exists in our database
	_, err := a.database.GetDomainByName(domain)
	return err == nil
}

// storeOutgoingEmail stores an outgoing email in the database
func (a *API) storeOutgoingEmail(from string, to, cc, bcc []string, subject, body string) error {
	// Extract domain from the from email
	parts := strings.Split(from, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid from email address")
	}
	domain := parts[1]
	
	// Get domain ID
	domainInfo, err := a.database.GetDomainByName(domain)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}
	
	// Get user ID from the domain email
	domainEmail, err := a.database.GetDomainEmailByAddress(parts[0], domain)
	if err != nil {
		return fmt.Errorf("domain email not found: %w", err)
	}
	
	// Store the email in database
	var userID int64 = 0
	if domainEmail.UserID != nil {
		userID = *domainEmail.UserID
	}
	err = a.database.StoreCustomDomainEmail(from, to, cc, bcc, subject, body, "sent", domainInfo.ID, userID)
	if err != nil {
		return fmt.Errorf("failed to store email: %w", err)
	}
	
	log.Printf("OUTGOING EMAIL: From=%s To=%v Cc=%v Bcc=%v Subject=%s (stored in database)", from, to, cc, bcc, subject)
	return nil
}

// verifyDomainMX checks and updates MX configuration status for a domain
func (a *API) verifyDomainMX(domainID int64, domain string) bool {
	// Check MX records
	mxConfigured := verifyMXRecord(domain)
	
	// Update database with MX status
	err := a.database.UpdateDomainMXStatus(domainID, mxConfigured)
	if err != nil {
		log.Printf("Failed to update MX status for domain %s: %v", domain, err)
	}
	
	return mxConfigured
}

// verifyMXRecord checks if a domain has proper MX records configured
func verifyMXRecord(domain string) bool {
	// Look up MX records for domain
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		log.Printf("MX record lookup failed for %s: %v", domain, err)
		return false
	}
	
	if len(mxRecords) == 0 {
		log.Printf("No MX records found for %s", domain)
		return false
	}
	
	log.Printf("Found MX records for %s: %d records", domain, len(mxRecords))
	for _, mx := range mxRecords {
		log.Printf("  MX: %s (Priority: %d)", mx.Host, mx.Pref)
	}
	
	return true
}

// handleVerifyMX handles MX record verification for a domain
func (a *API) handleVerifyMX(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var req struct {
			DomainID int64  `json:"domain_id"`
			Domain   string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		if req.DomainID == 0 || req.Domain == "" {
			http.Error(w, `{"error":"domain_id and domain required"}`, http.StatusBadRequest)
			return
		}

		// Verify MX records
		mxConfigured := a.verifyDomainMX(req.DomainID, req.Domain)

		json.NewEncoder(w).Encode(map[string]any{
			"success":       true,
			"mx_configured": mxConfigured,
		})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
