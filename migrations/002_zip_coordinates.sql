-- ZIP Code to Coordinates mapping for OpenStates API calls
CREATE TABLE IF NOT EXISTS zip_coordinates (
    zip_code VARCHAR(5) PRIMARY KEY,
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    city VARCHAR(100),
    state VARCHAR(2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_zip_coordinates_state ON zip_coordinates(state);
CREATE INDEX IF NOT EXISTS idx_zip_coordinates_city ON zip_coordinates(city);

-- Add updated_at trigger
CREATE TRIGGER update_zip_coordinates_updated_at BEFORE UPDATE ON zip_coordinates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Note: Actual data will be populated by the application startup process
-- This ensures we always have the most recent data available 