# Lettersmith

A containerized Go application that uses AI to generate and send unique letters to US representatives advocating for privacy and consumer protection laws.

## Features

### ✅ Working Features
- 🔍 **Representative lookup** via OpenStates API integration
- 🐳 **Fully containerized** with Docker Compose
- 🌐 **Web UI** for configuration and system monitoring
- 📊 **PostgreSQL database** with automatic migrations on startup
- 🔧 **Smart configuration management** with web UI persistence
- 🔒 **Privacy-first design**: Minimal data collection (only name, email, ZIP code)
- ✅ **System status dashboard** with real-time health checks
- 🗺️ **ZIP code to coordinates conversion** using US Census Bureau data
- 📧 **Email configuration** and testing (SMTP/SendGrid/Mailgun)

### 🔧 In Development
- 🤖 **AI letter generation** (OpenAI/Anthropic client interfaces exist, functionality in progress)

### 📋 Planned Features  
- 📧 **Automated daily email sending** to representatives
- 📝 **Template-based letter generation** as alternative to AI
- 📈 **Letter history and audit trail**
- ⏰ **Scheduler** for automated sending

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
- `ghcr.io/yourdatasucks/lettersmith:latest` - Latest stable release
- `ghcr.io/yourdatasucks/lettersmith:dev` - Latest development version  
- `ghcr.io/yourdatasucks/lettersmith:v1.0.0` - Specific version releases

**Note:** Docker Compose automatically uses the image specified in your `.env` file (`DOCKER_IMAGE` variable).

4. Open http://localhost:8080 in your browser to configure the application

5. Fill in your information and API keys, then click "Save Configuration"

**That's it!** The web UI will update your `.env` file automatically and persist your configuration across container restarts.

### 🚀 Current Implementation Status

**✅ Ready to Use:**
- Complete system configuration and monitoring
- Representative lookup and management (OpenStates integration)
- Email configuration and testing  
- ZIP code geocoding system
- Full web interface with real-time status

**🔧 Next: AI Letter Generation**  
The foundation is complete! AI letter generation is the primary development focus to enable automated advocacy.

### 🎯 Key Web Interface Features

Once configured, explore these interfaces:

- **📊 System Status** (`/status.html`) - Real-time health monitoring of all services
- **👥 Representatives** (`/representatives.html`) - Manage your representatives data
- **⚙️ Configuration** (`/`) - Update settings and test email configuration

The system status dashboard shows the health of:
- ✅ Database connectivity  
- ✅ Email configuration
- ✅ AI provider setup (when configured)
- ✅ ZIP geocoding service
- ✅ Representatives API integration

## Configuration

Lettersmith uses a **simple, user-friendly configuration system**:

- **Web UI** - Easy point-and-click configuration for non-technical users
- **`.env` file** - The source of truth that the web UI manages for you

### 🎯 For Most Users: Use the Web UI

The web interface is designed to be **user friendly** and handles all the technical details:

1. Navigate to http://localhost:8080
2. Fill in the required fields:
   - **User Information**: Your name, email, and ZIP code
   - **Letter Generation** — Choose your method:
   
   | Generation Method | How it Works | What You Need | Best For | Status |
   |-------------------|--------------|---------------|----------|---------|
   | **AI-Powered** | Creates unique letters using OpenAI/Anthropic | API key ($) | Personalized, varied content | 🔧 Client interfaces exist, functionality in progress |
   | **Template-Based** | Uses pre-written letter templates | Nothing extra | Quick setup, no costs | 📋 Planned feature |
   
   - **Email Provider**: Configure SMTP, SendGrid, or Mailgun
   - **Representative APIs**: Add API keys for OpenStates
   - **Letter Settings**: Customize tone and length
3. Click "Save Configuration"

**What happens when you save:**
- ✅ Creates/updates your `.env` file automatically
- 🔐 Stores all settings securely in environment variables
- 💾 Configuration persists across container restarts
- 🚀 Ready to run immediately

### For Advanced Users: Direct .env Editing

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

This application practices data minimization and transparency:
- **Minimal data collection**: Only name, email, and ZIP code required
- **No phone numbers** collected or stored
- **No street addresses** stored
- **No tracking or analytics**
- **Self-hosted**: All data stays in your own PostgreSQL database
- **Transparent processing**: ZIP code converted to coordinates for representative lookup only
- **Privacy-focused recommendations**: Supports ProtonMail and other privacy-focused email providers
- **Open source**: All code is auditable for transparency

## Representative Lookup APIs

The project uses the OpenStates API for privacy-respecting representative data:

- **OpenStates API**: State and federal legislature data (free tier available)
  - Get your free API key at [openstates.org/api/](https://openstates.org/api/)
  - Covers all US states and federal representatives
  - Uses geographic coordinates for precise district matching

### ZIP Code to Coordinates Conversion

Lettersmith automatically converts ZIP codes to latitude/longitude coordinates for the OpenStates API, which requires geographic coordinates rather than postal codes.

**How it works:**
1. **Official Source**: Downloads data from the **US Census Bureau's ZIP Code Tabulation Areas (ZCTA) Gazetteer** - the authoritative government source
2. **Future-Proof Loading**: Automatically tries multiple URL patterns and years to handle Census Bureau reorganizations
3. **Automatic Loading**: On first startup, the application downloads the latest available Census Bureau data  
4. **Fallback Protection**: If all Census Bureau URLs fail, it uses an embedded dataset of major US cities
5. **Docker-Friendly**: No manual intervention required - data is ready when the container starts
6. **Smart Updates**: Set `ZIP_DATA_UPDATE=true` to refresh data on startup (default for new installations)
7. **Configurable**: Override URLs via `CENSUS_BUREAU_URL` environment variable if needed

**Future-Proof URL Strategy:**
```
✅ 2024: https://...2024_Gazetteer/2024_Gaz_zcta_national.zip
✅ 2023: https://...2023_Gazetteer/2023_Gaz_zcta_national.zip  (fallback)
✅ 2022: https://...2022_Gazetteer/2022_Gaz_zcta_national.zip  (fallback)
✅ Alternative: https://...gazetteer/Gaz_zcta_national.zip     (no year)
✅ Alternative: https://...gazetteer/current/zcta_national.zip  (current)
✅ Custom: Set CENSUS_BUREAU_URL=your-url                      (override)
```

**Data Sources:**
- **Primary**: US Census Bureau ZIP Code Tabulation Areas (ZCTA) - Official government data, comprehensive & accurate
- **Fallback**: Embedded dataset of 15 major US metropolitan areas

This ensures reliable ZIP-to-coordinate conversion without external API dependencies during runtime, using the most authoritative data available, and automatically adapts to Census Bureau URL changes.

## Project Structure

```
lettersmith/
├── cmd/
│   ├── server/          # Main application server ✅
│   │   └── main.go      # HTTP server with automatic migrations
│   └── migrate/         # Database migration tool ✅ (optional - migrations auto-run on startup)
├── internal/
│   ├── config/          # Environment variable configuration ✅
│   ├── api/             # AI provider interfaces 🔧 (clients exist, functionality in progress)
│   ├── email/           # Email sending logic ✅
│   ├── reps/            # Representative lookup ✅ (OpenStates integration)
│   ├── geocoding/       # ZIP to coordinates conversion ✅ (US Census Bureau)
│   └── scheduler/       # Daily job runner 📋 (planned)
├── web/                 # Frontend static files ✅
│   ├── index.html       # Configuration UI ✅
│   ├── status.html      # System status dashboard ✅
│   ├── representatives.html # Representatives management ✅
│   ├── style.css        # Modern, privacy-focused styling ✅
│   └── app.js           # Frontend logic with .env management ✅
├── migrations/          # SQL migration files ✅ (auto-applied on startup)
├── docker-compose.yml   # Docker Compose configuration ✅
├── Dockerfile           # Multi-stage build ✅ (final image: <20MB)
├── env.example          # Example environment variables ✅
└── .env                 # Your configuration (created/managed by web UI) ✅
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

1. **Fork the repository** on GitHub
2. **Choose your development approach** (see [DEVELOPMENT.md](DEVELOPMENT.md) for details):
   - **Option A**: Use pre-built dev image (easiest for most contributors)
   - **Option B**: Build locally with Docker
   - **Option C**: Native Go development (fastest for active development)
3. **Create a feature branch** from `dev`: `git checkout -b feature/amazing-feature dev`
4. **Make your changes** following our coding standards
5. **Submit a PR** to the `dev` branch

### Development Environment Options

**For Most Contributors (Option A):**
```bash
git clone https://github.com/YOUR-USERNAME/lettersmith.git
cd lettersmith
git checkout dev
./init-env.sh
docker compose up -d  # Uses ghcr.io/yourdatasucks/lettersmith:dev
```

**For Docker Development (Option B):**
```bash
git clone https://github.com/YOUR-USERNAME/lettersmith.git
cd lettersmith
./init-env.sh
docker build -t lettersmith:local .
export DOCKER_IMAGE=lettersmith:local
docker compose up -d
```

**For Native Go Development (Option C):**
```bash
git clone https://github.com/YOUR-USERNAME/lettersmith.git
cd lettersmith
createdb lettersmith
export DATABASE_URL="postgres://localhost/lettersmith?sslmode=disable"
go run cmd/server/main.go  # Migrations run automatically on startup
```

### Development Guidelines

- **Privacy First**: Never log or expose sensitive data (API keys, passwords, emails)
- **User-Friendly**: Keep the web UI simple and accessible
- **Documentation**: Update both user and developer documentation
- **Testing**: Unit tests are optional - ensure your changes work end-to-end via the web interface
- **Go Standards**: Use `go fmt`, `go vet`, and ensure code quality

**Next Priority:** AI letter generation functionality is the primary development focus.

For detailed development setup, API documentation, and technical guidelines, see [DEVELOPMENT.md](DEVELOPMENT.md).

## License

GPL v3 License - see [License File](LICENSE) for details

## Acknowledgments

- Built with privacy as a core principle
- Inspired by the need for citizen advocacy
- Thanks to all contributors and privacy advocates