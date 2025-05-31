# Lettersmith Development Guide

This guide covers development setup, API details, and contribution guidelines for Lettersmith.

## Development Setup

### Prerequisites

- Docker and Docker Compose (recommended)
- Go 1.23+ (for local development without Docker)
- PostgreSQL (for local development without Docker)

### Local Development

**Option A: Docker Development (Recommended)**
```bash
# Clone and setup
git clone https://github.com/yourusername/lettersmith.git
cd lettersmith
./init-env.sh

# Start with hot reload
docker compose -f docker-compose.dev.yml up -d

# View logs
docker compose logs -f app
```

**Option B: Native Go Development**
```bash
# Install dependencies
go mod download

# Setup database (PostgreSQL required)
createdb lettersmith
export DATABASE_URL="postgres://localhost/lettersmith?sslmode=disable"

# Run migrations
go run cmd/migrate/main.go

# Run the server
go run cmd/server/main.go

# Run tests
go test -v ./...

# Format code
go fmt ./...
```

### Building

The application uses a multi-stage Docker build for minimal image size:

```dockerfile
# Build stage: ~400MB
FROM golang:1.23-alpine AS builder

# Final stage: ~16MB
FROM alpine:latest
```

**Manual build:**
```bash
# Build binary
go build -o lettersmith ./cmd/server

# Build Docker image
docker build -t lettersmith .
```

## API Documentation

### Configuration Endpoints

#### `GET /api/health`
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "service": "lettersmith"
}
```

#### `GET /api/config`
Get current configuration (sanitized, no secrets).

**Response:**
```json
{
  "server": { "host": "0.0.0.0", "port": 8080 },
  "ai": {
    "provider": "openai",
    "openai": { "model": "gpt-4", "configured": true },
    "anthropic": { "model": "claude-3-sonnet", "configured": false }
  },
  "email": {
    "provider": "smtp",
    "smtp": { "host": "smtp.example.com", "configured": true }
  },
  "user": { "Name": "John Doe", "Email": "john@example.com" },
  "env_values": { 
    "USER_NAME": "John Doe", 
    "USER_EMAIL": "john@example.com",
    "OPENAI_API_KEY": "••••••••",
    "SMTP_PASSWORD": "••••••••"
  }
}
```

**Note**: API keys and passwords are masked with bullet characters (`••••••••`) for security while still indicating they are configured.

#### `POST /api/config`
Update configuration (.env file).

**Request:**
```json
{
  "user": {
    "name": "John Doe",
    "email": "john@example.com",
    "zip_code": "12345"
  },
  "ai": {
    "provider": "openai",
    "openai": { "api_key": "sk-...", "model": "gpt-4" }
  },
  "email": {
    "provider": "smtp",
    "smtp": {
      "host": "smtp.gmail.com",
      "port": 587,
      "username": "user@gmail.com",
      "password": "password"
    }
  }
}
```

**Response:**
```json
{
  "status": "Configuration updated successfully"
}
```

#### `GET /api/config/debug`
Debug configuration status and environment variables.

**Response:**
```json
{
  "environment_variables": {
    "DATABASE_URL": "set",
    "OPENAI_API_KEY": "set (masked)",
    "USER_EMAIL": "john@example.com"
  },
  "configuration_status": {
    "user_configured": true,
    "ai_configured": true,
    "email_configured": true,
    "validation_result": "valid"
  }
}
```

#### `POST /api/config/test-email`
Test email configuration.

**Request:**
```json
{
  "user": { "email": "test@example.com" },
  "email": {
    "provider": "smtp",
    "smtp": {
      "host": "smtp.gmail.com",
      "port": 587,
      "username": "user@gmail.com",
      "password": "password"
    }
  }
}
```

### Planned Endpoints

- `POST /api/letters/generate` - Generate a test letter
- `POST /api/letters/send` - Manually trigger letter sending
- `GET /api/letters/history` - View sent letters
- `GET /api/representatives` - List found representatives

## Project Structure

```
lettersmith/
├── cmd/
│   ├── server/          # Main application server
│   │   └── main.go      # HTTP server, config handlers
│   └── migrate/         # Database migration tool
├── internal/
│   ├── config/          # Environment variable configuration
│   │   └── config.go    # Config structs and loading
│   ├── api/             # AI provider interfaces
│   │   ├── openai.go    # OpenAI API client
│   │   └── anthropic.go # Anthropic API client
│   ├── email/           # Email sending logic
│   │   ├── client.go    # Email client interface
│   │   ├── smtp.go      # SMTP implementation
│   │   ├── sendgrid.go  # SendGrid implementation
│   │   └── mailgun.go   # Mailgun implementation
│   ├── reps/            # Representative lookup
│   │   ├── openstates.go # OpenStates API
│   │   └── civicinfo.go # Google Civic Info API
│   ├── scheduler/       # Daily job runner
│   │   └── scheduler.go # Cron-like scheduler
│   └── web/             # Web UI handlers (if any backend logic)
├── web/                 # Frontend static files
│   ├── index.html       # Configuration UI
│   ├── style.css        # Modern, privacy-focused styling
│   └── app.js           # Frontend logic with .env management
├── migrations/          # SQL migration files
│   ├── 001_init.sql     # Initial schema
│   └── 002_letters.sql  # Letters table
├── docker-compose.yml   # Production Docker Compose
├── docker-compose.dev.yml # Development Docker Compose
├── Dockerfile           # Multi-stage build
├── env.example          # Example environment variables
├── .env                 # Your configuration (gitignored)
├── README.md            # User-facing documentation
└── DEVELOPMENT.md       # This file
```

## Configuration Details

### AI Providers

**OpenAI**
- Models: gpt-4, gpt-3.5-turbo, gpt-4-turbo
- Requires API key from https://platform.openai.com
- Costs: ~$0.01-0.10 per letter depending on model

**Anthropic**
- Models: claude-3-opus, claude-3-sonnet, claude-3-haiku
- Requires API key from https://console.anthropic.com
- Costs: ~$0.01-0.15 per letter depending on model

### Email Providers

**SMTP** (Recommended for privacy)
- Works with any SMTP server
- ProtonMail Bridge recommended for privacy
- Gmail requires app passwords
- Supports TLS/SSL

**SendGrid**
- Requires API key from https://sendgrid.com
- Good deliverability rates
- Free tier: 100 emails/day

**Mailgun**
- Requires API key and domain from https://mailgun.com
- Reliable for transactional email
- Free tier: 5,000 emails/month for 3 months

### Letter Customization

- **Tone Options**: professional, passionate, conversational, urgent
- **Max Length**: 100-2000 words (default: 500)
- **Themes**: Privacy rights, consumer protection, data transparency, corporate accountability
- **Template Variables**: {{name}}, {{zip_code}}, {{representative_name}}, {{state}}

## Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific package tests
go test -v ./internal/config

# Run integration tests (requires database)
go test -v -tags=integration ./...
```

### Test Email Configuration

The application includes a built-in email test feature:

```bash
# Via API
curl -X POST http://localhost:8080/api/config/test-email \
  -H "Content-Type: application/json" \
  -d '{"email":{"provider":"smtp","smtp":{"host":"smtp.gmail.com","port":587,"username":"test@gmail.com","password":"password"}}}'

# Via Web UI
# Navigate to configuration page and click "Test Email"
```

## Contributing

### Development Workflow

1. **Fork the repository**
2. **Create a feature branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **Make your changes**
   - Follow Go best practices
   - Add tests for new features
   - Update documentation
4. **Test your changes**
   ```bash
   go test -v ./...
   go fmt ./...
   go vet ./...
   ```
5. **Commit and push**
   ```bash
   git commit -m 'Add amazing feature'
   git push origin feature/amazing-feature
   ```
6. **Open a Pull Request**

### Code Standards

- **Go formatting**: Use `go fmt`
- **Linting**: Use `go vet` and `golangci-lint`
- **Testing**: Aim for >80% test coverage
- **Documentation**: Comment exported functions and types
- **Privacy**: Never log sensitive data (API keys, passwords, emails)

### Environment Variables

When adding new configuration options:

1. Add to `internal/config/config.go` struct
2. Add loading logic in `loadFromEnv()`
3. Add to `env.example` with documentation
4. Add to web UI form in `web/index.html`
5. Add JavaScript handling in `web/app.js`
6. Update README.md configuration section

### Database Migrations

Create new migration files in `migrations/` directory:

```sql
-- migrations/003_new_feature.sql
CREATE TABLE IF NOT EXISTS new_table (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Security Considerations

- **API Keys**: Never log or expose in responses - the `/api/config` endpoint masks all keys and passwords with bullet characters
- **Email Passwords**: Stored only in .env file, never exposed in API responses
- **Database**: Use parameterized queries
- **CORS**: Restrict origins in production
- **Rate Limiting**: Consider adding for API endpoints
- **Input Validation**: Sanitize all user inputs
- **Environment Variables**: Sensitive values are automatically detected and masked in API responses

## Performance Notes

- **Docker Image**: Multi-stage build keeps final image <20MB
- **Database**: Consider connection pooling for high volume
- **AI API Calls**: Implement retries and rate limiting
- **Email Sending**: Queue for bulk operations
- **Static Files**: Consider CDN for production

## Troubleshooting Development Issues

### Common Problems

**Go module issues:**
```bash
go mod tidy
go mod download
```

**Docker build fails:**
```bash
docker system prune
docker compose build --no-cache
```

**Database connection errors:**
```bash
# Check if PostgreSQL is running
docker compose ps
docker compose logs db
```

**Permission errors with .env:**
```bash
chmod 644 .env
ls -la .env
```

### Debug Mode

Enable verbose logging:
```bash
export LOG_LEVEL=debug
go run cmd/server/main.go
```

Or in Docker:
```yaml
# docker-compose.dev.yml
environment:
  - LOG_LEVEL=debug
```

## Release Process

Lettersmith uses semantic versioning (semver) for releases with an automated CI/CD pipeline.

### Version Tags

- **`latest`** - Latest stable release from main branch
- **`dev`** - Latest development build from dev branch  
- **`v1.2.3`** - Specific semantic version releases
- **`v1.2`** - Latest patch version for minor release
- **`v1`** - Latest minor version for major release

### Creating a Release

1. **Ensure you're on main branch with clean working directory**
2. **Use the release script**:
   ```bash
   ./scripts/release.sh
   ```
3. **Choose version bump type**:
   - **Patch** (v1.0.1) - Bug fixes, no new features
   - **Minor** (v1.1.0) - New features, backward compatible
   - **Major** (v2.0.0) - Breaking changes
   - **Custom** - For pre-releases like v1.0.0-beta.1

4. **The script will**:
   - Generate changelog from git commits
   - Create and push git tag
   - Trigger GitHub Actions build
   - Create GitHub release automatically

### Manual Release Process

If you prefer manual control:

```bash
# Ensure clean state
git checkout main
git pull origin main

# Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions will handle the rest
```

### Docker Image Publishing

GitHub Actions automatically publishes images to GitHub Container Registry:

- **Dev builds**: `ghcr.io/yourdatasucks/lettersmith:dev`
- **Main builds**: `ghcr.io/yourdatasucks/lettersmith:latest`  
- **Tagged releases**: `ghcr.io/yourdatasucks/lettersmith:v1.0.0`

### Using Different Versions

**In docker-compose.yml**:
```yaml
services:
  app:
    # Use latest stable
    image: ghcr.io/yourdatasucks/lettersmith:latest
    
    # Or use development version
    # image: ghcr.io/yourdatasucks/lettersmith:dev
    
    # Or pin to specific version
    # image: ghcr.io/yourdatasucks/lettersmith:v1.0.0
```

**Pull specific version**:
```bash
docker pull ghcr.io/yourdatasucks/lettersmith:v1.0.0
docker compose up -d
``` 