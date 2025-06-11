# Lettersmith Development Guide

This guide covers development setup, API details, and contribution guidelines for Lettersmith.

## Development Setup

### Prerequisites

- Docker and Docker Compose (recommended)
- Go 1.23+ (for local development without Docker)
- PostgreSQL (for local development without Docker)

### Local Development

For local development, see the [Development Workflow](#development-workflow) section below which covers contributor setup options in detail.

**Quick Start (Docker):**
```bash
# Clone the repository
git clone https://github.com/yourdatasucks/lettersmith.git
cd lettersmith
./init-env.sh

# Start with Docker (uses dev image by default)
docker compose up -d

# View logs
docker compose logs -f app
```

**Quick Start (Native Go):**
```bash
# Install dependencies
go mod download

# Setup database (PostgreSQL required)
createdb lettersmith
export DATABASE_URL="postgres://localhost/lettersmith?sslmode=disable"

# Run server (migrations run automatically on startup)
go run cmd/server/main.go
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

#### `GET /api/system/status`
Comprehensive system health check.

**Response:**
```json
{
  "status": "operational",
  "checks": {
    "database": { "status": "healthy", "message": "Connected to PostgreSQL" },
    "email": { "status": "healthy", "message": "SMTP configuration valid" },
    "ai": { "status": "warning", "message": "AI providers configured but not functional" },
    "geocoding": { "status": "healthy", "message": "ZIP geocoding operational" },
    "representatives": { "status": "healthy", "message": "OpenStates API operational" }
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### `GET /api/db/debug`
Database debug information and connection status.

### Planned Endpoints (Not Yet Implemented)

- `POST /api/letters/generate` - Generate a test letter using AI
- `POST /api/letters/send` - Manually send letters to representatives
- `GET /api/letters/history` - View sent letters and audit trail
- `POST /api/scheduler/trigger` - Manually trigger scheduled letter sending
- `GET /api/scheduler/status` - Check scheduled job status

### Representatives Endpoints (✅ Implemented)

#### `GET /api/representatives`
Get representatives for the user's ZIP code from local database.

**Response:**
```json
{
  "zip_code": "29414",
  "representatives": [
    {
      "id": 1,
      "name": "Tim Scott",
      "title": "Senator",
      "state": "SC",
      "district": "South Carolina",
      "party": "Republican",
      "email": null,
      "phone": null,
      "office_address": null,
      "website": null,
      "external_id": "ocd-person/...",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "count": 5
}
```

#### `POST /api/representatives`
Sync representatives from OpenStates API for the user's ZIP code.

**Response:**
```json
{
  "status": "Representatives synced successfully",
  "zip_code": "29414",
  "representatives": [...],
  "count": 5
}
```

#### `PUT /api/representatives/{id}`
Update representative information.

**Request:**
```json
{
  "name": "Updated Name",
  "email": "updated@example.com"
}
```

**Response:**
```json
{
  "status": "Representative updated successfully"
}
```

#### `DELETE /api/representatives/{id}`
Delete a representative from the database.

**Response:**
```json
{
  "status": "Representative deleted successfully"
}
```

#### `GET /api/test/representatives`
Test OpenStates API directly (returns raw API response for debugging).

**Response:**
```json
{
  "results": [...],
  "pagination": {...}
}
```

## Project Structure

```
lettersmith/
├── cmd/
│   ├── server/          # Main application server
│   │   └── main.go      # HTTP server, config handlers, representatives APIs
│   └── migrate/         # Database migration tool ✅ IMPLEMENTED
│       └── main.go      # SQL migration runner for PostgreSQL
├── internal/
│   ├── config/          # Environment variable configuration
│   │   └── config.go    # Config structs and loading
│   ├── api/             # AI provider interfaces
│   │   ├── client.go    # Common AI interface
│   │   ├── openai.go    # OpenAI API client (placeholder)
│   │   ├── anthropic.go # Anthropic API client (placeholder)
│   │   └── utils.go     # Utility functions
│   ├── email/           # Email sending logic
│   │   └── client.go    # SMTP email client
│   ├── reps/            # Representative lookup ✅ IMPLEMENTED
│   │   ├── types.go     # Representative structs and OpenStates API types
│   │   └── service.go   # CRUD operations and OpenStates integration
│   ├── geocoding/       # ZIP code to coordinates conversion ✅ IMPLEMENTED
│   │   ├── geocoding.go # Main geocoding service
│   │   ├── datasources.go # US Census Bureau data loading
│   │   └── openstates.go # OpenStates API integration
│   └── scheduler/       # Daily job runner (planned)
│       └── scheduler.go # Cron-like scheduler
├── web/                 # Frontend static files
│   ├── index.html       # Configuration UI
│   ├── status.html      # System status dashboard  
│   ├── representatives.html # Representatives management interface ✅
│   ├── style.css        # Modern, privacy-focused styling
│   └── app.js           # Frontend logic with .env management
├── migrations/          # SQL migration files
│   ├── 001_initial_schema.sql # Initial schema with representatives table
│   └── 002_zip_coordinates.sql # ZIP coordinates table
├── docker-compose.yml   # Docker Compose for development and production
├── Dockerfile           # Multi-stage build
├── env.example          # Example environment variables
├── .env                 # Your configuration (gitignored)
├── README.md            # User-facing documentation
├── DEVELOPMENT.md       # This file
├── IMPLEMENTATION_PLAN.md # Implementation roadmap
└── EMAIL_SETUP_GUIDE.md # Email provider setup guide
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

### Letter Customization (Planned)

**Note**: Letter generation is not yet implemented. These are planned features:

- **Tone Options**: professional, passionate, conversational, urgent
- **Max Length**: 100-2000 words (default: 500)
- **Themes**: Privacy rights, consumer protection, data transparency, corporate accountability
- **Template Variables**: {{name}}, {{zip_code}}, {{representative_name}}, {{state}}

For current implementation status, see [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md).

## Testing (Optional)

Unit tests are **optional** for contributors. The most important thing is ensuring the application works as a whole using the web interface and API endpoints.

### Running Tests (Optional)

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific package tests
go test -v ./internal/config

# Run integration tests (requires database)
go test -v -tags=integration ./...

# Format and vet code
go fmt ./...
go vet ./...
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

### Integration Testing (Recommended)

Instead of unit tests, focus on **end-to-end validation**:

1. **Start the application**: `docker compose up -d`
2. **Test via web UI**: Navigate to http://localhost:8080 and verify your changes work
3. **Test API endpoints**: Use the built-in test features or curl commands
4. **Check system status**: Visit `/status.html` to ensure all components are healthy

This approach ensures your changes work in the real environment users will experience.

## Contributing Guidelines

For the complete development workflow, see the [Development Workflow](#development-workflow) section below.

**Quick contributing steps:**
1. Fork the repository and clone your fork
2. Follow one of the development setup options in [Development Workflow](#development-workflow)
3. Create a feature branch from `dev`: `git checkout -b feature/amazing-feature dev`
4. Make your changes following the code standards below
5. Verify your changes work end-to-end (web UI, API functionality) and submit a PR to the `dev` branch

### Code Standards

- **Go formatting**: Use `go fmt`
- **Linting**: Use `go vet` and `golangci-lint`
- **Testing**: Unit tests are optional - focus on ensuring the application works end-to-end
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
# docker-compose.yml
environment:
  - LOG_LEVEL=debug
```

## Development Workflow

### For Contributors

Contributors have several options for development. The `DOCKER_IMAGE` environment variable in `.env` controls which Docker image is used (defaults to a test image, but can be overridden):

**Option A: Use Pre-built Dev Image (Easiest)**
```bash
# Fork the repo on GitHub, then clone your fork
git clone https://github.com/YOUR-USERNAME/lettersmith.git
cd lettersmith
git checkout dev
./init-env.sh

# Uses dev image by default (DOCKER_IMAGE=ghcr.io/yourdatasucks/lettersmith:dev)
docker compose up -d
```

**Option B: Local Docker Build**
```bash
# Clone your fork  
git clone https://github.com/YOUR-USERNAME/lettersmith.git
cd lettersmith
./init-env.sh

# Build locally instead of pulling from registry
docker build -t lettersmith:local .

# Override the .env file setting (temporary for this session)
export DOCKER_IMAGE=lettersmith:local
docker compose up -d

# Note: .env file has DOCKER_IMAGE=lettersmith-test:latest by default
# Your export temporarily overrides this setting
```

**Option C: Native Go Development**
```bash
# Clone your fork
git clone https://github.com/YOUR-USERNAME/lettersmith.git
cd lettersmith

# Run natively (requires Go + PostgreSQL)
go mod download
createdb lettersmith
export DATABASE_URL="postgres://localhost/lettersmith?sslmode=disable"
go run cmd/server/main.go
# Note: Database migrations run automatically on server startup
```

**Which option should I choose?**
- **Option A**: Best for most contributors - uses stable dev environment, no local building
- **Option B**: Good when you need to test Docker-specific changes or custom builds  
- **Option C**: Fastest for active development and debugging Go code

**Contributing workflow:**
1. Fork the repository on GitHub
2. Clone your fork locally  
3. Create feature branch from `dev`: `git checkout -b feature/my-feature dev`
4. Make changes and test using one of the options above
5. Push to your fork: `git push origin feature/my-feature`
6. Submit PR from your fork's feature branch to main repo's `dev` branch
7. After merge, main repo CI builds and publishes new `ghcr.io/yourdatasucks/lettersmith:dev`

### Docker Image Tags

The main repository CI/CD system automatically publishes:
- **`ghcr.io/yourdatasucks/lettersmith:dev`** - Latest dev branch (updated when dev branch changes)
- **`ghcr.io/yourdatasucks/lettersmith:latest`** - Latest stable release from main branch
- **`ghcr.io/yourdatasucks/lettersmith:v1.0.0`** - Specific version releases

**Note**: Contributor forks don't automatically publish to GHCR. Contributors should use Option A (pre-built dev image), Option B (local build), or Option C (native development) above.

### Using Different Versions

Use the `DOCKER_IMAGE` environment variable to switch between versions:

**For development (default):**
```bash
# Uses dev branch image (default)
docker compose up -d
# Same as: DOCKER_IMAGE=ghcr.io/yourdatasucks/lettersmith:dev docker compose up -d
```

**For production/stable use:**
```bash
# Temporary (this session only):
DOCKER_IMAGE=ghcr.io/yourdatasucks/lettersmith:latest docker compose up -d

# Or pin to specific version for reproducibility:
DOCKER_IMAGE=ghcr.io/yourdatasucks/lettersmith:v1.0.0 docker compose up -d
```

**For local development:**
```bash
# Build and use local image
docker build -t lettersmith:local .

# Temporary (this session only):
DOCKER_IMAGE=lettersmith:local docker compose up -d

# Or export for multiple commands in same session:
export DOCKER_IMAGE=lettersmith:local
docker compose up -d
docker compose logs -f app
```

**Persistent version selection:**
```bash
# Edit the existing DOCKER_IMAGE line in .env file
# Change: DOCKER_IMAGE=lettersmith-test:latest
# To:     DOCKER_IMAGE=ghcr.io/yourdatasucks/lettersmith:latest

# Or use sed to replace it:
sed -i 's|DOCKER_IMAGE=.*|DOCKER_IMAGE=ghcr.io/yourdatasucks/lettersmith:latest|' .env
docker compose up -d
```

## Release Process (Maintainer Only)

### Creating Releases

**Only project maintainers create releases using:**

```bash
# Ensure you're on main branch with clean working directory
./scripts/release.sh
```

**The release script will:**
1. Check you're on main branch with clean working directory
2. Show current version and changelog
3. Let you choose version bump (patch/minor/major/custom)
4. Create and push git tag
5. GitHub Actions automatically builds and publishes release images
6. GitHub release is created automatically with changelog

### Version Management

The project uses semantic versioning (semver):
- **Patch** (v1.0.1) - Bug fixes, no breaking changes
- **Minor** (v1.1.0) - New features, backward compatible  
- **Major** (v2.0.0) - Breaking changes

### Manual Release (Alternative)

```bash
# Ensure clean state on main
git checkout main
git pull origin main

# Create and push tag  
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions handles the rest automatically
``` 