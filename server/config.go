package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Config stores server configuration (minimal - only API key and accounts in JSON)
type Config struct {
	APIKey   string    `json:"api_key"`
	Accounts []Account `json:"accounts"`
}

// InternalEmail represents emails between local users
type InternalEmail struct {
	ID      string   `json:"id"`
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Date    string   `json:"date"`
	Read    bool     `json:"read"`
	Starred bool     `json:"starred"`
	Labels  []string `json:"labels"`
}

// User represents a user account
type User struct {
	Username     string `json:"username"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

// hashPassword creates a SHA-256 hash of the password
func hashPassword(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

// verifyPassword checks if the provided password matches the stored hash
func verifyPassword(password, hash string) bool {
	return hashPassword(password) == hash
}

// Account represents an email account
type Account struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	IMAPServer string `json:"imap_server"`
	IMAPPort   int    `json:"imap_port"`
	SMTPServer string `json:"smtp_server"`
	SMTPPort   int    `json:"smtp_port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	UseTLS     bool   `json:"use_tls"`
}

// LoadConfig loads or creates config with generated API key
func LoadConfig(dataDir string) (*Config, error) {
	configPath := filepath.Join(dataDir, "config.json")

	config := &Config{}

	// Try to load existing config
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
		return config, nil
	}

	// Generate new API key on first run
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	config.APIKey = apiKey

	// Save config
	if err := config.Save(dataDir); err != nil {
		return nil, err
	}

	return config, nil
}

// Save writes config to disk
func (c *Config) Save(dataDir string) error {
	configPath := filepath.Join(dataDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}

// generateAPIKey creates a random 32-byte hex key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
