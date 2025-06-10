package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Database        DatabaseConfig
	Server          ServerConfig
	AI              AIConfig
	Email           EmailConfig
	Representatives RepresentativesConfig
	User            UserConfig
	Scheduler       SchedulerConfig
	Letter          LetterConfig
	ZipDataUpdate   bool
	CensusBureauURL string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type ServerConfig struct {
	Port int
	Host string
}

type AIConfig struct {
	Provider  string // openai, anthropic
	OpenAI    OpenAIConfig
	Anthropic AnthropicConfig
}

type OpenAIConfig struct {
	APIKey string
	Model  string
}

type AnthropicConfig struct {
	APIKey string
	Model  string
}

type EmailConfig struct {
	Provider string // smtp, sendgrid, mailgun
	SMTP     SMTPConfig
	SendGrid SendGridConfig
	Mailgun  MailgunConfig
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type SendGridConfig struct {
	APIKey string
	From   string
}

type MailgunConfig struct {
	APIKey string
	Domain string
	From   string
}

type RepresentativesConfig struct {
	OpenStatesAPIKey string
}

type UserConfig struct {
	Name           string
	Email          string
	ZipCode        string
	SendCopyToSelf bool
}

type SchedulerConfig struct {
	SendTime string // 24-hour format, e.g., "09:00"
	Timezone string
	Enabled  bool
}

type LetterConfig struct {
	Themes           []string
	Tone             string // professional, passionate, conversational
	MaxLength        int
	GenerationMethod string // ai, templates
	TemplateConfig   *TemplateConfig
}

type TemplateConfig struct {
	Directory        string
	RotationStrategy string // sequential, random-unique, random
	Personalize      bool
}

func Load() (*Config, error) {
	cfg := &Config{}

	loadFromEnv(cfg)

	setDefaults(cfg)

	return cfg, nil
}

func loadFromEnv(cfg *Config) {
	// Load individual database configuration variables
	if user := os.Getenv("POSTGRES_USER"); user != "" {
		cfg.Database.User = user
	}
	if password := os.Getenv("POSTGRES_PASSWORD"); password != "" {
		cfg.Database.Password = password
	}
	if db := os.Getenv("POSTGRES_DB"); db != "" {
		cfg.Database.Name = db
	}
	if port := os.Getenv("POSTGRES_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Database.Port = p
		}
	}

	// Load DATABASE_URL if available, otherwise construct from individual fields
	if url := os.Getenv("DATABASE_URL"); url != "" {
		if parsed, err := parsePostgreSQLURL(url); err == nil {
			cfg.Database = *parsed
		}
	} else if cfg.Database.User != "" && cfg.Database.Password != "" && cfg.Database.Name != "" {
		// Construct DATABASE_URL from individual fields if not provided
		cfg.Database.Host = "localhost" // Default for Docker Compose
		if cfg.Database.Port == 0 {
			cfg.Database.Port = 5432 // Default PostgreSQL port
		}
		cfg.Database.SSLMode = "disable" // Default for local development
	}

	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}
	if host := os.Getenv("SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}

	if provider := os.Getenv("AI_PROVIDER"); provider != "" {
		cfg.AI.Provider = provider
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		cfg.AI.OpenAI.APIKey = apiKey
	}
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		cfg.AI.OpenAI.Model = model
	}
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.AI.Anthropic.APIKey = apiKey
	}
	if model := os.Getenv("ANTHROPIC_MODEL"); model != "" {
		cfg.AI.Anthropic.Model = model
	}

	if provider := os.Getenv("EMAIL_PROVIDER"); provider != "" {
		cfg.Email.Provider = provider
	}
	if host := os.Getenv("SMTP_HOST"); host != "" {
		cfg.Email.SMTP.Host = host
	}
	if port := os.Getenv("SMTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Email.SMTP.Port = p
		}
	}
	if username := os.Getenv("SMTP_USERNAME"); username != "" {
		cfg.Email.SMTP.Username = username
		cfg.Email.SMTP.From = username // Default from to username
	}
	if password := os.Getenv("SMTP_PASSWORD"); password != "" {
		cfg.Email.SMTP.Password = password
	}
	if from := os.Getenv("SMTP_FROM"); from != "" {
		cfg.Email.SMTP.From = from
	}

	if apiKey := os.Getenv("SENDGRID_API_KEY"); apiKey != "" {
		cfg.Email.SendGrid.APIKey = apiKey
		if from := os.Getenv("SENDGRID_FROM"); from != "" {
			cfg.Email.SendGrid.From = from
		}
	}

	if apiKey := os.Getenv("MAILGUN_API_KEY"); apiKey != "" {
		cfg.Email.Mailgun.APIKey = apiKey
	}
	if domain := os.Getenv("MAILGUN_DOMAIN"); domain != "" {
		cfg.Email.Mailgun.Domain = domain
		cfg.Email.Mailgun.From = fmt.Sprintf("lettersmith@%s", domain)
	}
	if from := os.Getenv("MAILGUN_FROM"); from != "" {
		cfg.Email.Mailgun.From = from
	}

	if apiKey := os.Getenv("OPENSTATES_API_KEY"); apiKey != "" {
		cfg.Representatives.OpenStatesAPIKey = apiKey
	}

	if name := os.Getenv("USER_NAME"); name != "" {
		cfg.User.Name = name
	}
	if email := os.Getenv("USER_EMAIL"); email != "" {
		cfg.User.Email = email
	}
	if zip := os.Getenv("USER_ZIP_CODE"); zip != "" {
		cfg.User.ZipCode = zip
	}
	if sendCopy := os.Getenv("SEND_COPY_TO_SELF"); sendCopy != "" {
		cfg.User.SendCopyToSelf = sendCopy == "true"
	}

	if sendTime := os.Getenv("SCHEDULER_SEND_TIME"); sendTime != "" {
		cfg.Scheduler.SendTime = sendTime
	}
	if tz := os.Getenv("SCHEDULER_TIMEZONE"); tz != "" {
		cfg.Scheduler.Timezone = tz
	}
	if enabled := os.Getenv("SCHEDULER_ENABLED"); enabled != "" {
		cfg.Scheduler.Enabled = enabled == "true"
	}

	if tone := os.Getenv("LETTER_TONE"); tone != "" {
		cfg.Letter.Tone = tone
	}
	if maxLength := os.Getenv("LETTER_MAX_LENGTH"); maxLength != "" {
		if length, err := strconv.Atoi(maxLength); err == nil {
			cfg.Letter.MaxLength = length
		}
	}
	if method := os.Getenv("LETTER_GENERATION_METHOD"); method != "" {
		cfg.Letter.GenerationMethod = method
	}

	if themes := os.Getenv("LETTER_THEMES"); themes != "" {
		cfg.Letter.Themes = strings.Split(themes, ",")

		for i, theme := range cfg.Letter.Themes {
			cfg.Letter.Themes[i] = strings.TrimSpace(theme)
		}
	}

	if cfg.Letter.GenerationMethod == "templates" {
		if cfg.Letter.TemplateConfig == nil {
			cfg.Letter.TemplateConfig = &TemplateConfig{}
		}
		if dir := os.Getenv("TEMPLATE_DIRECTORY"); dir != "" {
			cfg.Letter.TemplateConfig.Directory = dir
		}
		if strategy := os.Getenv("TEMPLATE_ROTATION_STRATEGY"); strategy != "" {
			cfg.Letter.TemplateConfig.RotationStrategy = strategy
		}
		if personalize := os.Getenv("TEMPLATE_PERSONALIZE"); personalize != "" {
			cfg.Letter.TemplateConfig.Personalize = personalize == "true"
		}
	}

	// ZIP code data update setting
	if zipUpdate := os.Getenv("ZIP_DATA_UPDATE"); zipUpdate != "" {
		cfg.ZipDataUpdate = zipUpdate == "true"
	}

	if censusBureauURL := os.Getenv("CENSUS_BUREAU_URL"); censusBureauURL != "" {
		cfg.CensusBureauURL = censusBureauURL
	}
}

func parsePostgreSQLURL(url string) (*DatabaseConfig, error) {

	if !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "postgresql://") {
		return nil, fmt.Errorf("invalid postgresql url format")
	}

	url = strings.TrimPrefix(url, "postgres://")
	url = strings.TrimPrefix(url, "postgresql://")

	var config DatabaseConfig

	atIndex := strings.LastIndex(url, "@")
	if atIndex == -1 {
		return nil, fmt.Errorf("invalid postgresql url: missing @ separator")
	}

	credentials := url[:atIndex]
	hostPart := url[atIndex+1:]

	if colonIndex := strings.Index(credentials, ":"); colonIndex != -1 {
		config.User = credentials[:colonIndex]
		config.Password = credentials[colonIndex+1:]
	} else {
		config.User = credentials
	}

	slashIndex := strings.Index(hostPart, "/")
	if slashIndex == -1 {
		return nil, fmt.Errorf("invalid postgresql url: missing database name")
	}

	hostPort := hostPart[:slashIndex]
	dbPart := hostPart[slashIndex+1:]

	if colonIndex := strings.LastIndex(hostPort, ":"); colonIndex != -1 {
		config.Host = hostPort[:colonIndex]
		if port, err := strconv.Atoi(hostPort[colonIndex+1:]); err == nil {
			config.Port = port
		}
	} else {
		config.Host = hostPort
		config.Port = 5432 // default
	}

	questionIndex := strings.Index(dbPart, "?")
	if questionIndex != -1 {
		config.Name = dbPart[:questionIndex]
		params := dbPart[questionIndex+1:]

		for _, param := range strings.Split(params, "&") {
			if kv := strings.SplitN(param, "=", 2); len(kv) == 2 {
				if kv[0] == "sslmode" {
					config.SSLMode = kv[1]
				}
			}
		}
	} else {
		config.Name = dbPart
	}

	return &config, nil
}

func setDefaults(cfg *Config) {

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}

	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
	}
	if cfg.Database.User == "" {
		cfg.Database.User = "lettersmith"
	}
	if cfg.Database.Name == "" {
		cfg.Database.Name = "lettersmith"
	}

	if cfg.Scheduler.Timezone == "" {
		cfg.Scheduler.Timezone = "America/Los_Angeles"
	}
	if cfg.Scheduler.SendTime == "" {
		cfg.Scheduler.SendTime = "09:00"
	}

	if cfg.Letter.MaxLength == 0 {
		cfg.Letter.MaxLength = 500
	}
	if cfg.Letter.Tone == "" {
		cfg.Letter.Tone = "professional"
	}
	if cfg.Letter.GenerationMethod == "" {
		cfg.Letter.GenerationMethod = "ai"
	}

	if len(cfg.Letter.Themes) == 0 {
		cfg.Letter.Themes = []string{
			"data privacy protection",
			"consumer rights",
			"corporate accountability",
			"transparent data practices",
		}
	}

	if cfg.Letter.GenerationMethod == "templates" && cfg.Letter.TemplateConfig == nil {
		cfg.Letter.TemplateConfig = &TemplateConfig{
			Directory:        "templates/",
			RotationStrategy: "random-unique",
			Personalize:      true,
		}
	}

	if cfg.AI.OpenAI.Model == "" {
		cfg.AI.OpenAI.Model = "gpt-4"
	}
	if cfg.AI.Anthropic.Model == "" {
		cfg.AI.Anthropic.Model = "claude-3-sonnet-20240229"
	}

	// Set default ZIP data update to true for fresh installs
	if cfg.ZipDataUpdate == false {
		cfg.ZipDataUpdate = true
	}
}

// DatabaseURL returns the database connection string
func (cfg *Config) DatabaseURL() string {
	if cfg.Database.Host == "" {
		// If no database config is provided, use default for development
		return "postgres://lettersmith:lettersmith_pass@localhost:5432/lettersmith?sslmode=disable"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
}
