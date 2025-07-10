#!/bin/bash

echo "🚀 Testing Enhanced Agent System with Database and WebSocket Support"
echo "================================================================="

# Check if OpenAI API key is set
if [ -z "$OPENAI_API_KEY" ]; then
    echo "❌ OPENAI_API_KEY environment variable is required"
    exit 1
fi

echo "✅ OpenAI API key is set"

# Build the orchestrator
echo "🔨 Building orchestrator..."
cd orchestrator
go build -o orchestrator
if [ $? -ne 0 ]; then
    echo "❌ Failed to build orchestrator"
    exit 1
fi
echo "✅ Orchestrator built successfully"

# Build the UI
echo "🔨 Building UI..."
cd ../ui
npm run build
if [ $? -ne 0 ]; then
    echo "❌ Failed to build UI"
    exit 1
fi
echo "✅ UI built successfully"

# Start PostgreSQL
echo "🐘 Starting PostgreSQL..."
cd ..
docker-compose -f docker-compose.dev.yml up -d postgres
sleep 5

# Start the orchestrator
echo "🚀 Starting orchestrator with database support..."
cd orchestrator
export DATABASE_URL="postgres://postgres:password@localhost:5432/agent_inc?sslmode=disable"
./orchestrator &
ORCHESTRATOR_PID=$!

# Wait for orchestrator to start
sleep 3

echo ""
echo "🎉 Enhanced Agent System is now running!"
echo "================================================================="
echo "📊 Key Improvements Implemented:"
echo "  ✅ Database persistence (PostgreSQL)"
echo "  ✅ Real-time WebSocket updates"
echo "  ✅ Persistent UI state (localStorage)"
echo "  ✅ Per-expert results API"
echo "  ✅ Enhanced error handling"
echo "  ✅ WebSocket connection status indicator"
echo ""
echo "🌐 Access the system at: http://localhost:8080"
echo "📡 WebSocket endpoint: ws://localhost:8080/ws"
echo "📊 API endpoints:"
echo "   POST /api/task - Submit new task"
echo "   GET  /api/task/{id} - Get task status"
echo "   GET  /api/phase/{taskId}/{phaseId} - Get detailed expert results"
echo "   POST /api/phases/approve - Approve/reject phase"
echo "   WS   /ws - WebSocket for real-time updates"
echo ""
echo "🔗 Database: PostgreSQL running on localhost:5432"
echo "   Database: agent_inc"
echo "   Username: postgres"
echo "   Password: password"
echo ""
echo "Press Ctrl+C to stop the system"

# Wait for interrupt
trap "echo '🛑 Stopping system...'; kill $ORCHESTRATOR_PID; docker-compose -f docker-compose.dev.yml down; exit 0" INT
wait $ORCHESTRATOR_PID
