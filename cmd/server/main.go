package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yourdatasucks/lettersmith/internal/config"
	"github.com/yourdatasucks/lettersmith/internal/email"
	"github.com/yourdatasucks/lettersmith/internal/geocoding"

	_ "github.com/lib/pq"
)

var geocoderInstance *geocoding.ZipGeocoder

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
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

	// Initialize geocoding service for ZIP code to coordinates conversion
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

	// Store geocoder for use in handlers
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
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleConfigDebug(w, r, cfg)
	})

	// Add system health endpoint
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleSystemStatus(w, r, cfg, db)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "web/index.html")
			return
		}
		http.FileServer(http.Dir("web")).ServeHTTP(w, r)
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
		freshCfg.Letter.GenerationMethod = "ai" // default
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
		"letter":     currentCfg.Letter,            // Now uses fresh config from .env
		"env_values": sanitizeEnvValues(envValues), // Sanitize secrets before returning
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(safeConfig)
}

func handleUpdateConfig(w http.ResponseWriter, r *http.Request, _ *config.Config) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := updateEnvFile(updates); err != nil {
		log.Printf("Error updating .env file: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to update .env file",
		})
		return
	}

	log.Printf("Configuration updated in .env file")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "Configuration updated successfully",
	})
}

func handleConfigDebug(w http.ResponseWriter, _ *http.Request, cfg *config.Config) {

	envValues := readEnvFile()

	freshCfg := &config.Config{}

	if method := envValues["LETTER_GENERATION_METHOD"]; method != "" {
		freshCfg.Letter.GenerationMethod = method
	} else {
		freshCfg.Letter.GenerationMethod = "ai"
	}

	if provider := envValues["AI_PROVIDER"]; provider != "" {
		freshCfg.AI.Provider = provider
		if provider == "openai" {
			freshCfg.AI.OpenAI.APIKey = envValues["OPENAI_API_KEY"]
		} else if provider == "anthropic" {
			freshCfg.AI.Anthropic.APIKey = envValues["ANTHROPIC_API_KEY"]
		}
	}

	if freshCfg.Letter.GenerationMethod == "templates" {
		freshCfg.Letter.TemplateConfig = &config.TemplateConfig{
			Directory: envValues["TEMPLATE_DIRECTORY"],
		}
		if freshCfg.Letter.TemplateConfig.Directory == "" {
			freshCfg.Letter.TemplateConfig.Directory = "templates/"
		}
	}

	freshCfg.User = cfg.User
	freshCfg.Email = cfg.Email
	freshCfg.Representatives = cfg.Representatives

	debugInfo := map[string]interface{}{
		"environment_variables": map[string]string{
			"DATABASE_URL":               getEnvFileStatus(envValues, "DATABASE_URL"),
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
		"database_url_parsed": cfg.DatabaseURL(),
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

func updateEnvFile(updates map[string]interface{}) error {
	cwd, _ := os.Getwd()
	log.Printf("Current working directory: %s", cwd)

	if info, err := os.Stat(".env"); err == nil {
		log.Printf(".env file exists with permissions: %v", info.Mode())
	} else {
		log.Printf(".env file does not exist yet: %v", err)
	}

	// Test write permissions
	testFile := ".env_test_write"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		log.Printf("ERROR: Cannot write to current directory: %v", err)
		return fmt.Errorf("cannot write to current directory: %w", err)
	} else {
		os.Remove(testFile) // Clean up test file
		log.Printf("Current directory is writable")
	}

	existingEnv := readEnvFile()
	log.Printf("Read %d existing environment variables from .env", len(existingEnv))

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

	writeEnvSection(&envContent, "Database", map[string]string{
		"DATABASE_URL": existingEnv["DATABASE_URL"],
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
		} else if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "password") {
			sanitized[key] = "••••••••" // Use bullet characters to indicate it's set but hidden
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

func handleSystemStatus(w http.ResponseWriter, _ *http.Request, _ *config.Config, db *sql.DB) {
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

	// Check Database
	totalServices++
	dbStatus := map[string]interface{}{
		"name":    "Database",
		"status":  "unknown",
		"details": "",
	}
	if db != nil {
		if err := db.Ping(); err != nil {
			dbStatus["status"] = "error"
			dbStatus["details"] = fmt.Sprintf("Database connection failed: %v", err)
		} else {
			dbStatus["status"] = "healthy"
			dbStatus["details"] = "PostgreSQL connection successful"
			healthyCount++
		}
	} else {
		dbStatus["status"] = "error"
		dbStatus["details"] = "Database not initialized"
	}
	services["database"] = dbStatus

	// Check Email Configuration
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
			// Test SMTP connection
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

	// Check AI/Template Configuration
	totalServices++
	aiStatus := map[string]interface{}{
		"name":    "Letter Generation Method",
		"status":  "unknown",
		"details": "",
	}

	generationMethod := envValues["LETTER_GENERATION_METHOD"]
	if generationMethod == "" {
		generationMethod = "ai" // default
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

	// Check Representative Lookup
	totalServices++
	repsStatus := map[string]interface{}{
		"name":    "Representative Lookup",
		"status":  "unknown",
		"details": "",
	}

	if envValues["OPENSTATES_API_KEY"] != "" {
		repsStatus["status"] = "not_implemented"
		repsStatus["details"] = "OpenStates API key configured but client not implemented"
		missingComponents = append(missingComponents, "OpenStates Client Implementation")
	} else {
		repsStatus["status"] = "not_configured"
		repsStatus["details"] = "No representative lookup API configured"
		missingComponents = append(missingComponents, "Representative API")
	}
	services["representatives"] = repsStatus

	// Check Geocoding Service
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

	// Check Scheduler
	totalServices++
	schedulerStatus := map[string]interface{}{
		"name":    "Scheduler",
		"status":  "not_implemented",
		"details": "Automated scheduling not yet implemented",
	}
	missingComponents = append(missingComponents, "Scheduler Implementation")
	services["scheduler"] = schedulerStatus

	// Check User Configuration
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

	// Update summary
	status["missing_components"] = missingComponents
	status["summary"] = map[string]interface{}{
		"healthy_services":      healthyCount,
		"total_services":        totalServices,
		"completion_percentage": int((float64(healthyCount) / float64(totalServices)) * 100),
		"ready_for_operation":   len(missingComponents) == 0 && healthyCount == totalServices,
	}

	// Set overall status
	if len(missingComponents) > 0 {
		status["overall_status"] = "incomplete"
	} else if healthyCount < totalServices {
		status["overall_status"] = "degraded"
	} else {
		status["overall_status"] = "healthy"
	}

	json.NewEncoder(w).Encode(status)
}
