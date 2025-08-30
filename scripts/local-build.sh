#!/bin/bash
# Local build script for development and testing

set -e

echo "🔨 Building tw2dynalist locally..."

# Build the Docker image
docker build -t tw2dynalist:local .

echo "✅ Build completed successfully!"
echo ""
echo "To run locally with your .env file:"
echo "docker run --rm -it --env-file .env -p 8080:8080 -v \$(pwd)/data:/app/data tw2dynalist:local"
echo ""
echo "Or use docker-compose for easier management:"
echo "docker-compose up -d"