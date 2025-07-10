# Agentic Engineering System

A dynamic, phased multi-agent AI system that functions as an elite autonomous engineering team spanning software, hardware, product, and safety departments. The system implements **phased execution** with user approval workflows, optimizing for scalability, collaboration, traceability, and controlled delivery.

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
- **Technology**: Go + Docker SDK + gRPC + HTTP API for UI

### 3. Web UI (React/Vite)
- **Files**: `ui/src/*`
- **Role**: 
  - Real-time task and phase monitoring
  - Phase completion visualization
  - User approval workflow interface
  - Expert status and progress tracking
- **Technology**: React + TypeScript + Vite + Tailwind CSS

### 4. Task Tree Manager
- **File**: `orchestrator/tasktree/manager.go`
- **Role**: Manages the hierarchical task structure with thread-safe operations

### 5. Docker Manager
- **File**: `orchestrator/docker/manager.go`
- **Role**: Spawns and manages agent containers dynamically

## Prerequisites

1. **Docker**: Must be installed and running
   - **macOS**: Download Docker Desktop from https://docker.com
   - **Linux**: `sudo apt-get install docker.io docker-compose` (Ubuntu/Debian)
   - **Windows**: Download Docker Desktop from https://docker.com
   
2. **Go**: Version 1.21 or later
   - Download from https://golang.org/dl/
   
3. **Python**: Version 3.10 or later (usually pre-installed on macOS/Linux)
   
4. **OpenAI API Key**: Get from https://platform.openai.com/api-keys
   
5. **protoc**: Protocol buffer compiler
   - **macOS**: `brew install protobuf`
   - **Linux**: `sudo apt-get install protobuf-compiler`
   - **Windows**: Download from https://protobuf.dev/downloads/

## Setup & Installation

1. **Clone and navigate to the project**:
   ```bash
   cd /path/to/agent_inc
   ```

2. **Set your OpenAI API key**:
   ```bash
   export OPENAI_API_KEY=your_api_key_here
   ```

3. **Build the system**:
   ```bash
   ./build.sh
   ```

4. **Run the system**:
   ```bash
   # Start the orchestrator
   cd orchestrator
   go run . &
   
   # Start the web UI (in a new terminal)
   cd ../ui
   npm install
   npm run dev
   ```

5. **Access the web interface**:
   - Open your browser to `http://localhost:5173`
   - Submit tasks and monitor phased execution
   - Approve or reject phase progressions

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
Phase 1 completes → User reviews results → Approves
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
- `DOCKER_HOST`: Override Docker connection (optional)

### Model Configuration
Modify in `agents/generic_agent/agent.py`:
- **Decision Model**: `gpt-4o` (for phased planning)
- **Execution Model**: `gpt-4-turbo` (for domain expert tasks)

### Phase Configuration
Modify in `orchestrator/main.go`:
- **Max Experts per Phase**: 10 (configurable)
- **Phase Timeout**: Adjustable per phase type
- **Approval Timeout**: User approval wait time

### UI Configuration
Modify in `ui/src/config.ts`:
- **API Base URL**: Orchestrator endpoint
- **Polling Interval**: Real-time update frequency
- **Theme**: UI appearance settings

### Container Configuration  
Modify in `orchestrator/docker/manager.go`:
- **Port Range**: Starting from 50060
- **Startup Delay**: 3 seconds for container readiness
- **Image Name**: `agentic-engineering-system_generic_agent`

## Monitoring & Debugging

### Web UI Dashboard
The React web interface provides:
- **Real-time Task Monitoring**: Live updates on task progress
- **Phase Visualization**: Clear phase breakdown and status
- **Expert Tracking**: Individual expert progress and results
- **Approval Workflow**: User-friendly phase approval interface

### Logs
The system provides comprehensive logging:
- **Phase Planning**: Lead agent decision-making
- **Expert Execution**: Individual domain expert progress
- **Container Management**: Docker lifecycle events
- **API Communication**: HTTP and gRPC interactions

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

2. **"protoc command not found"**
   - Install Protocol Buffers: `brew install protobuf`

3. **"OPENAI_API_KEY not set"**
   - Export the environment variable before running

4. **"Port already in use"**
   - System automatically handles port allocation
   - If issues persist, restart Docker

5. **"Web UI not loading"**
   - Ensure Node.js and npm are installed
   - Check if port 5173 is available
   - Run `npm install` in the ui directory

6. **"Phase not progressing"**
   - Check that all experts in the phase have completed
   - Review expert logs for any failures
   - Verify user approval is not required

7. **"Expert containers not spawning"**
   - Verify Docker daemon is running
   - Check Docker image exists: `agentic-engineering-system_generic_agent`
   - Review container logs for startup issues

## Performance Considerations

- **Parallel Phase Execution**: All experts in a phase run concurrently
- **Resource Management**: Containers cleaned up immediately after expert completion
- **Model Selection**: Balance between capability and cost/speed per expert
- **UI Responsiveness**: Real-time updates without overwhelming the interface
- **Phase Optimization**: Logical phase boundaries reduce unnecessary work

## Security Notes

- Never commit API keys to version control
- Container isolation provides security boundaries
- gRPC communication is local-only by default
- Consider using secrets management for production

## Future Enhancements

- **Persistent Phase Storage**: Database backend for phase history
- **Advanced UI Features**: Gantt charts, phase analytics, expert performance metrics
- **Multi-Cloud Support**: Deploy across cloud providers
- **Custom Expert Types**: Specialized agent types beyond generic personas
- **Integration Hooks**: Webhooks and API integrations for external tools
- **Phase Templates**: Pre-defined phase patterns for common project types
- **Collaborative Features**: Multi-user approval workflows

## License

This project demonstrates a production-ready implementation of a phased multi-agent AI system with user-controlled progression and approval workflows. It showcases modern software architecture principles with real-time monitoring and collaborative intelligence.
