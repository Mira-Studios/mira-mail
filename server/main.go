package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// downloadWebsite downloads and extracts the website files from GitHub
func downloadSite(siteDir string) error {
	// GitHub raw URL for the main branch website dist folder
	// Using a zipball approach for simplicity
	githubURL := "https://github.com/Mira-Studios/mira-mail/archive/refs/heads/main.zip"
	
	log.Printf("Downloading site files from GitHub...")
	
	// Create temp file for download
	tempFile, err := os.CreateTemp("", "miramail-site-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	
	// Download the zip
	log.Printf("Fetching: %s", githubURL)
	resp, err := http.Get(githubURL)
	if err != nil {
		return fmt.Errorf("failed to download from GitHub: %w", err)
	}
	defer resp.Body.Close()
	
	log.Printf("GitHub response status: %d", resp.StatusCode)
	
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("GitHub returned 404 - repo or branch not found (check URL: %s)", githubURL)
	}
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub returned status: %d", resp.StatusCode)
	}
	
	// Write to temp file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}
	tempFile.Close()
	
	// Open and extract the zip
	zipReader, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer zipReader.Close()
	
	// Create site directory
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		return fmt.Errorf("failed to create site dir: %w", err)
	}
	
	// Extract files from mira-mail/website/dist/ folder in the zip
	prefix := "mira-mail-main/website/dist/"
	for _, file := range zipReader.File {
		if !strings.HasPrefix(file.Name, prefix) {
			continue
		}
		
		// Get relative path
		relPath := strings.TrimPrefix(file.Name, prefix)
		if relPath == "" {
			continue
		}
		
		targetPath := filepath.Join(siteDir, relPath)
		
		if file.FileInfo().IsDir() {
			os.MkdirAll(targetPath, file.Mode())
			continue
		}
		
		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create dir: %w", err)
		}
		
		// Extract file
		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip entry: %w", err)
		}
		
		dstFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}
		
		_, err = io.Copy(dstFile, srcFile)
		dstFile.Close()
		srcFile.Close()
		
		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}
	}
	
	log.Printf("Site files downloaded and extracted successfully")
	return nil
}

// serveIndexHTML serves index.html with injected server configuration
func serveIndexHTML(w http.ResponseWriter, dataDir, apiKey string) {
	indexPath := filepath.Join(dataDir, "site", "index.html")
	
	// Read the original HTML
	content, err := os.ReadFile(indexPath)
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		return
	}
	
	// Create server config script to inject
	serverConfig := fmt.Sprintf(`<script>window.__SERVER_CONFIG__={url:window.location.origin,token:%q};</script>`, apiKey)
	
	// Inject before </head> tag
	html := string(content)
	html = strings.Replace(html, "</head>", serverConfig+"</head>", 1)
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func main() {
	// Parse command line flags
	var port = flag.Int("port", 8080, "Port to run the server on")
	var dataDir = flag.String("data", ".", "Directory to store data")
	var skipDownload = flag.Bool("skip-download", false, "Skip downloading site files from GitHub")
	flag.Parse()

	// Get executable directory for data storage
	if *dataDir == "." {
		if exePath, err := os.Executable(); err == nil {
			*dataDir = filepath.Dir(exePath)
		}
	}
	
	// Download site files if they don't exist
	siteDir := filepath.Join(*dataDir, "site")
	if !*skipDownload {
		// Check if site files exist
		indexPath := filepath.Join(siteDir, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			if err := downloadSite(siteDir); err != nil {
				log.Printf("Warning: Failed to download site files: %v", err)
				log.Printf("Server will use embedded fallback or require manual site setup")
			}
		} else {
			log.Printf("Site files already exist, skipping download")
		}
	}

	// Load configuration (using extended struct to handle migration from old format)
	type configWithUsers struct {
		APIKey   string       `json:"api_key"`
		Accounts []Account    `json:"accounts"`
		Users    []User       `json:"users"`
	}
	
	configPath := filepath.Join(*dataDir, "config.json")
	var oldConfig configWithUsers
	configData, readErr := os.ReadFile(configPath)
	if readErr == nil {
		json.Unmarshal(configData, &oldConfig)
	}
	
	config, err := LoadConfig(*dataDir)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		// Create default config if none exists
		config = &Config{
			Accounts: []Account{},
		}
		if err := config.Save(*dataDir); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
	}

	// Initialize database
	database, err := NewDatabase(*dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Migrate users from old config to database (if any exist)
	if len(oldConfig.Users) > 0 {
		log.Printf("Migrating %d user(s) from config to database...", len(oldConfig.Users))
		for _, user := range oldConfig.Users {
			// Check if user already exists in DB
			existing, _ := database.GetUserByUsername(user.Username)
			if existing == nil {
				if err := database.CreateUser(user.Username, user.Name, user.Email, user.PasswordHash); err != nil {
					log.Printf("Failed to migrate user %s: %v", user.Username, err)
				} else {
					log.Printf("Migrated user: %s", user.Username)
				}
			} else {
				log.Printf("User %s already exists in database, skipping", user.Username)
			}
		}
		// Config is already saved without users by LoadConfig, migration complete
		log.Printf("User migration complete. Users are now stored in database only.")
	}

	// Create API instance
	api := NewAPI(config, database, *dataDir)

	// Setup CORS middleware
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Username, X-User-Token")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Serve static files from site directory
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			serveIndexHTML(w, *dataDir, config.APIKey)
			return
		}

		// Serve other static files
		staticPath := filepath.Join(*dataDir, "site")
		if _, err := os.Stat(filepath.Join(staticPath, r.URL.Path)); err == nil {
			http.ServeFile(w, r, filepath.Join(staticPath, r.URL.Path))
			return
		}

		// API routes
		if r.URL.Path == "/api/login" || r.URL.Path == "/api/health" {
			api.Handler().ServeHTTP(w, r)
			return
		}

		// All other API routes require auth
		if len(r.URL.Path) > 4 && r.URL.Path[:4] == "/api" {
			api.Handler().ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA
		http.ServeFile(w, r, filepath.Join(*dataDir, "site", "index.html"))
	}

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting Mira Mail server on %s", addr)
	log.Printf("Data directory: %s", *dataDir)
	log.Printf("Open http://localhost:%d in your browser", *port)

	if err := http.ListenAndServe(addr, http.HandlerFunc(handler)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
