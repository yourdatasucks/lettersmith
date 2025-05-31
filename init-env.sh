#!/bin/bash

# Initialize .env file for Lettersmith
# This creates the .env file required by Docker Compose

echo "🚀 Lettersmith Environment Setup"
echo "================================"

if [ ! -f .env ]; then
    echo "Creating .env file from template..."
    cp env.example .env
    echo "✅ .env file created successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Run: docker compose up -d"
    echo "2. Open: http://localhost:8080"
    echo "3. Configure via the web interface"
else
    echo "ℹ️  .env file already exists"
    echo ""
    echo "You can now run: docker compose up -d"
fi 