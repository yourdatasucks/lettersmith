-- Initial schema for lettersmith

-- Users table (minimal data - privacy first)
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL, -- For sending letter copies
    name VARCHAR(255) NOT NULL, -- For signing letters
    zip_code VARCHAR(10) NOT NULL, -- For representative lookup
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Representatives table
CREATE TABLE IF NOT EXISTS representatives (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    title VARCHAR(100) NOT NULL, -- Senator, Representative
    state VARCHAR(2) NOT NULL,
    district VARCHAR(50), -- For House representatives
    party VARCHAR(50),
    email VARCHAR(255),
    phone VARCHAR(50),
    office_address TEXT,
    website VARCHAR(255),
    external_id VARCHAR(100), -- ID from external API
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(external_id)
);

-- Letters table
CREATE TABLE IF NOT EXISTS letters (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    representative_id INTEGER REFERENCES representatives(id) ON DELETE CASCADE,
    subject VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    ai_provider VARCHAR(50) NOT NULL, -- openai, anthropic
    ai_model VARCHAR(100) NOT NULL,
    theme VARCHAR(255) NOT NULL,
    tone VARCHAR(50) NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE,
    email_provider VARCHAR(50), -- smtp, sendgrid, mailgun
    email_status VARCHAR(50) DEFAULT 'pending', -- pending, sent, failed
    email_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Configuration table (for storing app-wide settings)
CREATE TABLE IF NOT EXISTS configurations (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Scheduled jobs table
CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    job_type VARCHAR(100) NOT NULL, -- daily_letters
    schedule_time TIME NOT NULL,
    timezone VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    last_run_at TIMESTAMP WITH TIME ZONE,
    next_run_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, job_type)
);

-- Create indexes
CREATE INDEX idx_representatives_state ON representatives(state);
CREATE INDEX idx_letters_user_id ON letters(user_id);
CREATE INDEX idx_letters_sent_at ON letters(sent_at);
CREATE INDEX idx_letters_status ON letters(email_status);
CREATE INDEX idx_scheduled_jobs_next_run ON scheduled_jobs(next_run_at) WHERE enabled = true;

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_representatives_updated_at BEFORE UPDATE ON representatives
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_configurations_updated_at BEFORE UPDATE ON configurations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scheduled_jobs_updated_at BEFORE UPDATE ON scheduled_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column(); 