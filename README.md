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
git clone https://github.com/yourdatasucks/lettersmith.git
cd lettersmith
```

2. Create a `.env` file (required for Docker):
```bash
# Option A: Create a blank .env file
touch .env

# Option B: Copy from example (recommended)
cp env.example .env
```

3. Start the application:
```bash
docker compose up -d
```

**Docker Image Versions:**
- `latest` - Latest stable release (recommended)
- `dev` - Development version with latest features
- `v1.0.0` - Specific version tags

4. Open http://localhost:8080 in your browser to configure the application

5. Fill in your information and API keys, then click "Save Configuration"

**That's it!** The web UI will update your `.env` file automatically and persist your configuration across container restarts.

## Configuration

Lettersmith uses a **simple, user-friendly configuration system**:

- **Web UI** - Easy point-and-click configuration for non-technical users
- **`.env` file** - The source of truth that the web UI manages for you

### 🎯 For Most Users: Use the Web UI

The web interface is designed to be **noob-friendly** and handles all the technical details:

1. Navigate to http://localhost:8080
2. Fill in the required fields:
   - **User Information**: Your name, email, and ZIP code
   - **Letter Generation** — Choose your method:
   
   | Generation Method | How it Works | What You Need | Best For |
   |-------------------|--------------|---------------|----------|
   | **AI-Powered** | Creates unique letters using ChatGPT/Claude | API key ($) | Personalized, varied content |
   | **Template-Based** | Uses pre-written letter templates | Nothing extra | Quick setup, no costs |
   
   - **Email Provider**: Configure SMTP, SendGrid, or Mailgun
   - **Representative APIs**: Add API keys for OpenStates
   - **Letter Settings**: Customize tone and length
3. Click "Save Configuration"

**What happens when you save:**
- ✅ Creates/updates your `.env` file automatically
- 🔐 Stores all settings securely in environment variables
- 💾 Configuration persists across container restarts
- 🚀 Ready to run immediately

### 🤓 For Advanced Users: Direct .env Editing

For manual configuration or if you prefer to pre-populate your settings:

**Option A: Use the initialization script**
```bash
./init-env.sh
# This copies env.example to .env, then edit your values:
nano .env
docker compose up -d
```

**Option B: Copy and edit manually**
```bash
# Copy the example file
cp env.example .env

# Edit with your values
nano .env

# Start the application
docker compose up -d
```

**Note**: The web UI can still update any `.env` file you create manually. Your manual edits will be preserved and merged with web UI changes.

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
OPENSTATES_API_KEY=your-openstates-key

# Database (Docker handles this automatically)
DATABASE_URL=postgres://lettersmith:lettersmith_pass@db:5432/lettersmith?sslmode=disable
```

### 🔧 Configuration Priority

The application loads configuration in this order:
1. **Environment Variables** (`.env` file) - Primary source
2. **Application Defaults**

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

## Contributing

We welcome contributions to Lettersmith! This project is built with privacy as a core principle and aims to empower citizen advocacy.

### Quick Start for Contributors

1. Check out the **[Development Guide](DEVELOPMENT.md)** for detailed setup instructions
2. Fork the repository and create a feature branch
3. Follow our coding standards and add tests for new features
4. Open a pull request with a clear description of your changes

### Development Guidelines

- **Privacy First**: Never log or expose sensitive data (API keys, passwords, emails)
- **User-Friendly**: Keep the web UI simple and accessible
- **Documentation**: Update both user and developer documentation
- **Testing**: Maintain good test coverage for reliability

For detailed development setup, API documentation, and technical guidelines, see [DEVELOPMENT.md](DEVELOPMENT.md).

## License

GPL v3 License - see [License File](LICENSE) for details

## Acknowledgments

- Built with privacy as a core principle
- Inspired by the need for citizen advocacy
- Thanks to all contributors and privacy advocates