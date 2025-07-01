#!/bin/bash

# Build script for the Agentic Engineering System

set -e

echo "=== Building Agentic Engineering System ==="

# Check if OPENAI_API_KEY is set
if [ -z "$OPENAI_API_KEY" ]; then
    echo "Error: OPENAI_API_KEY environment variable is required"
    echo "Please set it with: export OPENAI_API_KEY=your_api_key_here"
    exit 1
fi

# Check if Docker is installed and running
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed"
    echo "Please install Docker from https://docker.com"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo "Error: Docker is not running"
    echo "Please start Docker Desktop or the Docker service"
    exit 1
fi

# Check if we can use docker compose or docker-compose
if command -v "docker compose" &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    echo "Error: Neither 'docker compose' nor 'docker-compose' is available"
    echo "Please install Docker Compose"
    exit 1
fi

# Build the generic agent Docker image
echo "Building generic agent Docker image..."
$DOCKER_COMPOSE build generic_agent

echo "=== Build Complete ==="
echo ""
echo "To run the system:"
echo "1. Ensure Docker is running"
echo "2. Run: cd orchestrator && go run ."
echo ""
echo "The system will:"
echo "1. Create agent containers dynamically"
echo "2. Execute tasks recursively"
echo "3. Synthesize results hierarchically"
echo "4. Clean up containers automatically"
