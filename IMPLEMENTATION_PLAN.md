# Lettersmith Implementation Plan

## Current Status

✅ **Working Components:**
- Configuration web UI with .env management
- HTTP server with health checks and config APIs
- SMTP email client with connection testing
- PostgreSQL database with comprehensive schema
- ZIP-to-coordinates geocoding service
- Docker containerization

❌ **Missing Core Components:**
- AI letter generation (OpenAI/Anthropic clients)
- Representative lookup (OpenStates API integration)
- Letter generation engine
- Automated scheduler
- Template-based generation
- System status dashboard

## Phase 1: System Health Monitoring (DONE ✅)

- [x] Add `/api/system/status` endpoint
- [x] Check database connectivity
- [x] Test email configuration
- [x] Validate AI provider setup
- [x] Monitor geocoding service
- [x] Report missing components

## Phase 2: Core AI Integration (PRIORITY)

### 2.1 OpenAI Client Implementation
```bash
internal/api/openai.go     # OpenAI API client
internal/api/client.go     # Common AI interface
```

**Key Features:**
- Letter generation with configurable prompts
- API key validation
- Error handling and retries
- Cost tracking

### 2.2 Anthropic Client Implementation  
```bash
internal/api/anthropic.go  # Anthropic API client
```

**Key Features:**
- Claude API integration
- Similar interface to OpenAI client
- Rate limiting compliance

## Phase 3: Representative Lookup Service

### 3.1 OpenStates Integration
```bash
internal/reps/openstates.go    # OpenStates API client
internal/reps/client.go        # Representative interface
```

**Key Features:**
- Find representatives by ZIP/coordinates
- Cache representative data
- Handle API rate limits
- Privacy-focused data collection

### 3.2 Representative Database Management
```bash
internal/reps/storage.go       # Database operations
```

**Key Features:**
- Store and update representative information
- Deduplicate representatives
- Track last contact dates

## Phase 4: Letter Generation Engine

### 4.1 Core Generation Logic
```bash
internal/letters/generator.go  # Main generation engine
internal/letters/prompts.go    # AI prompt templates
```

**Key Features:**
- AI-powered letter generation
- Template-based generation
- Theme and tone customization
- Variable substitution (name, rep, ZIP, etc.)

### 4.2 Template System
```bash
internal/letters/templates.go  # Template management
templates/                     # Template files directory
```

**Key Features:**
- Markdown-based templates
- Rotation strategies (random, sequential, unique)
- Personalization variables
- Theme-based templates

## Phase 5: Scheduler Implementation

### 5.1 Background Job System
```bash
internal/scheduler/scheduler.go    # Main scheduler
internal/scheduler/jobs.go         # Job definitions
```

**Key Features:**
- Daily letter sending
- Timezone handling
- User-specific schedules
- Error handling and retries

### 5.2 Job Management
- Queue management
- Status tracking
- Failed job recovery
- Manual trigger capabilities

## Phase 6: Enhanced Web Interface

### 6.1 Status Dashboard
```bash
web/dashboard.html         # System status page
web/dashboard.js           # Dashboard logic
```

**Key Features:**
- Real-time system health
- Service status indicators
- Recent activity logs
- Quick action buttons

### 6.2 Letter Management
```bash
web/letters.html          # Letter history/management
web/test.html             # Manual testing interface
```

**Key Features:**
- View sent letters
- Test letter generation
- Representative preview
- Manual sending

## Phase 7: Testing & Production Readiness

### 7.1 Comprehensive Testing
- Unit tests for all components
- Integration tests
- Email delivery testing
- AI provider testing

### 7.2 Production Features
- Metrics and monitoring
- Log aggregation
- Health checks
- Error reporting

## Implementation Priority Order

### Week 1: AI Integration
1. Implement OpenAI client (`internal/api/openai.go`)
2. Add letter generation endpoint (`POST /api/letters/generate`)
3. Create basic prompt templates
4. Add AI testing to system status

### Week 2: Representative Lookup
1. Implement OpenStates client (`internal/reps/openstates.go`)
2. Add representative lookup endpoint (`GET /api/representatives`)
3. Integrate with geocoding service
4. Test representative finding by ZIP

### Week 3: Letter Generation Engine
1. Create letter generator (`internal/letters/generator.go`)
2. Implement full letter generation flow
3. Add manual letter sending endpoint (`POST /api/letters/send`)
4. Create template system basics

### Week 4: Scheduler & Dashboard
1. Implement basic scheduler (`internal/scheduler/scheduler.go`)
2. Create system status dashboard (`web/dashboard.html`)
3. Add letter history viewing
4. Polish and testing

## Quick Win: Immediate Value

**Create a working end-to-end flow:**
1. User configures system via web UI ✅
2. System validates all services via `/api/system/status` ✅
3. User can generate a test letter via AI
4. User can find their representatives
5. User can send a letter manually
6. System provides confirmation and logging

This gives immediate value while building toward full automation.

## API Endpoints Roadmap

```bash
# Configuration (DONE ✅)
GET  /api/health
GET  /api/config
POST /api/config
GET  /api/config/debug
POST /api/config/test-email
GET  /api/system/status ✅

# Letter Generation (TODO)
POST /api/letters/generate     # Generate test letter
POST /api/letters/send        # Send letter manually
GET  /api/letters/history     # View sent letters

# Representatives (TODO)  
GET  /api/representatives     # Find reps by ZIP
GET  /api/representatives/test # Test rep lookup

# Scheduler (TODO)
POST /api/scheduler/trigger   # Manual trigger
GET  /api/scheduler/status    # Check job status
```

## Development Environment Setup

```bash
# Start with Docker
docker compose up -d

# Test current functionality
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/status

# Configure via web UI
open http://localhost:8080
```

## Success Metrics

- [ ] User can configure system completely via web UI
- [ ] System status shows all green checks
- [ ] User can generate a test letter via AI
- [ ] User can find their representatives automatically
- [ ] User can send letters manually
- [ ] Scheduler can send letters automatically daily
- [ ] System provides full audit trail of sent letters

This plan transforms Lettersmith from a configuration-only tool into a fully functional privacy advocacy platform. 