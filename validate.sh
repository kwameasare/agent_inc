#!/bin/bash

# System validation script for Agentic Engineering System

set -e

echo "=== Agentic Engineering System - Validation Script ==="
echo ""

# Check prerequisites
echo "1. Checking prerequisites..."

# Check Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed"
    exit 1
else
    echo "✅ Go is installed: $(go version)"
fi

# Check Python
if ! command -v python3 &> /dev/null; then
    echo "❌ Python3 is not installed"
    exit 1
else
    echo "✅ Python3 is installed: $(python3 --version)"
fi

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed"
    exit 1
else
    echo "✅ Docker is installed: $(docker --version)"
fi

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo "❌ Docker is not running"
    exit 1
else
    echo "✅ Docker is running"
fi

echo ""

# Check Go build
echo "2. Validating Go build..."
cd orchestrator
if go build -o orchestrator .; then
    echo "✅ Go build successful"
    rm -f orchestrator
else
    echo "❌ Go build failed"
    exit 1
fi

echo ""

# Check Python syntax
echo "3. Validating Python agent..."
cd ../agents/generic_agent
if python3 -m py_compile agent.py; then
    echo "✅ Python agent syntax valid"
else
    echo "❌ Python agent syntax error"
    exit 1
fi

# Check Python dependencies
echo "4. Checking Python dependencies..."
if python3 -c "import grpc, litellm, json, logging" 2>/dev/null; then
    echo "✅ Required Python packages available"
else
    echo "⚠️  Some Python packages may be missing. Run: pip3 install -r requirements.txt"
fi

echo ""

# Check protobuf files
echo "5. Validating protobuf files..."
if [ -f "agent_pb2.py" ] && [ -f "agent_pb2_grpc.py" ]; then
    echo "✅ Python protobuf files exist"
else
    echo "❌ Python protobuf files missing"
    exit 1
fi

cd ../../orchestrator
if [ -f "proto/agentic-engineering-system/proto/agent.pb.go" ] && [ -f "proto/agentic-engineering-system/proto/agent_grpc.pb.go" ]; then
    echo "✅ Go protobuf files exist"
else
    echo "❌ Go protobuf files missing"
    exit 1
fi

echo ""

# Check API key
echo "6. Checking environment..."
if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  OPENAI_API_KEY not set. This is required for running the system."
    echo "   Set it with: export OPENAI_API_KEY=your_api_key_here"
else
    echo "✅ OPENAI_API_KEY is set"
fi

echo ""

# Check Docker compose
echo "7. Validating Docker configuration..."
cd ..
if docker compose config &> /dev/null; then
    echo "✅ Docker compose configuration valid"
else
    echo "❌ Docker compose configuration invalid"
    exit 1
fi

echo ""
echo "=== Validation Complete ==="
echo ""
echo "✅ System is ready to run!"
echo ""
echo "To start the system:"
echo "1. Set your API key: export OPENAI_API_KEY=your_key"
echo "2. Build agent image: ./build.sh"
echo "3. Run orchestrator: cd orchestrator && go run ."
echo ""
echo "The system will automatically:"
echo "- Spawn agent containers dynamically"
echo "- Execute tasks recursively"
echo "- Synthesize results hierarchically"
echo "- Clean up resources automatically"
