# Agentic Engineering System

A dynamic, phased multi-agent AI system that functions as an elite autonomous engineering team spanning software, hardware, product, and safety departments. The system implements **phased execution** with user approval workflows, optimizing for scalability, collaboration, traceability, and controlled delivery.

## 🚀 Quick Start

1. **Prerequisites**: Docker, Docker Compose, OpenAI API key
2. **Setup**: `echo "OPENAI_API_KEY=your_key_here" > .env`
3. **Deploy**: `docker compose -f docker-compose.dev.yml up -d`
4. **Access**: Open http://localhost:8081 for Mission Control UI
5. **Test**: Create a task and watch the phased execution in real-time

## Architecture

The system implements a **phased multi-agent architecture** with these core principles:

1. **Agents as a Commodity**: There is only one type of "worker" agent service. Its specialization (persona) is defined at runtime by the instruction prompt it receives.

2. **Phased Execution**: Tasks are broken down into distinct phases by a Lead Agent, with each phase containing a maximum of 10 domain experts who execute their work in parallel without further delegation.

3. **User-Controlled Progression**: After each phase completes, the user is presented with the results and can decide whether to proceed to the next phase or stop the execution.

This creates a **controlled, phased approach** where complex projects are systematically broken down into manageable phases with clear deliverables and approval gates.

## Components

### 1. Generic Agent (Python)
- **File**: `agents/generic_agent/agent.py`
- **Role**: Universal worker that can be anything you tell it to be
- **Technology**: Python + gRPC + LiteLLM (supports OpenAI, Anthropic, etc.)
- **Key Feature**: Makes execute/delegate decisions autonomously using LLM reasoning

### 2. Orchestrator (Go)
- **File**: `orchestrator/main.go`
- **Role**: 
  - Lead Agent Coordinator (manages phased execution)
  - Domain Expert Manager (spawns and coordinates up to 10 experts per phase)
  - Phase Controller (handles phase completion and user approval)
  - Docker Manager (controls agent container lifecycle)
- **Technology**: Go + Docker SDK + gRPC + HTTP API + PostgreSQL

### 3. Mission Control UI (React/TypeScript)
- **Files**: `ui/src/*`
- **Role**: 
  - Professional three-panel Mission Control interface
  - Real-time task and phase monitoring with WebSocket updates
  - Phase completion visualization with expert tracking
  - User approval workflow interface
  - Modern responsive design with Tailwind CSS v4
- **Technology**: React + TypeScript + Vite + Tailwind CSS v4

### 4. PostgreSQL Database
- **Files**: `orchestrator/database/db.go`
- **Role**: 
  - Persistent storage for tasks and phase results
  - JSONB storage for complex phase data
  - ACID compliance and structured querying
  - Production-ready scalability
- **Technology**: PostgreSQL 15 with JSONB support

### 5. Task Tree Manager
- **File**: `orchestrator/tasktree/manager.go`
- **Role**: Manages the hierarchical task structure with thread-safe operations

### 6. Docker Manager
- **File**: `orchestrator/docker/manager.go`
- **Role**: Spawns and manages agent containers dynamically

## Prerequisites

1. **Docker & Docker Compose**: Must be installed and running
   - **macOS**: Download Docker Desktop from https://docker.com
   - **Linux**: `sudo apt-get install docker.io docker-compose-v2` (Ubuntu/Debian)
   - **Windows**: Download Docker Desktop from https://docker.com
   
2. **OpenAI API Key**: Get from https://platform.openai.com/api-keys
   
3. **For Development** (optional):
   - **Go**: Version 1.21 or later (download from https://golang.org/dl/)
   - **Node.js**: Version 18+ for UI development
   - **Python**: Version 3.10+ for agent development

## 🚀 Installation & Quick Start

### Method 1: Docker Compose (Recommended)

This is the easiest way to run the complete system with PostgreSQL database.

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd agent_inc
   ```

2. **Create environment file**:
   ```bash
   # Create .env file with your OpenAI API key
   echo "OPENAI_API_KEY=your_api_key_here" > .env
   ```

3. **Start the system**:
   ```bash
   # Start all services (PostgreSQL, Orchestrator, Generic Agent)
   docker compose -f docker-compose.dev.yml up -d
   ```

4. **Access Mission Control**:
   - Open your browser to **http://localhost:8081**
   - Submit tasks and monitor phased execution
   - Approve or reject phase progressions

5. **Verify system health**:
   ```bash
   # Check all services are running
   docker compose -f docker-compose.dev.yml ps
   
   # Test API endpoint
   curl http://localhost:8081/health
   
   # Should return: {"status":"healthy","tasks":N,"timestamp":"..."}
   ```

6. **Stop the system**:
   ```bash
   docker compose -f docker-compose.dev.yml down
   ```

### Method 2: Development Setup

For development and debugging, you can run components individually.

1. **Set your OpenAI API key**:
   ```bash
   export OPENAI_API_KEY=your_api_key_here
   ```

2. **Start PostgreSQL database**:
   ```bash
   docker compose -f docker-compose.dev.yml up -d postgres
   ```

3. **Build the generic agent image**:
   ```bash
   docker compose -f docker-compose.dev.yml build generic_agent
   ```

4. **Run the orchestrator**:
   ```bash
   cd orchestrator
   export DATABASE_URL="postgres://postgres:password@localhost:5434/agent_inc?sslmode=disable"
   go run .
   ```

5. **Run the UI (optional, for development)**:
   ```bash
   cd ui
   npm install
   npm run dev
   # Access at http://localhost:5173
   ```

## How It Works

### 1. Initial Task Submission
Submit a high-level task through the web UI or directly to the orchestrator API.

### 2. Lead Agent Planning
A Lead Agent (Project Manager persona) analyzes the task and creates a phased execution plan with:
- **Phase breakdown**: Logical phases (e.g., "Requirements and Planning", "Design and Architecture")
- **Domain experts**: Up to 10 specialists per phase with specific personas and tasks
- **No sub-delegation**: Experts in Phase 1 complete their work directly without further delegation

### 3. Phase Execution
- All domain experts in the current phase execute their tasks in parallel
- Each expert focuses on their specific domain expertise
- Real-time progress monitoring through the web UI

### 4. Phase Completion and Approval
- When all experts in a phase complete their work, the phase is marked as complete
- User is presented with all expert results and the next phase plan
- User can choose to:
  - **Approve**: Proceed to the next phase
  - **Reject**: Stop execution and review results

### 5. Iterative Progression
- Approved phases trigger the next phase execution
- Process continues until all phases complete or user stops
- Final deliverable integrates all phase results

## Example Workflow

```
Initial Task: "Create a user authentication system"
    ↓
Lead Agent creates phased plan:

Phase 1: "Requirements and Planning"
    ├── Business Analyst → Requirements gathering
    ├── UX Designer → User experience design  
    ├── Technical Architect → System architecture
    └── Security Specialist → Security requirements
        ↓
Phase 1 completes → User reviews results in Mission Control → Approves
        ↓
Phase 2: "Implementation and Testing"
    ├── Backend Developer → API implementation
    ├── Frontend Developer → UI implementation
    ├── Database Engineer → Data layer design
    ├── QA Engineer → Test plan and automation
    └── DevOps Engineer → Deployment strategy
        ↓
Phase 2 completes → User reviews results → Can approve or stop
        ↓
Final deliverable: Complete authentication system with all components
```

### Using Mission Control

1. **Submit Task**: Use the "New Mission" button to create a complex engineering task
2. **Monitor Progress**: Watch real-time updates as the Lead Agent creates phases
3. **Review Phases**: Each phase shows expert assignments and progress
4. **Approve/Reject**: When a phase completes, review results and decide to continue
5. **View Results**: Expert outputs are displayed with syntax highlighting and formatting
6. **Activity Log**: Track all system events and user decisions in the activity panel

## Key Features

### 🎯 **Phased Execution**
- Tasks broken into logical phases by Lead Agent
- Maximum 10 domain experts per phase
- No sub-delegation in Phase 1 for focused execution

### 👤 **User Control**
- Approval gates between phases
- Real-time monitoring through web UI
- Ability to stop execution at any phase

### 🔄 **Parallel Processing**
- All experts in a phase execute simultaneously
- Efficient resource utilization
- Faster completion times

### 🎭 **Dynamic Specialization**
- Expert personas defined at runtime
- Task-specific domain expertise
- Flexible agent assignment

### 🔍 **Full Transparency**
- Complete phase and expert tracking
- Real-time progress monitoring
- Detailed result presentation

### 🛡️ **Controlled Progression**
- User approval required between phases
- Risk mitigation through staged delivery
- Quality gates at each phase

## Configuration

### Environment Variables
- `OPENAI_API_KEY`: Required for LLM access
- `DATABASE_URL`: PostgreSQL connection string (auto-configured in Docker)
- `DOCKER_HOST`: Override Docker connection (optional)

### Services Configuration

#### PostgreSQL Database
- **Host**: `localhost:5434` (external), `postgres:5432` (internal)
- **Database**: `agent_inc`
- **Username**: `postgres`
- **Password**: `password`
- **Data**: Persisted in Docker volume `postgres_data`

#### Orchestrator Service
- **Port**: `8081` (external), `8080` (internal)
- **Database**: Automatic connection to PostgreSQL
- **Docker**: Manages agent containers on dynamic ports (50060+)

#### Mission Control UI
- **Development**: `http://localhost:5173` (Vite dev server)
- **Production**: `http://localhost:8081` (served by orchestrator)
- **Features**: Real-time WebSocket updates, phase approval workflow

### Model Configuration
Modify in `agents/generic_agent/agent.py`:
- **Decision Model**: `gpt-4o` (for phased planning)
- **Execution Model**: `gpt-4-turbo` (for domain expert tasks)

### Phase Configuration
Modify in `orchestrator/main.go`:
- **Max Experts per Phase**: 10 (configurable)
- **Phase Timeout**: Adjustable per phase type
- **Approval Timeout**: User approval wait time

### Container Configuration  
Modify in `orchestrator/docker/manager.go`:
- **Port Range**: Starting from 50060
- **Startup Delay**: 3 seconds for container readiness
- **Image Name**: `agentic-engineering-system_generic_agent`

## Monitoring & Debugging

### Mission Control Dashboard
The professional React interface provides:
- **Real-time Task Monitoring**: Live WebSocket updates on task progress
- **Three-Panel Layout**: Sidebar navigation, main content, activity panel
- **Phase Visualization**: Clear phase breakdown with progress indicators
- **Expert Tracking**: Individual expert progress and results
- **Approval Workflow**: User-friendly phase approval interface
- **Modern Design**: Professional UI with Tailwind CSS v4 styling

### API Endpoints
The orchestrator exposes these endpoints at `http://localhost:8081`:
- `POST /api/task` - Submit new task
- `GET /api/task/{id}` - Get task status
- `GET /api/phase/{taskId}/{phaseId}` - Get phase results
- `POST /api/phases/approve` - Approve/reject phase
- `GET /health` - System health check
- `WS /ws` - WebSocket for real-time updates

### Container Management
View running containers:
```bash
# See all system containers
docker compose -f docker-compose.dev.yml ps

# View orchestrator logs
docker compose -f docker-compose.dev.yml logs orchestrator

# View PostgreSQL logs
docker compose -f docker-compose.dev.yml logs postgres

# View agent containers (created dynamically)
docker ps | grep generic_agent
```

### Database Management
Access PostgreSQL directly:
```bash
# Connect to database
docker compose -f docker-compose.dev.yml exec postgres psql -U postgres -d agent_inc

# View tables
\dt

# Check task data
SELECT id, task, status FROM task_executions ORDER BY created_at DESC LIMIT 5;
```

### Logs
The system provides comprehensive logging:
- **Phase Planning**: Lead agent decision-making
- **Expert Execution**: Individual domain expert progress
- **Container Management**: Docker lifecycle events
- **API Communication**: HTTP and gRPC interactions
- **Database Operations**: PostgreSQL queries and transactions

### Phase State Inspection
The system maintains detailed phase state:
- **Phase Status**: Planning, executing, completed, approved
- **Expert Status**: Individual expert progress tracking
- **Result Aggregation**: Phase deliverable compilation
- **User Decisions**: Approval/rejection history

## Extending the System

### Adding New Phase Types
Modify the Lead Agent prompts in `orchestrator/main.go` to include new phase templates.

### Custom Expert Personas
Add specialized domain expert personas by extending the phase planning logic.

### Integration APIs
Extend the HTTP API in `orchestrator/main.go` for external system integration.

### UI Customization
Modify React components in `ui/src/` for custom user experiences.

### Alternative LLM Providers
LiteLLM supports many providers - just change the model name in the agent code.

## Troubleshooting

### Common Issues

1. **"Docker daemon not running"**
   - Ensure Docker Desktop is running
   - Check Docker socket permissions

2. **"OPENAI_API_KEY not set"**
   - Create `.env` file with `OPENAI_API_KEY=your_key`
   - Or export the environment variable: `export OPENAI_API_KEY=your_key`

3. **"Port already in use"**
   - Stop existing containers: `docker compose -f docker-compose.dev.yml down`
   - Check for conflicting services on ports 8081, 5434

4. **"Mission Control UI not loading"**
   - Verify orchestrator is running: `curl http://localhost:8081/health`
   - Check orchestrator logs: `docker compose -f docker-compose.dev.yml logs orchestrator`
   - Ensure port 8081 is accessible

5. **"Database connection failed"**
   - Verify PostgreSQL is running: `docker compose -f docker-compose.dev.yml logs postgres`
   - Check database health: `docker compose -f docker-compose.dev.yml exec postgres pg_isready -U postgres`
   - Verify connection: `docker compose -f docker-compose.dev.yml exec postgres psql -U postgres -d agent_inc -c "SELECT version();"`

6. **"Phase not progressing"**
   - Check that all experts in the phase have completed
   - Review expert logs for any failures
   - Verify user approval is not required

7. **"Expert containers not spawning"**
   - Verify Docker daemon is running
   - Check generic agent image exists: `docker images | grep generic_agent`
   - Review container logs for startup issues
   - Ensure OpenAI API key is valid

8. **"WebSocket connection failed"**
   - Check browser console for WebSocket errors
   - Verify orchestrator WebSocket endpoint: `ws://localhost:8081/ws`
   - Ensure no firewall blocking WebSocket connections

### Development Issues

9. **"Build failed"**
   - Ensure Go 1.21+ is installed: `go version`
   - Check for missing dependencies: `go mod tidy`
   - Verify Docker Compose file syntax

10. **"UI development server won't start"**
    - Ensure Node.js 18+ is installed: `node --version`
    - Install dependencies: `cd ui && npm install`
    - Check for port conflicts on 5173

### System Health Checks

```bash
# Check all services
docker compose -f docker-compose.dev.yml ps

# Test API connectivity
curl http://localhost:8081/health

# Test database connectivity
docker compose -f docker-compose.dev.yml exec postgres pg_isready -U postgres

# View system logs
docker compose -f docker-compose.dev.yml logs --tail 50
```

### Performance Considerations

- **Parallel Phase Execution**: All experts in a phase run concurrently
- **Resource Management**: Containers cleaned up immediately after expert completion
- **Model Selection**: Balance between capability and cost/speed per expert
- **UI Responsiveness**: Real-time updates via WebSocket without overwhelming the interface
- **Phase Optimization**: Logical phase boundaries reduce unnecessary work
- **Database Performance**: PostgreSQL with JSONB indexing for fast queries

## Security Notes

- **API Keys**: Never commit API keys to version control. Use `.env` files or environment variables
- **Container Isolation**: Containers provide security boundaries between agents
- **Network Security**: gRPC communication is local-only by default
- **Database Security**: PostgreSQL credentials should be changed for production deployments
- **Secrets Management**: Consider using Docker secrets or external secret management for production
- **Access Control**: Mission Control UI has no authentication - add security for production use

## Future Enhancements

- **Advanced UI Features**: Gantt charts, phase analytics, expert performance metrics
- **Multi-Cloud Support**: Deploy across cloud providers with Kubernetes
- **Custom Expert Types**: Specialized agent types beyond generic personas
- **Integration Hooks**: Webhooks and API integrations for external tools
- **Phase Templates**: Pre-defined phase patterns for common project types
- **Collaborative Features**: Multi-user approval workflows with role-based access
- **Advanced Database Features**: Task history, audit trails, performance analytics
- **Monitoring & Observability**: Prometheus metrics, distributed tracing, alerting
- **Auto-scaling**: Dynamic container scaling based on workload
- **Multi-tenant Support**: Isolated environments for different organizations

## Architecture Decisions

### Database Choice: PostgreSQL
- **Structured Data**: Moved from BoltDB to PostgreSQL for better querying and relationships
- **JSONB Storage**: Complex phase data stored as JSONB for flexibility with SQL queries
- **ACID Compliance**: Ensures data consistency across concurrent operations
- **Scalability**: Production-ready with replication and backup capabilities

### UI Architecture: Mission Control
- **Three-Panel Design**: Optimized for task management workflows
- **Real-time Updates**: WebSocket integration for instant feedback
- **Professional Styling**: Tailwind CSS v4 for modern, responsive design
- **Component Architecture**: Modular React components for maintainability

### Container Orchestration
- **Docker Compose**: Simplified deployment and service management
- **Dynamic Scaling**: Agents created on-demand with automatic cleanup
- **Network Isolation**: Services communicate through Docker networks
- **Volume Persistence**: Database data persisted across container restarts

## License

This project demonstrates a production-ready implementation of a phased multi-agent AI system with user-controlled progression and approval workflows. It showcases modern software architecture principles with real-time monitoring, PostgreSQL database backend, containerized deployment, and professional Mission Control interface.

## Project Status

- ✅ **Production Ready**: Full Docker Compose deployment with PostgreSQL
- ✅ **Mission Control UI**: Professional three-panel interface with real-time updates
- ✅ **Database Migration**: Upgraded from BoltDB to PostgreSQL for scalability
- ✅ **Container Orchestration**: Dynamic agent management with Docker
- ✅ **Phase Management**: Complete workflow with user approval gates
- ✅ **Real-time Monitoring**: WebSocket integration for live updates

## Related Files

- **[QUICKSTART.md](QUICKSTART.md)**: 5-minute setup guide
- **[docker-compose.dev.yml](docker-compose.dev.yml)**: Development deployment configuration
- **[build.sh](build.sh)**: System build and validation script
- **[IMPLEMENTATION.md](IMPLEMENTATION.md)**: Technical implementation details (if available)

---

**Need Help?** Check the troubleshooting section above or review the logs for specific error messages. For development questions, examine the codebase structure and component interactions.
