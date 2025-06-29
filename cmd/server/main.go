package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yourdatasucks/lettersmith/internal/ai"
	"github.com/yourdatasucks/lettersmith/internal/config"
	"github.com/yourdatasucks/lettersmith/internal/email"
	"github.com/yourdatasucks/lettersmith/internal/geocoding"
	"github.com/yourdatasucks/lettersmith/internal/reps"

	_ "github.com/lib/pq"
)

var geocoderInstance *geocoding.ZipGeocoder

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	dbURL := cfg.DatabaseURL()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	log.Println("Connected to database successfully")

	log.Println("Running database migrations...")
	if err := runMigrations(db); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
		log.Println("Some features may not work without proper schema")
	} else {
		log.Println("Database migrations completed successfully")
	}

	geocodingConfig := &geocoding.GeocodingConfig{
		CustomCensusBureauURL: cfg.CensusBureauURL,
	}
	geocoder := geocoding.NewZipGeocoderWithConfig(db, geocodingConfig)
	if cfg.ZipDataUpdate {
		log.Println("Initializing ZIP code geocoding data...")
		if err := geocoder.LoadZipData(); err != nil {
			log.Printf("Warning: Failed to load ZIP code data: %v", err)
			log.Println("OpenStates API calls may fail without ZIP coordinate data")
		}
	}

	geocoderInstance = geocoder

	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "lettersmith",
		})
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetConfig(w, r, cfg)
		case http.MethodPost:
			handleUpdateConfig(w, r, cfg)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/config/test-email", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleTestEmail(w, r, cfg)
	})

	mux.HandleFunc("/api/config/debug", func(w http.ResponseWriter, r *http.Request) {
		handleConfigDebug(w, r, cfg)
	})

	mux.HandleFunc("/api/db/debug", func(w http.ResponseWriter, r *http.Request) {
		handleDatabaseDebug(w, r, db)
	})

	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleSystemStatus(w, r, cfg, db)
	})

	mux.HandleFunc("/api/representatives", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetRepresentatives(w, r, db)
		case http.MethodPost:
			handleSyncRepresentatives(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/representatives/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut || r.Method == http.MethodDelete {
			handleRepresentativeByID(w, r, db)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/test/representatives", func(w http.ResponseWriter, r *http.Request) {
		handleTestRepresentatives(w, r, db)
	})

	mux.HandleFunc("/api/letters/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleGenerateLetter(w, r, cfg, db)
	})

	// Serve static files from the web directory
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("web/css"))))
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("web/js"))))
	mux.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("web/html"))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "web/html/index.html")
			return
		}
		http.ServeFile(w, r, "web/html"+r.URL.Path)
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting Lettersmith server on %s...", addr)
	log.Printf("Configuration loaded from environment variables")
	log.Printf("AI Provider: %s, Email Provider: %s", cfg.AI.Provider, cfg.Email.Provider)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleGetConfig(w http.ResponseWriter, _ *http.Request, cfg *config.Config) {

	envValues := readEnvFile()

	freshCfg := &config.Config{}

	if method := envValues["LETTER_GENERATION_METHOD"]; method != "" {
		freshCfg.Letter.GenerationMethod = method
	} else {
		freshCfg.Letter.GenerationMethod = "ai"
	}

	if tone := envValues["LETTER_TONE"]; tone != "" {
		freshCfg.Letter.Tone = tone
	} else {
		freshCfg.Letter.Tone = "professional"
	}

	if maxLen := envValues["LETTER_MAX_LENGTH"]; maxLen != "" {
		if length, err := strconv.Atoi(maxLen); err == nil {
			freshCfg.Letter.MaxLength = length
		} else {
			freshCfg.Letter.MaxLength = 500
		}
	} else {
		freshCfg.Letter.MaxLength = 500
	}

	if freshCfg.Letter.GenerationMethod == "templates" {
		freshCfg.Letter.TemplateConfig = &config.TemplateConfig{
			Directory:        envValues["TEMPLATE_DIRECTORY"],
			RotationStrategy: envValues["TEMPLATE_ROTATION_STRATEGY"],
			Personalize:      envValues["TEMPLATE_PERSONALIZE"] == "true",
		}
		if freshCfg.Letter.TemplateConfig.Directory == "" {
			freshCfg.Letter.TemplateConfig.Directory = "templates/"
		}
		if freshCfg.Letter.TemplateConfig.RotationStrategy == "" {
			freshCfg.Letter.TemplateConfig.RotationStrategy = "random-unique"
		}
	}

	if provider := envValues["AI_PROVIDER"]; provider != "" {
		freshCfg.AI.Provider = provider
		if provider == "openai" {
			freshCfg.AI.OpenAI.APIKey = envValues["OPENAI_API_KEY"]
			freshCfg.AI.OpenAI.Model = envValues["OPENAI_MODEL"]
			if freshCfg.AI.OpenAI.Model == "" {
				freshCfg.AI.OpenAI.Model = "gpt-4"
			}
		} else if provider == "anthropic" {
			freshCfg.AI.Anthropic.APIKey = envValues["ANTHROPIC_API_KEY"]
			freshCfg.AI.Anthropic.Model = envValues["ANTHROPIC_MODEL"]
			if freshCfg.AI.Anthropic.Model == "" {
				freshCfg.AI.Anthropic.Model = "claude-3-sonnet-20240229"
			}
		}
	}

	currentCfg := freshCfg

	if currentCfg.User.Name == "" {
		currentCfg.User = cfg.User
	}
	if currentCfg.Email.Provider == "" {
		currentCfg.Email = cfg.Email
	}
	if currentCfg.Representatives.OpenStatesAPIKey == "" {
		currentCfg.Representatives = cfg.Representatives
	}
	if currentCfg.Scheduler.SendTime == "" {
		currentCfg.Scheduler = cfg.Scheduler
	}

	safeConfig := map[string]interface{}{
		"server": currentCfg.Server,
		"ai": map[string]interface{}{
			"provider": currentCfg.AI.Provider,
			"openai": map[string]interface{}{
				"model":      currentCfg.AI.OpenAI.Model,
				"configured": currentCfg.AI.OpenAI.APIKey != "",
			},
			"anthropic": map[string]interface{}{
				"model":      currentCfg.AI.Anthropic.Model,
				"configured": currentCfg.AI.Anthropic.APIKey != "",
			},
		},
		"email": map[string]interface{}{
			"provider": currentCfg.Email.Provider,
			"smtp": map[string]interface{}{
				"host":       currentCfg.Email.SMTP.Host,
				"port":       currentCfg.Email.SMTP.Port,
				"username":   currentCfg.Email.SMTP.Username,
				"configured": currentCfg.Email.SMTP.Password != "",
			},
			"sendgrid": map[string]interface{}{
				"configured": currentCfg.Email.SendGrid.APIKey != "",
			},
			"mailgun": map[string]interface{}{
				"domain":     currentCfg.Email.Mailgun.Domain,
				"configured": currentCfg.Email.Mailgun.APIKey != "",
			},
		},
		"representatives": map[string]interface{}{
			"openstates_configured": currentCfg.Representatives.OpenStatesAPIKey != "",
		},
		"user":       currentCfg.User,
		"scheduler":  currentCfg.Scheduler,
		"letter":     currentCfg.Letter,
		"env_values": sanitizeEnvValues(envValues),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(safeConfig)
}

func handleUpdateConfig(w http.ResponseWriter, r *http.Request, _ *config.Config) {
	w.Header().Set("Content-Type", "application/json")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON format",
		})
		return
	}

	var dbConn *sql.DB
	if cfg, err := config.Load(); err == nil {
		dbURL := cfg.DatabaseURL()
		if db, err := sql.Open("postgres", dbURL); err == nil {
			dbConn = db
			defer db.Close()
		}
	}

	if err := updateEnvFile(updates, dbConn); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to update configuration: %s", err.Error()),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "Configuration updated successfully",
	})
}

func handleConfigDebug(w http.ResponseWriter, _ *http.Request, cfg *config.Config) {
	w.Header().Set("Content-Type", "application/json")

	freshCfg, err := config.Load()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to load configuration: %v", err),
		})
		return
	}

	envValues := readEnvFile()

	debugInfo := map[string]interface{}{
		"environment_variables": map[string]string{
			"POSTGRES_USER":              getEnvFileStatus(envValues, "POSTGRES_USER"),
			"POSTGRES_PASSWORD":          getEnvFileStatus(envValues, "POSTGRES_PASSWORD"),
			"POSTGRES_DB":                getEnvFileStatus(envValues, "POSTGRES_DB"),
			"POSTGRES_PORT":              getEnvFileStatus(envValues, "POSTGRES_PORT"),
			"DATABASE_URL":               getEnvFileStatus(envValues, "DATABASE_URL"),
			"ZIP_DATA_UPDATE":            getEnvFileStatus(envValues, "ZIP_DATA_UPDATE"),
			"PORT":                       getEnvFileStatus(envValues, "PORT"),
			"AI_PROVIDER":                getEnvFileStatus(envValues, "AI_PROVIDER"),
			"OPENAI_API_KEY":             getEnvFileStatus(envValues, "OPENAI_API_KEY"),
			"ANTHROPIC_API_KEY":          getEnvFileStatus(envValues, "ANTHROPIC_API_KEY"),
			"EMAIL_PROVIDER":             getEnvFileStatus(envValues, "EMAIL_PROVIDER"),
			"SMTP_HOST":                  getEnvFileStatus(envValues, "SMTP_HOST"),
			"SMTP_PORT":                  getEnvFileStatus(envValues, "SMTP_PORT"),
			"SMTP_USERNAME":              getEnvFileStatus(envValues, "SMTP_USERNAME"),
			"SMTP_PASSWORD":              getEnvFileStatus(envValues, "SMTP_PASSWORD"),
			"SMTP_FROM":                  getEnvFileStatus(envValues, "SMTP_FROM"),
			"SENDGRID_API_KEY":           getEnvFileStatus(envValues, "SENDGRID_API_KEY"),
			"MAILGUN_API_KEY":            getEnvFileStatus(envValues, "MAILGUN_API_KEY"),
			"MAILGUN_DOMAIN":             getEnvFileStatus(envValues, "MAILGUN_DOMAIN"),
			"OPENSTATES_API_KEY":         getEnvFileStatus(envValues, "OPENSTATES_API_KEY"),
			"USER_NAME":                  getEnvFileStatus(envValues, "USER_NAME"),
			"USER_EMAIL":                 getEnvFileStatus(envValues, "USER_EMAIL"),
			"USER_ZIP_CODE":              getEnvFileStatus(envValues, "USER_ZIP_CODE"),
			"SEND_COPY_TO_SELF":          getEnvFileStatus(envValues, "SEND_COPY_TO_SELF"),
			"SCHEDULER_SEND_TIME":        getEnvFileStatus(envValues, "SCHEDULER_SEND_TIME"),
			"SCHEDULER_TIMEZONE":         getEnvFileStatus(envValues, "SCHEDULER_TIMEZONE"),
			"SCHEDULER_ENABLED":          getEnvFileStatus(envValues, "SCHEDULER_ENABLED"),
			"LETTER_TONE":                getEnvFileStatus(envValues, "LETTER_TONE"),
			"LETTER_MAX_LENGTH":          getEnvFileStatus(envValues, "LETTER_MAX_LENGTH"),
			"LETTER_GENERATION_METHOD":   getEnvFileStatus(envValues, "LETTER_GENERATION_METHOD"),
			"LETTER_THEMES":              getEnvFileStatus(envValues, "LETTER_THEMES"),
			"TEMPLATE_DIRECTORY":         getEnvFileStatus(envValues, "TEMPLATE_DIRECTORY"),
			"TEMPLATE_ROTATION_STRATEGY": getEnvFileStatus(envValues, "TEMPLATE_ROTATION_STRATEGY"),
			"TEMPLATE_PERSONALIZE":       getEnvFileStatus(envValues, "TEMPLATE_PERSONALIZE"),
			"DOCKER_IMAGE":               getEnvFileStatus(envValues, "DOCKER_IMAGE"),
			"CENSUS_BUREAU_URL":          getEnvFileStatus(envValues, "CENSUS_BUREAU_URL"),
		},
		"configuration_status": map[string]interface{}{
			"user_configured":            freshCfg.User.Name != "" && freshCfg.User.Email != "" && freshCfg.User.ZipCode != "",
			"generation_method":          freshCfg.Letter.GenerationMethod,
			"ai_configured":              freshCfg.Letter.GenerationMethod == "ai" && freshCfg.AI.Provider != "" && isAIConfigured(freshCfg),
			"templates_configured":       freshCfg.Letter.GenerationMethod == "templates" && isTemplatesConfigured(freshCfg),
			"email_configured":           freshCfg.Email.Provider != "" && isEmailConfigured(freshCfg),
			"representatives_configured": isRepresentativesConfigured(freshCfg),
			"validation_result":          getValidationResult(freshCfg),
		},
		"database_url_parsed": freshCfg.DatabaseURL(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(debugInfo)
}

func isAIConfigured(cfg *config.Config) bool {

	if cfg.Letter.GenerationMethod != "ai" {
		return false
	}

	switch cfg.AI.Provider {
	case "openai":
		return cfg.AI.OpenAI.APIKey != ""
	case "anthropic":
		return cfg.AI.Anthropic.APIKey != ""
	default:
		return false
	}
}

func isEmailConfigured(cfg *config.Config) bool {
	switch cfg.Email.Provider {
	case "smtp":
		return cfg.Email.SMTP.Host != "" && cfg.Email.SMTP.Port != 0 && cfg.Email.SMTP.Password != ""
	case "sendgrid":
		return cfg.Email.SendGrid.APIKey != ""
	case "mailgun":
		return cfg.Email.Mailgun.APIKey != "" && cfg.Email.Mailgun.Domain != ""
	default:
		return false
	}
}

func isTemplatesConfigured(cfg *config.Config) bool {
	if cfg.Letter.GenerationMethod != "templates" {
		return false
	}

	if cfg.Letter.TemplateConfig == nil || cfg.Letter.TemplateConfig.Directory == "" {
		return false
	}

	return true
}

func isRepresentativesConfigured(cfg *config.Config) bool {
	return cfg.Representatives.OpenStatesAPIKey != ""
}

func getValidationResult(cfg *config.Config) string {

	if cfg.User.Name == "" || cfg.User.Email == "" || cfg.User.ZipCode == "" {
		return "missing required user information"
	}
	if cfg.AI.Provider == "" && cfg.Letter.GenerationMethod == "ai" {
		return "AI provider required for AI generation"
	}
	if cfg.Email.Provider == "" {
		return "email provider required"
	}
	return "valid"
}

func readEnvFile() map[string]string {
	envValues := make(map[string]string)

	data, err := os.ReadFile(".env")
	if err != nil {
		return envValues
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if len(value) > 1 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			envValues[key] = value
		}
	}

	return envValues
}

func updateEnvFile(updates map[string]interface{}, dbConn *sql.DB) error {
	cwd, _ := os.Getwd()
	log.Printf("Current working directory: %s", cwd)

	if info, err := os.Stat(".env"); err == nil {
		log.Printf(".env file exists with permissions: %v", info.Mode())
	} else {
		log.Printf(".env file does not exist yet: %v", err)
	}

	testFile := ".env_test_write"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		log.Printf("ERROR: Cannot write to current directory: %v", err)
		return fmt.Errorf("cannot write to current directory: %w", err)
	} else {
		os.Remove(testFile)
		log.Printf("Current directory is writable")
	}

	existingEnv := readEnvFile()
	log.Printf("Read %d existing environment variables from .env", len(existingEnv))

	if database, ok := updates["database"].(map[string]interface{}); ok {
		oldPassword := existingEnv["POSTGRES_PASSWORD"]
		oldUser := existingEnv["POSTGRES_USER"]
		if oldUser == "" {
			oldUser = "lettersmith"
		}

		newUser := oldUser
		var newPassword string

		if user, ok := database["user"].(string); ok && user != "" {
			newUser = strings.TrimSpace(user)
			existingEnv["POSTGRES_USER"] = newUser
		}
		if password, ok := database["password"].(string); ok && password != "" {
			newPassword = strings.TrimSpace(password)
			existingEnv["POSTGRES_PASSWORD"] = newPassword

			if oldPassword != "" && oldPassword != newPassword && dbConn != nil {
				log.Printf("Updating database password for user: %s", newUser)
				if err := updateDatabaseUserPassword(dbConn, newUser, newPassword); err != nil {
					log.Printf("Warning: Failed to update database user password: %v", err)
					return fmt.Errorf("failed to update database password: %w", err)
				} else {
					log.Printf("Successfully updated database user password")
				}
			}
		}
		if dbName, ok := database["db"].(string); ok && dbName != "" {
			existingEnv["POSTGRES_DB"] = strings.TrimSpace(dbName)
		}
		if port, ok := database["port"].(float64); ok && port > 0 {
			existingEnv["POSTGRES_PORT"] = strconv.Itoa(int(port))
		}
		if zipDataUpdate, ok := database["zip_data_update"].(bool); ok {
			existingEnv["ZIP_DATA_UPDATE"] = strconv.FormatBool(zipDataUpdate)
		}

		user := existingEnv["POSTGRES_USER"]
		if user == "" {
			user = "lettersmith"
		}
		password := existingEnv["POSTGRES_PASSWORD"]
		if password == "" {
			password = "lettersmith_pass"
		}
		dbName := existingEnv["POSTGRES_DB"]
		if dbName == "" {
			dbName = "lettersmith"
		}
		port := existingEnv["POSTGRES_PORT"]
		if port == "" {
			port = "5432"
		}

		databaseURL := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", user, password, port, dbName)
		existingEnv["DATABASE_URL"] = databaseURL
	}

	if user, ok := updates["user"].(map[string]interface{}); ok {
		if name, ok := user["name"].(string); ok && name != "" {
			existingEnv["USER_NAME"] = strings.TrimSpace(name)
		}
		if email, ok := user["email"].(string); ok && email != "" {
			existingEnv["USER_EMAIL"] = strings.TrimSpace(email)
		}
		if zip, ok := user["zip_code"].(string); ok && zip != "" {
			existingEnv["USER_ZIP_CODE"] = strings.TrimSpace(zip)
		}
		if sendCopy, ok := user["send_copy_to_self"].(bool); ok {
			existingEnv["SEND_COPY_TO_SELF"] = strconv.FormatBool(sendCopy)
		}
	}

	if ai, ok := updates["ai"].(map[string]interface{}); ok {
		if provider, ok := ai["provider"].(string); ok && provider != "" {
			existingEnv["AI_PROVIDER"] = strings.TrimSpace(provider)
		}
		if openai, ok := ai["openai"].(map[string]interface{}); ok {
			if model, ok := openai["model"].(string); ok && model != "" {
				existingEnv["OPENAI_MODEL"] = strings.TrimSpace(model)
			}
			if apiKey, ok := openai["api_key"].(string); ok && apiKey != "" {
				existingEnv["OPENAI_API_KEY"] = strings.TrimSpace(apiKey)
			}
		}
		if anthropic, ok := ai["anthropic"].(map[string]interface{}); ok {
			if model, ok := anthropic["model"].(string); ok && model != "" {
				existingEnv["ANTHROPIC_MODEL"] = strings.TrimSpace(model)
			}
			if apiKey, ok := anthropic["api_key"].(string); ok && apiKey != "" {
				existingEnv["ANTHROPIC_API_KEY"] = strings.TrimSpace(apiKey)
			}
		}
	}

	if email, ok := updates["email"].(map[string]interface{}); ok {

		if provider, ok := email["provider"].(string); ok && provider != "" {
			currentProvider := existingEnv["EMAIL_PROVIDER"]
			if currentProvider != "" && currentProvider != provider {

				switch currentProvider {
				case "smtp":
					delete(existingEnv, "SMTP_HOST")
					delete(existingEnv, "SMTP_PORT")
					delete(existingEnv, "SMTP_USERNAME")
					delete(existingEnv, "SMTP_PASSWORD")
					delete(existingEnv, "SMTP_FROM")
				case "sendgrid":
					delete(existingEnv, "SENDGRID_API_KEY")
					delete(existingEnv, "SENDGRID_FROM")
				case "mailgun":
					delete(existingEnv, "MAILGUN_API_KEY")
					delete(existingEnv, "MAILGUN_DOMAIN")
					delete(existingEnv, "MAILGUN_FROM")
				}
			}
			existingEnv["EMAIL_PROVIDER"] = strings.TrimSpace(provider)
		}

		if smtp, ok := email["smtp"].(map[string]interface{}); ok {
			if host, ok := smtp["host"].(string); ok {
				existingEnv["SMTP_HOST"] = strings.TrimSpace(host)
			}
			if port, ok := smtp["port"].(float64); ok && port > 0 {
				existingEnv["SMTP_PORT"] = strconv.Itoa(int(port))
			}
			if username, ok := smtp["username"].(string); ok {
				existingEnv["SMTP_USERNAME"] = strings.TrimSpace(username)
			}
			if password, ok := smtp["password"].(string); ok && password != "" {
				existingEnv["SMTP_PASSWORD"] = strings.TrimSpace(password)
			}
			if from, ok := smtp["from"].(string); ok {
				existingEnv["SMTP_FROM"] = strings.TrimSpace(from)
			}
		}
		if sendgrid, ok := email["sendgrid"].(map[string]interface{}); ok {
			if apiKey, ok := sendgrid["api_key"].(string); ok && apiKey != "" {
				existingEnv["SENDGRID_API_KEY"] = strings.TrimSpace(apiKey)
			}
			if from, ok := sendgrid["from"].(string); ok && from != "" {
				existingEnv["SENDGRID_FROM"] = strings.TrimSpace(from)
			}
		}
		if mailgun, ok := email["mailgun"].(map[string]interface{}); ok {
			if apiKey, ok := mailgun["api_key"].(string); ok && apiKey != "" {
				existingEnv["MAILGUN_API_KEY"] = strings.TrimSpace(apiKey)
			}
			if domain, ok := mailgun["domain"].(string); ok && domain != "" {
				existingEnv["MAILGUN_DOMAIN"] = strings.TrimSpace(domain)
			}
			if from, ok := mailgun["from"].(string); ok && from != "" {
				existingEnv["MAILGUN_FROM"] = strings.TrimSpace(from)
			}
		}
	}

	if reps, ok := updates["representatives"].(map[string]interface{}); ok {
		if apiKey, ok := reps["openstates_api_key"].(string); ok && apiKey != "" {
			existingEnv["OPENSTATES_API_KEY"] = strings.TrimSpace(apiKey)
		}
	}

	if scheduler, ok := updates["scheduler"].(map[string]interface{}); ok {
		if sendTime, ok := scheduler["send_time"].(string); ok && sendTime != "" {
			existingEnv["SCHEDULER_SEND_TIME"] = strings.TrimSpace(sendTime)
		}
		if timezone, ok := scheduler["timezone"].(string); ok && timezone != "" {
			existingEnv["SCHEDULER_TIMEZONE"] = strings.TrimSpace(timezone)
		}
		if enabled, ok := scheduler["enabled"].(bool); ok {
			existingEnv["SCHEDULER_ENABLED"] = strconv.FormatBool(enabled)
		}
	}

	if letter, ok := updates["letter"].(map[string]interface{}); ok {
		if tone, ok := letter["tone"].(string); ok && tone != "" {
			existingEnv["LETTER_TONE"] = strings.TrimSpace(tone)
		}
		if maxLength, ok := letter["max_length"].(float64); ok && maxLength > 0 {
			existingEnv["LETTER_MAX_LENGTH"] = strconv.Itoa(int(maxLength))
		}
		if method, ok := letter["generation_method"].(string); ok && method != "" {
			existingEnv["LETTER_GENERATION_METHOD"] = strings.TrimSpace(method)

			if method == "templates" {

				delete(existingEnv, "AI_PROVIDER")
				delete(existingEnv, "OPENAI_API_KEY")
				delete(existingEnv, "OPENAI_MODEL")
				delete(existingEnv, "ANTHROPIC_API_KEY")
				delete(existingEnv, "ANTHROPIC_MODEL")
			} else if method == "ai" {

				delete(existingEnv, "TEMPLATE_DIRECTORY")
				delete(existingEnv, "TEMPLATE_ROTATION_STRATEGY")
				delete(existingEnv, "TEMPLATE_PERSONALIZE")
			}
		}
		if themes, ok := letter["themes"].([]interface{}); ok && len(themes) > 0 {
			var themeStrings []string
			for _, theme := range themes {
				if themeStr, ok := theme.(string); ok && themeStr != "" {
					themeStrings = append(themeStrings, strings.TrimSpace(themeStr))
				}
			}
			if len(themeStrings) > 0 {
				existingEnv["LETTER_THEMES"] = strings.Join(themeStrings, ",")
			}
		}

		if templateConfig, ok := letter["template_config"].(map[string]interface{}); ok {
			if directory, ok := templateConfig["directory"].(string); ok && directory != "" {
				existingEnv["TEMPLATE_DIRECTORY"] = strings.TrimSpace(directory)
			}
			if strategy, ok := templateConfig["rotation_strategy"].(string); ok && strategy != "" {
				existingEnv["TEMPLATE_ROTATION_STRATEGY"] = strings.TrimSpace(strategy)
			}
			if personalize, ok := templateConfig["personalize"].(bool); ok {
				existingEnv["TEMPLATE_PERSONALIZE"] = strconv.FormatBool(personalize)
			}
		}
	}

	var envContent strings.Builder
	envContent.WriteString("# Lettersmith Configuration\n")
	envContent.WriteString("# Generated by web UI - edit via http://localhost:8080\n\n")

	writeEnvSection(&envContent, "Database Configuration", map[string]string{
		"POSTGRES_USER":     existingEnv["POSTGRES_USER"],
		"POSTGRES_PASSWORD": existingEnv["POSTGRES_PASSWORD"],
		"POSTGRES_DB":       existingEnv["POSTGRES_DB"],
		"POSTGRES_PORT":     existingEnv["POSTGRES_PORT"],
		"ZIP_DATA_UPDATE":   existingEnv["ZIP_DATA_UPDATE"],
		"DATABASE_URL":      existingEnv["DATABASE_URL"],
	})

	writeEnvSection(&envContent, "Advanced Configuration", map[string]string{
		"DOCKER_IMAGE":      existingEnv["DOCKER_IMAGE"],
		"CENSUS_BUREAU_URL": existingEnv["CENSUS_BUREAU_URL"],
	})

	writeEnvSection(&envContent, "Server", map[string]string{
		"PORT": existingEnv["PORT"],
	})

	writeEnvSection(&envContent, "User Information", map[string]string{
		"USER_NAME":         existingEnv["USER_NAME"],
		"USER_EMAIL":        existingEnv["USER_EMAIL"],
		"USER_ZIP_CODE":     existingEnv["USER_ZIP_CODE"],
		"SEND_COPY_TO_SELF": existingEnv["SEND_COPY_TO_SELF"],
	})

	generationMethod := existingEnv["LETTER_GENERATION_METHOD"]
	if generationMethod == "ai" {
		writeEnvSection(&envContent, "AI Provider", map[string]string{
			"AI_PROVIDER":       existingEnv["AI_PROVIDER"],
			"OPENAI_API_KEY":    existingEnv["OPENAI_API_KEY"],
			"OPENAI_MODEL":      existingEnv["OPENAI_MODEL"],
			"ANTHROPIC_API_KEY": existingEnv["ANTHROPIC_API_KEY"],
			"ANTHROPIC_MODEL":   existingEnv["ANTHROPIC_MODEL"],
		})
	}

	currentEmailProvider := existingEnv["EMAIL_PROVIDER"]
	emailSettings := map[string]string{
		"EMAIL_PROVIDER": existingEnv["EMAIL_PROVIDER"],
	}

	switch currentEmailProvider {
	case "smtp":
		emailSettings["SMTP_HOST"] = existingEnv["SMTP_HOST"]
		emailSettings["SMTP_PORT"] = existingEnv["SMTP_PORT"]
		emailSettings["SMTP_USERNAME"] = existingEnv["SMTP_USERNAME"]
		emailSettings["SMTP_PASSWORD"] = existingEnv["SMTP_PASSWORD"]
		emailSettings["SMTP_FROM"] = existingEnv["SMTP_FROM"]
	case "sendgrid":
		emailSettings["SENDGRID_API_KEY"] = existingEnv["SENDGRID_API_KEY"]
		emailSettings["SENDGRID_FROM"] = existingEnv["SENDGRID_FROM"]
	case "mailgun":
		emailSettings["MAILGUN_API_KEY"] = existingEnv["MAILGUN_API_KEY"]
		emailSettings["MAILGUN_DOMAIN"] = existingEnv["MAILGUN_DOMAIN"]
		emailSettings["MAILGUN_FROM"] = existingEnv["MAILGUN_FROM"]
	}

	writeEnvSection(&envContent, "Email Provider", emailSettings)

	writeEnvSection(&envContent, "Representative APIs", map[string]string{
		"OPENSTATES_API_KEY": existingEnv["OPENSTATES_API_KEY"],
	})

	writeEnvSection(&envContent, "Scheduler", map[string]string{
		"SCHEDULER_SEND_TIME": existingEnv["SCHEDULER_SEND_TIME"],
		"SCHEDULER_TIMEZONE":  existingEnv["SCHEDULER_TIMEZONE"],
		"SCHEDULER_ENABLED":   existingEnv["SCHEDULER_ENABLED"],
	})

	letterSettings := map[string]string{
		"LETTER_GENERATION_METHOD": existingEnv["LETTER_GENERATION_METHOD"],
		"LETTER_THEMES":            existingEnv["LETTER_THEMES"],
	}

	if generationMethod == "ai" {
		letterSettings["LETTER_TONE"] = existingEnv["LETTER_TONE"]
		letterSettings["LETTER_MAX_LENGTH"] = existingEnv["LETTER_MAX_LENGTH"]
	} else if generationMethod == "templates" {
		letterSettings["TEMPLATE_DIRECTORY"] = existingEnv["TEMPLATE_DIRECTORY"]
		letterSettings["TEMPLATE_ROTATION_STRATEGY"] = existingEnv["TEMPLATE_ROTATION_STRATEGY"]
		letterSettings["TEMPLATE_PERSONALIZE"] = existingEnv["TEMPLATE_PERSONALIZE"]
	}

	writeEnvSection(&envContent, "Letter Settings", letterSettings)

	content := envContent.String()
	log.Printf("Generated .env content (%d bytes)", len(content))

	envPath := ".env"
	log.Printf("Attempting to write to: %s", envPath)

	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		log.Printf("ERROR writing .env file: %v", err)
		log.Printf("Error type: %T", err)

		if os.IsPermission(err) {
			return fmt.Errorf("permission denied writing .env file - check directory permissions: %w", err)
		}
		if os.IsExist(err) {
			return fmt.Errorf(".env file exists and cannot be overwritten: %w", err)
		}
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	log.Printf("Successfully wrote .env file")
	return nil
}

func writeEnvSection(builder *strings.Builder, sectionName string, variables map[string]string) {
	hasContent := false
	for _, value := range variables {
		if value != "" {
			hasContent = true
			break
		}
	}

	if hasContent {
		builder.WriteString(fmt.Sprintf("# %s\n", sectionName))
		for key, value := range variables {
			if value != "" {

				if strings.ContainsAny(value, " \t\n\"'\\") {
					value = fmt.Sprintf(`"%s"`, strings.ReplaceAll(value, `"`, `\"`))
				}
				builder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			}
		}
		builder.WriteString("\n")
	}
}

func getEnvFileStatus(envValues map[string]string, key string) string {
	value := envValues[key]
	if value == "" {
		return "not set"
	}
	if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "password") {
		return "set (masked)"
	}
	return value
}

func sanitizeEnvValues(envValues map[string]string) map[string]string {
	sanitized := make(map[string]string)

	for key, value := range envValues {
		if value == "" {
			sanitized[key] = ""
		} else if key == "DATABASE_URL" {
			urlPattern := `(postgres://[^:]+:)([^@]+)(@.+)`
			re := regexp.MustCompile(urlPattern)
			if re.MatchString(value) {
				sanitized[key] = re.ReplaceAllString(value, "${1}••••••••${3}")
			} else {
				sanitized[key] = "••••••••"
			}
		} else if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "password") {
			sanitized[key] = "••••••••"
		} else {
			sanitized[key] = value
		}
	}

	return sanitized
}

func handleTestEmail(w http.ResponseWriter, r *http.Request, _ *config.Config) {
	w.Header().Set("Content-Type", "application/json")

	var reqData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON",
		})
		return
	}

	envValues := readEnvFile()

	emailConfig := &config.EmailConfig{}

	if email, ok := reqData["email"].(map[string]interface{}); ok {
		if provider, ok := email["provider"].(string); ok {
			emailConfig.Provider = strings.TrimSpace(provider)
		}

		if smtp, ok := email["smtp"].(map[string]interface{}); ok {
			if host, ok := smtp["host"].(string); ok {
				emailConfig.SMTP.Host = strings.TrimSpace(host)
			} else {

				emailConfig.SMTP.Host = envValues["SMTP_HOST"]
			}

			if port, ok := smtp["port"].(float64); ok {
				emailConfig.SMTP.Port = int(port)
			} else {

				if portStr := envValues["SMTP_PORT"]; portStr != "" {
					if p, err := strconv.Atoi(portStr); err == nil {
						emailConfig.SMTP.Port = p
					}
				}
			}

			if username, ok := smtp["username"].(string); ok {
				emailConfig.SMTP.Username = strings.TrimSpace(username)
			} else {

				emailConfig.SMTP.Username = envValues["SMTP_USERNAME"]
			}

			if password, ok := smtp["password"].(string); ok && password != "" {
				emailConfig.SMTP.Password = strings.TrimSpace(password)
			} else {

				emailConfig.SMTP.Password = envValues["SMTP_PASSWORD"]
			}

			if from, ok := smtp["from"].(string); ok {
				emailConfig.SMTP.From = strings.TrimSpace(from)
			} else {

				if fromAddr := envValues["SMTP_FROM"]; fromAddr != "" {
					emailConfig.SMTP.From = fromAddr
				} else {
					emailConfig.SMTP.From = emailConfig.SMTP.Username
				}
			}
		}
	}

	if emailConfig.Provider == "" {
		emailConfig.Provider = envValues["EMAIL_PROVIDER"]
		if emailConfig.Provider == "smtp" {
			emailConfig.SMTP.Host = envValues["SMTP_HOST"]
			emailConfig.SMTP.Username = envValues["SMTP_USERNAME"]
			emailConfig.SMTP.Password = envValues["SMTP_PASSWORD"]
			if fromAddr := envValues["SMTP_FROM"]; fromAddr != "" {
				emailConfig.SMTP.From = fromAddr
			} else {
				emailConfig.SMTP.From = emailConfig.SMTP.Username
			}
			if portStr := envValues["SMTP_PORT"]; portStr != "" {
				if p, err := strconv.Atoi(portStr); err == nil {
					emailConfig.SMTP.Port = p
				}
			}
		}
	}

	if emailConfig.Provider != "smtp" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Only SMTP testing is currently supported",
		})
		return
	}

	if emailConfig.SMTP.Host == "" || emailConfig.SMTP.Port == 0 ||
		emailConfig.SMTP.Username == "" || emailConfig.SMTP.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing required SMTP configuration fields",
		})
		return
	}

	var userEmail string
	if user, ok := reqData["user"].(map[string]interface{}); ok {
		if email, ok := user["email"].(string); ok {
			userEmail = strings.TrimSpace(email)
		}
	}

	if userEmail == "" {
		userEmail = envValues["USER_EMAIL"]
	}

	if userEmail == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "User email is required for test",
		})
		return
	}

	client := email.NewClient(emailConfig)

	log.Printf("Testing SMTP connection to %s:%d", emailConfig.SMTP.Host, emailConfig.SMTP.Port)
	if err := client.TestConnection(); err != nil {
		log.Printf("SMTP connection test failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("SMTP connection failed: %s", err.Error()),
		})
		return
	}

	log.Printf("SMTP connection successful, sending test email to %s", userEmail)
	subject := "🧪 Lettersmith Email Test"
	body := `Hello!

This is a test email from Lettersmith to verify your email configuration is working correctly.

If you're reading this, your SMTP settings are properly configured and Lettersmith can successfully send emails.

Configuration tested:
- Provider: SMTP
- Host: ` + emailConfig.SMTP.Host + `
- Port: ` + strconv.Itoa(emailConfig.SMTP.Port) + `
- Username: ` + emailConfig.SMTP.Username + `

Best regards,
The Lettersmith Team

---
This email was sent as part of your email configuration testing.`

	if err := client.SendEmail(userEmail, subject, body); err != nil {
		log.Printf("Test email sending failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Email sending failed: %s", err.Error()),
		})
		return
	}

	log.Printf("Test email sent successfully to %s", userEmail)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Test email sent successfully! Check your inbox.",
		"details": fmt.Sprintf("Email sent to %s via %s:%d", userEmail, emailConfig.SMTP.Host, emailConfig.SMTP.Port),
	})
}

func handleSystemStatus(w http.ResponseWriter, _ *http.Request, cfg *config.Config, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	envValues := readEnvFile()
	status := map[string]interface{}{
		"overall_status":     "healthy",
		"timestamp":          fmt.Sprintf("%v", time.Now().UTC().Format(time.RFC3339)),
		"services":           map[string]interface{}{},
		"missing_components": []string{},
		"summary":            map[string]interface{}{},
	}

	services := status["services"].(map[string]interface{})
	var missingComponents []string
	healthyCount := 0
	totalServices := 0

	totalServices++
	dbStatus := map[string]interface{}{
		"name":    "Database",
		"status":  "unknown",
		"details": "",
	}

	dbHost := cfg.Database.Host
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := strconv.Itoa(cfg.Database.Port)
	if cfg.Database.Port == 0 {
		dbPort = "5432"
	}
	dbName := cfg.Database.Name
	if dbName == "" {
		dbName = "lettersmith"
	}
	dbUser := cfg.Database.User
	if dbUser == "" {
		dbUser = "lettersmith"
	}

	if db != nil {
		if err := db.Ping(); err != nil {
			dbStatus["status"] = "error"
			dbStatus["details"] = fmt.Sprintf("Connection failed to %s:%s/%s (user: %s): %v", dbHost, dbPort, dbName, dbUser, err)
		} else {
			dbStatus["status"] = "healthy"
			dbStatus["details"] = fmt.Sprintf("Connected to %s:%s/%s (user: %s)", dbHost, dbPort, dbName, dbUser)
			healthyCount++
		}
	} else {
		dbStatus["status"] = "error"
		dbStatus["details"] = fmt.Sprintf("Database not initialized for %s:%s/%s", dbHost, dbPort, dbName)
	}
	services["database"] = dbStatus

	totalServices++
	emailStatus := map[string]interface{}{
		"name":    "Email Service",
		"status":  "unknown",
		"details": "",
	}

	emailProvider := envValues["EMAIL_PROVIDER"]
	if emailProvider == "" {
		emailStatus["status"] = "not_configured"
		emailStatus["details"] = "No email provider configured"
		missingComponents = append(missingComponents, "Email Provider")
	} else if emailProvider == "smtp" {
		if envValues["SMTP_HOST"] != "" && envValues["SMTP_PORT"] != "" &&
			envValues["SMTP_USERNAME"] != "" && envValues["SMTP_PASSWORD"] != "" {
			emailConfig := &config.EmailConfig{
				Provider: "smtp",
				SMTP: config.SMTPConfig{
					Host:     envValues["SMTP_HOST"],
					Username: envValues["SMTP_USERNAME"],
					Password: envValues["SMTP_PASSWORD"],
				},
			}
			if port, err := strconv.Atoi(envValues["SMTP_PORT"]); err == nil {
				emailConfig.SMTP.Port = port
			}

			emailClient := email.NewClient(emailConfig)
			if err := emailClient.TestConnection(); err != nil {
				emailStatus["status"] = "error"
				emailStatus["details"] = fmt.Sprintf("SMTP connection failed: %v", err)
			} else {
				emailStatus["status"] = "healthy"
				emailStatus["details"] = fmt.Sprintf("SMTP connection to %s:%s successful", envValues["SMTP_HOST"], envValues["SMTP_PORT"])
				healthyCount++
			}
		} else {
			emailStatus["status"] = "misconfigured"
			emailStatus["details"] = "SMTP provider selected but missing required configuration"
			missingComponents = append(missingComponents, "SMTP Configuration")
		}
	} else {
		emailStatus["status"] = "not_implemented"
		emailStatus["details"] = fmt.Sprintf("Email provider '%s' not yet implemented", emailProvider)
		missingComponents = append(missingComponents, fmt.Sprintf("%s Email Implementation", emailProvider))
	}
	services["email"] = emailStatus

	totalServices++
	aiStatus := map[string]interface{}{
		"name":    "Letter Generation Method",
		"status":  "unknown",
		"details": "",
	}

	generationMethod := envValues["LETTER_GENERATION_METHOD"]
	if generationMethod == "" {
		generationMethod = "ai"
	}

	if generationMethod == "ai" {
		aiProvider := envValues["AI_PROVIDER"]
		if aiProvider == "" {
			aiStatus["status"] = "not_configured"
			aiStatus["details"] = "AI generation selected but no provider configured"
			missingComponents = append(missingComponents, "AI Provider")
		} else if aiProvider == "openai" && envValues["OPENAI_API_KEY"] != "" {
			aiStatus["status"] = "not_implemented"
			aiStatus["details"] = "OpenAI API key configured but client not implemented"
			missingComponents = append(missingComponents, "OpenAI Client Implementation")
		} else if aiProvider == "anthropic" && envValues["ANTHROPIC_API_KEY"] != "" {
			aiStatus["status"] = "not_implemented"
			aiStatus["details"] = "Anthropic API key configured but client not implemented"
			missingComponents = append(missingComponents, "Anthropic Client Implementation")
		} else {
			aiStatus["status"] = "misconfigured"
			aiStatus["details"] = fmt.Sprintf("AI provider '%s' selected but API key missing", aiProvider)
			missingComponents = append(missingComponents, fmt.Sprintf("%s API Key", aiProvider))
		}
	} else if generationMethod == "templates" {
		templateDir := envValues["TEMPLATE_DIRECTORY"]
		if templateDir == "" {
			templateDir = "templates/"
		}
		aiStatus["status"] = "not_implemented"
		aiStatus["details"] = fmt.Sprintf("Template generation configured (dir: %s) but not implemented", templateDir)
		missingComponents = append(missingComponents, "Template Engine Implementation")
	}
	services["ai"] = aiStatus

	totalServices++
	repsStatus := map[string]interface{}{
		"name":    "Representative Lookup",
		"status":  "unknown",
		"details": "",
	}

	openstatesKey := envValues["OPENSTATES_API_KEY"]
	userZip := envValues["USER_ZIP_CODE"]

	if openstatesKey == "" {
		repsStatus["status"] = "not_configured"
		repsStatus["details"] = "No OpenStates API key configured"
		missingComponents = append(missingComponents, "OpenStates API Key")
	} else if userZip == "" {
		repsStatus["status"] = "incomplete"
		repsStatus["details"] = "OpenStates API key configured but USER_ZIP_CODE missing"
		missingComponents = append(missingComponents, "User ZIP Code")
	} else if geocoderInstance == nil {
		repsStatus["status"] = "error"
		repsStatus["details"] = "OpenStates configured but geocoding service unavailable"
	} else {
		var repCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM representatives").Scan(&repCount); err != nil {
			repsStatus["status"] = "error"
			repsStatus["details"] = fmt.Sprintf("Database error: %v", err)
		} else if repCount > 0 {
			repsStatus["status"] = "healthy"
			repsStatus["details"] = fmt.Sprintf("OpenStates integration working - %d representatives loaded", repCount)
			healthyCount++
		} else {
			repsStatus["status"] = "ready"
			repsStatus["details"] = "OpenStates configured and ready - click 'Sync from OpenStates' to load representatives"
			healthyCount++
		}
	}
	services["representatives"] = repsStatus

	totalServices++
	geoStatus := map[string]interface{}{
		"name":    "Geocoding Service",
		"status":  "unknown",
		"details": "",
	}

	if geocoderInstance != nil {
		geoStatus["status"] = "healthy"
		geoStatus["details"] = "ZIP code geocoding service initialized"
		healthyCount++
	} else {
		geoStatus["status"] = "error"
		geoStatus["details"] = "Geocoding service not initialized"
	}
	services["geocoding"] = geoStatus

	totalServices++
	schedulerStatus := map[string]interface{}{
		"name":    "Scheduler",
		"status":  "not_implemented",
		"details": "Automated scheduling not yet implemented",
	}
	missingComponents = append(missingComponents, "Scheduler Implementation")
	services["scheduler"] = schedulerStatus

	totalServices++
	userStatus := map[string]interface{}{
		"name":    "User Configuration",
		"status":  "unknown",
		"details": "",
	}

	if envValues["USER_NAME"] != "" && envValues["USER_EMAIL"] != "" && envValues["USER_ZIP_CODE"] != "" {
		userStatus["status"] = "healthy"
		userStatus["details"] = "User information configured"
		healthyCount++
	} else {
		userStatus["status"] = "incomplete"
		userStatus["details"] = "Missing required user information (name, email, or ZIP code)"
		missingComponents = append(missingComponents, "User Information")
	}
	services["user_config"] = userStatus

	status["missing_components"] = missingComponents
	status["summary"] = map[string]interface{}{
		"healthy_services":      healthyCount,
		"total_services":        totalServices,
		"completion_percentage": int((float64(healthyCount) / float64(totalServices)) * 100),
		"ready_for_operation":   len(missingComponents) == 0 && healthyCount == totalServices,
	}

	if len(missingComponents) > 0 {
		status["overall_status"] = "incomplete"
	} else if healthyCount < totalServices {
		status["overall_status"] = "degraded"
	} else {
		status["overall_status"] = "healthy"
	}

	json.NewEncoder(w).Encode(status)
}

func handleDatabaseDebug(w http.ResponseWriter, _ *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Database not connected",
		})
		return
	}

	debugInfo := map[string]interface{}{
		"connection_status": "connected",
		"database_version":  "",
		"current_database":  "",
		"tables":            []map[string]interface{}{},
		"total_tables":      0,
	}

	var version string
	if err := db.QueryRow("SELECT version()").Scan(&version); err == nil {
		debugInfo["database_version"] = version
	}

	var dbName string
	if err := db.QueryRow("SELECT current_database()").Scan(&dbName); err == nil {
		debugInfo["current_database"] = dbName
	}

	query := `
		SELECT 
			t.table_name,
			COALESCE(s.n_tup_ins, 0) as estimated_rows,
			t.table_type
		FROM information_schema.tables t
		LEFT JOIN pg_stat_user_tables s ON t.table_name = s.relname
		WHERE t.table_schema = 'public' 
		ORDER BY t.table_name
	`

	rows, err := db.Query(query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to query tables: %v", err),
		})
		return
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var tableName, tableType string
		var estimatedRows int64

		if err := rows.Scan(&tableName, &estimatedRows, &tableType); err != nil {
			continue
		}

		var actualRows int64
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if err := db.QueryRow(countQuery).Scan(&actualRows); err == nil {
			tables = append(tables, map[string]interface{}{
				"name":           tableName,
				"type":           tableType,
				"row_count":      actualRows,
				"estimated_rows": estimatedRows,
			})
		} else {
			tables = append(tables, map[string]interface{}{
				"name":           tableName,
				"type":           tableType,
				"row_count":      "error",
				"estimated_rows": estimatedRows,
				"error":          err.Error(),
			})
		}
	}

	debugInfo["tables"] = tables
	debugInfo["total_tables"] = len(tables)
	debugInfo["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	json.NewEncoder(w).Encode(debugInfo)
}

func updateDatabaseUserPassword(db *sql.DB, newUser string, newPassword string) error {
	query := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s'", newUser, newPassword)
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to update database user password: %w", err)
	}
	return nil
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	_, err = db.Exec(`
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ language 'plpgsql';
	`)
	if err != nil {
		return fmt.Errorf("failed to create update function: %w", err)
	}

	migrations := []string{
		"001_initial_schema.sql",
		"002_zip_coordinates.sql",
	}

	for _, migration := range migrations {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", migration).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", migration, err)
		}

		if count > 0 {
			log.Printf("Migration %s already applied, skipping", migration)
			continue
		}

		log.Printf("Applying migration: %s", migration)
		content, err := os.ReadFile(fmt.Sprintf("migrations/%s", migration))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migration, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration, err)
		}

		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration, err)
		}

		log.Printf("Successfully applied migration: %s", migration)
	}

	return nil
}

func handleTestRepresentatives(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	envValues := readEnvFile()
	userZip := envValues["USER_ZIP_CODE"]
	if userZip == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "USER_ZIP_CODE not configured in .env file",
		})
		return
	}

	openstatesKey := envValues["OPENSTATES_API_KEY"]
	if openstatesKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "OPENSTATES_API_KEY not configured in .env file",
			"note":  "Get a free API key from https://openstates.org/api/",
		})
		return
	}

	if geocoderInstance == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Geocoding service not available",
		})
		return
	}

	coords, err := geocoderInstance.GetCoordinates(userZip)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    fmt.Sprintf("Failed to get coordinates for ZIP %s: %v", userZip, err),
			"zip_code": userZip,
		})
		return
	}

	resp, err := geocoderInstance.GetRepresentativesFromZip(userZip, openstatesKey)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    fmt.Sprintf("Failed to get representatives: %v", err),
			"zip_code": userZip,
			"coordinates": map[string]interface{}{
				"latitude":  coords.Latitude,
				"longitude": coords.Longitude,
				"city":      coords.City,
				"state":     coords.State,
			},
		})
		return
	}
	defer resp.Body.Close()

	var representatives map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&representatives); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    fmt.Sprintf("Failed to parse OpenStates response: %v", err),
			"zip_code": userZip,
		})
		return
	}

	result := map[string]interface{}{
		"zip_code": userZip,
		"coordinates": map[string]interface{}{
			"latitude":  coords.Latitude,
			"longitude": coords.Longitude,
			"city":      coords.City,
			"state":     coords.State,
		},
		"representatives": representatives,
		"api_status":      resp.StatusCode,
	}

	json.NewEncoder(w).Encode(result)
}

func handleGetRepresentatives(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	envValues := readEnvFile()
	userZip := envValues["USER_ZIP_CODE"]
	if userZip == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "USER_ZIP_CODE not configured in .env file",
		})
		return
	}

	repsService := reps.NewService(db)

	representatives, err := repsService.GetUserRepresentatives(userZip)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to get representatives: %v", err),
		})
		return
	}

	result := map[string]interface{}{
		"zip_code":        userZip,
		"representatives": representatives,
		"count":           len(representatives),
	}

	json.NewEncoder(w).Encode(result)
}

func handleSyncRepresentatives(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	envValues := readEnvFile()
	userZip := envValues["USER_ZIP_CODE"]
	openstatesKey := envValues["OPENSTATES_API_KEY"]

	if userZip == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "USER_ZIP_CODE not configured in .env file",
		})
		return
	}

	if openstatesKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "OPENSTATES_API_KEY not configured in .env file",
		})
		return
	}

	if geocoderInstance == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Geocoding service not available",
		})
		return
	}

	coords, err := geocoderInstance.GetCoordinates(userZip)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to get coordinates for ZIP %s: %v", userZip, err),
		})
		return
	}

	repsService := reps.NewService(db)
	err = repsService.SyncFromOpenStates(coords.Latitude, coords.Longitude, openstatesKey, coords.State)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to sync representatives: %v", err),
		})
		return
	}

	representatives, err := repsService.GetUserRepresentatives(userZip)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to get updated representatives: %v", err),
		})
		return
	}

	result := map[string]interface{}{
		"status":          "Representatives synced successfully",
		"zip_code":        userZip,
		"representatives": representatives,
		"count":           len(representatives),
	}

	json.NewEncoder(w).Encode(result)
}

func handleRepresentativeByID(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	id, err := reps.ExtractIDFromPath(r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Invalid representative ID: %v", err),
		})
		return
	}

	repsService := reps.NewService(db)

	switch r.Method {
	case http.MethodPut:
		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid JSON format",
			})
			return
		}

		err := repsService.UpdateRepresentative(id, updates)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Failed to update representative: %v", err),
			})
			return
		}

		rep, err := repsService.GetRepresentativeByID(id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Failed to get updated representative: %v", err),
			})
			return
		}

		json.NewEncoder(w).Encode(rep)

	case http.MethodDelete:
		err := repsService.DeleteRepresentative(id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Failed to delete representative: %v", err),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "Representative deleted successfully",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGenerateLetter(w http.ResponseWriter, r *http.Request, cfg *config.Config, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	var requestData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON format",
		})
		return
	}

	advocacy, ok := requestData["advocacy"].(map[string]interface{})
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing advocacy data",
		})
		return
	}

	mainIssue, _ := advocacy["main_issue"].(string)
	specificConcern, _ := advocacy["specific_concern"].(string)
	requestedAction, _ := advocacy["requested_action"].(string)

	if mainIssue == "" || specificConcern == "" || requestedAction == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing required fields: main_issue, specific_concern, requested_action",
		})
		return
	}

	envValues := readEnvFile()
	userName := envValues["USER_NAME"]
	userZipCode := envValues["USER_ZIP_CODE"]

	if userName == "" || userZipCode == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "User name and ZIP code must be configured",
		})
		return
	}

	aiProvider := envValues["AI_PROVIDER"]
	if aiProvider == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "AI provider not configured",
		})
		return
	}

	var aiAPIKey, aiModel string
	if aiProvider == "openai" {
		aiAPIKey = envValues["OPENAI_API_KEY"]
		aiModel = envValues["OPENAI_MODEL"]
		if aiModel == "" {
			aiModel = "gpt-4"
		}
	} else if aiProvider == "anthropic" {
		aiAPIKey = envValues["ANTHROPIC_API_KEY"]
		aiModel = envValues["ANTHROPIC_MODEL"]
		if aiModel == "" {
			aiModel = "claude-3-sonnet-20240229"
		}
	}

	if aiAPIKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("%s API key not configured", aiProvider),
		})
		return
	}

	// Get all available representatives so AI can choose
	repsService := reps.NewService(db)
	representatives, err := repsService.GetUserRepresentatives(userZipCode)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to get representatives: %v", err),
		})
		return
	}

	if len(representatives) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "No representatives found. Please sync representatives first.",
		})
		return
	}

	// Convert representatives to the format expected by AI
	availableReps := make([]ai.RepresentativeOption, len(representatives))
	for i, rep := range representatives {
		availableReps[i] = ai.RepresentativeOption{
			ID:       rep.ID,
			Name:     rep.Name,
			Title:    rep.Title,
			State:    rep.State,
			Party:    rep.Party,
			District: rep.District,
		}
	}

	aiClient, err := ai.NewClient(aiProvider, aiAPIKey, aiModel)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to create AI client: %v", err),
		})
		return
	}

	letterTone := envValues["LETTER_TONE"]
	if letterTone == "" {
		letterTone = "professional"
	}

	maxLength := 500
	if maxLenStr := envValues["LETTER_MAX_LENGTH"]; maxLenStr != "" {
		if parsed, err := strconv.Atoi(maxLenStr); err == nil {
			maxLength = parsed
		}
	}

	generationRequest := &ai.GenerationRequest{
		MainIssue:                mainIssue,
		SpecificIssue:            specificConcern,
		RequestedAction:          requestedAction,
		UserName:                 userName,
		UserZipCode:              userZipCode,
		AvailableRepresentatives: availableReps,
		Tone:                     letterTone,
		MaxLength:                maxLength,
	}

	ctx := context.Background()
	letter, err := aiClient.GenerateLetter(ctx, generationRequest)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to generate letter: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "Letter generated successfully",
		"letter": map[string]interface{}{
			"subject":                 letter.Subject,
			"content":                 letter.Content,
			"metadata":                letter.Metadata,
			"created_at":              letter.CreatedAt,
			"selected_representative": letter.SelectedRepresentative,
		},
		"input": map[string]string{
			"main_issue":       mainIssue,
			"specific_concern": specificConcern,
			"requested_action": requestedAction,
		},
		"ai_selection": map[string]interface{}{
			"selected_representative_id": letter.Metadata.SelectedRepresentativeID,
			"reasoning":                  "AI automatically selected the most appropriate representative for this issue",
		},
		"configuration_used": map[string]interface{}{
			"max_length":  maxLength,
			"tone":        letterTone,
			"ai_provider": aiProvider,
			"ai_model":    aiModel,
		},
	})
}
