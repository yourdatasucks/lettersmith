# Lettersmith

A containerized Go application that uses AI to generate and send unique letters to US representatives advocating for privacy and consumer protection laws.

## Features

- 🤖 AI-powered letter generation using OpenAI or Anthropic
- 📧 Automated daily email sending to representatives
- 🔍 Privacy-respecting representative lookup via ProPublica, OpenStates, and USA.gov APIs
- 🐳 Fully containerized with Docker Compose
- 🌐 Web UI for easy configuration
- 📊 PostgreSQL database for tracking sent letters
- 🔧 Smart configuration management with persistence
- 🔒 Privacy-first: Minimal data collection (only name, email, and ZIP code)
- ✅ Visual indicators for configured API keys and credentials

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/yourusername/lettersmith.git
cd lettersmith
```

2. Start the application:
```bash
docker compose up -d
```

3. Open http://localhost:8080 in your browser to configure the application

4. Fill in your information and API keys, then click "Save Configuration"

That's it! The web UI will create your `.env` file automatically.

## Configuration

Lettersmith uses a **simple, user-friendly configuration system**:

- **Web UI** - Easy point-and-click configuration for non-technical users
- **`.env` file** - The source of truth that the web UI manages for you
- **JSON config** - Optional backup for advanced users

### 🎯 For Most Users: Use the Web UI

The web interface is designed to be **noob-friendly** and handles all the technical details:

1. Navigate to http://localhost:8080
2. Fill in the required fields:
   - **User Information**: Your name, email, and ZIP code
   - **AI Provider**: Choose OpenAI or Anthropic and provide API key
   - **Email Provider**: Configure SMTP, SendGrid, or Mailgun
   - **Representative APIs**: Add API keys for ProPublica and OpenStates
   - **Letter Settings**: Customize tone and length
3. Click "Save Configuration"

**What happens when you save:**
- ✅ Creates/updates your `.env` file automatically
- 🔐 Stores all settings securely in environment variables
- 💾 Configuration persists across container restarts
- 🚀 Ready to run immediately

### 🤓 For Advanced Users: Direct .env Editing

If you prefer manual configuration, you can edit the `.env` file directly:

```bash
# Copy the example file
cp env.example .env

# Edit with your values
nano .env
```

**Key environment variables:**
```bash
# User Information (required)
USER_NAME="Your Name"
USER_EMAIL=your-email@example.com
USER_ZIP_CODE=12345

# AI Provider (choose one)
AI_PROVIDER=openai
OPENAI_API_KEY=your-openai-api-key
# OR
AI_PROVIDER=anthropic
ANTHROPIC_API_KEY=your-anthropic-api-key

# Email Provider (choose one)
EMAIL_PROVIDER=smtp
SMTP_HOST=127.0.0.1
SMTP_PORT=1025
SMTP_USERNAME=your-email@example.com
SMTP_PASSWORD=your-password

# Optional: Representative APIs
PROPUBLICA_API_KEY=your-propublica-key
OPENSTATES_API_KEY=your-openstates-key
USAGOV_API_ENABLED=true

# Database (Docker handles this automatically)
DATABASE_URL=postgres://lettersmith:lettersmith_pass@db:5432/lettersmith?sslmode=disable
```

### 🔧 Configuration Priority

The application loads configuration in this order:
1. **Environment Variables** (`.env` file) - Primary source
2. **JSON Configuration** (optional fallback)
3. **Application Defaults**

The web UI reads from and writes to your `.env` file, making it both beginner-friendly and technically sound.

## Privacy-First Design

This application practices data minimization:
- **No phone numbers** collected
- **No street addresses** stored
- **No tracking or analytics**
- Only ZIP code used for representative lookup
- All data stays in your self-hosted database
- Recommends privacy-focused email providers (ProtonMail)

## Representative Lookup APIs

The project uses privacy-respecting APIs:

- **OpenStates API**: State legislature data (free tier available)

## Project Structure

```
lettersmith/
├── cmd/
│   ├── server/          # Main application server
│   └── migrate/         # Database migration tool
├── internal/
│   ├── config/          # Environment variable configuration
│   ├── api/             # AI provider interfaces
│   ├── email/           # Email sending logic
│   ├── reps/            # Representative lookup
│   ├── scheduler/       # Daily job runner
│   └── web/             # Configuration web UI handlers
├── web/                 # Frontend static files
│   ├── index.html       # Configuration UI
│   ├── style.css        # Modern, privacy-focused styling
│   └── app.js           # Frontend logic with .env management
├── migrations/          # SQL migration files
├── docker-compose.yml   # Docker Compose configuration
├── Dockerfile           # Multi-stage build (final image: 16.3MB)
├── env.example          # Example environment variables
└── .env                 # Your configuration (created by web UI)
```

## Docker Commands

```bash
# Start the application
docker compose up -d

# View logs
docker compose logs -f

# Stop the application (preserves data)
docker compose down

# Stop and remove all data
docker compose down -v

# Rebuild after code changes
docker compose up -d --build

# View specific service logs
docker compose logs -f app
docker compose logs -f db

# Access application shell
docker compose exec app sh

# Access database
docker compose exec db psql -U lettersmith
```

## Development

### Prerequisites

- Docker and Docker Compose
- Go 1.23+ (only for local development without Docker)
- PostgreSQL (only for local development without Docker)

### Local Development

For development without Docker:

```bash
# Install dependencies
go mod download

# Set up configuration
cp config.example.json config/config.json
# Edit config/config.json with your settings

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

## API Endpoints

- `GET /` - Web configuration UI
- `GET /api/health` - Health check endpoint
- `GET /api/config` - Get current configuration (sanitized, no secrets)
- `POST /api/config` - Update configuration (.env file)
- `GET /api/config/debug` - Debug configuration status and environment variables

### Planned Endpoints

- `POST /api/letters/generate` - Generate a test letter
- `POST /api/letters/send` - Manually trigger letter sending
- `GET /api/letters/history` - View sent letters
- `GET /api/representatives` - List found representatives

## Configuration Details

### AI Providers

**OpenAI**
- Models: gpt-4, gpt-3.5-turbo
- Requires API key from https://platform.openai.com

**Anthropic**
- Models: claude-3-opus, claude-3-sonnet, claude-3-haiku
- Requires API key from https://console.anthropic.com

### Email Providers

**SMTP** (Recommended for privacy)
- Works with any SMTP server
- ProtonMail Bridge recommended for privacy
- Supports standard SMTP authentication

**SendGrid**
- Requires API key from https://sendgrid.com
- Good deliverability rates

**Mailgun**
- Requires API key and domain
- Reliable for transactional email

### Letter Customization

- **Tone Options**: professional, passionate, conversational
- **Max Length**: Configurable (default: 500 words)
- **Themes**: Privacy rights, consumer protection, data transparency, corporate accountability

## Troubleshooting

### Configuration Issues

**Web UI not saving settings:**
- Check that the application has write permissions to the current directory
- Look for errors in the logs: `docker compose logs -f app`

**Environment variables not taking effect:**
- Restart the application: `docker compose restart`
- Check the debug endpoint: http://localhost:8080/api/config/debug

### Container Issues

```bash
# Check if services are running
docker compose ps

# View recent logs
docker compose logs --tail=50

# Restart services
docker compose restart

# Rebuild from scratch
docker compose down
docker compose up -d --build
```

### Database Connection

The application automatically connects to the PostgreSQL container. If you see connection errors:

1. Ensure the database container is healthy: `docker compose ps`
2. Check database logs: `docker compose logs db`
3. Verify DATABASE_URL in your .env file

## Security Notes

- All sensitive configuration is stored in your `.env` file
- API keys and passwords are never returned by the API
- The `.env` file should be kept secure and never committed to version control
- Use volume mounts for persistent storage
- Consider encrypting the `.env` file in production

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices
- Add tests for new features
- Update documentation
- Respect user privacy

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with privacy as a core principle
- Inspired by the need for citizen advocacy
- Thanks to all contributors and privacy advocates 