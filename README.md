# Agentic Engineering System

A dynamic, hierarchical multi-agent AI system that functions as an elite autonomous engineering team spanning software, hardware, product, and safety departments. The system optimizes for scalability, collaboration, traceability, and review loops using efficient serverless/cloud-native infrastructure.

## Architecture

The system implements a **hierarchical multi-agent architecture** with these core principles:

1. **Agents as a Commodity**: There is only one type of "worker" agent service. Its specialization (persona) is defined at runtime by the instruction prompt it receives.

2. **Recursion as a Core Mechanic**: Any agent can become a manager. If a task is too complex, the agent's job is not to solve it directly, but to break it down and request that the Orchestrator spawn a team of sub-agents to handle the pieces.

This creates a **tree of tasks** that grows dynamically until every leaf-node task is simple enough for a single agent to execute.

## Components

### 1. Generic Agent (Python)
- **File**: `agents/generic_agent/agent.py`
- **Role**: Universal worker that can be anything you tell it to be
- **Technology**: Python + gRPC + LiteLLM (supports OpenAI, Anthropic, etc.)
- **Key Feature**: Makes execute/delegate decisions autonomously using LLM reasoning

### 2. Orchestrator (Go)
- **File**: `orchestrator/main.go`
- **Role**: 
  - Factory for agents (creates new instances on demand)
  - Tree Manager (maintains state of entire task hierarchy) 
  - Router (dispatches tasks and routes results)
  - Docker Manager (controls agent container lifecycle)
- **Technology**: Go + Docker SDK + gRPC

### 3. Task Tree Manager
- **File**: `orchestrator/tasktree/manager.go`
- **Role**: Manages the hierarchical task structure with thread-safe operations

### 4. Docker Manager
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

4. **Run the orchestrator**:
   ```bash
   cd orchestrator
   go run .
   ```

## How It Works

### 1. Initial Task Submission
The system starts with a high-level task given to a Lead Agent (Project Manager persona).

### 2. Dynamic Task Decomposition
- Each agent uses an LLM to decide: "Can I solve this alone, or is it too complex?"
- **If simple**: Executes the task directly and returns results
- **If complex**: Breaks it down into sub-tasks with specific personas and delegates

### 3. Recursive Agent Spawning
- For each sub-task, the Orchestrator spawns a new agent container
- Each container runs the same generic agent code but with different personas
- Creates a tree structure: Lead → Department Heads → Specialists → Sub-specialists

### 4. Result Synthesis
- When all sub-tasks complete, parent agents synthesize the results
- Results flow back up the hierarchy
- Final output is a comprehensive, integrated deliverable

### 5. Automatic Cleanup
- Containers are automatically stopped and removed after use
- Full traceability maintained throughout the process

## Example Workflow

```
Initial Task: "Design a scalable real-time chat application"
    ↓
Lead Agent decides to delegate:
    ├── Software Architecture Agent
    ├── Database Design Agent  
    ├── Frontend Design Agent
    ├── Security Analysis Agent
    └── Deployment Strategy Agent
        ↓
Each may further delegate:
Software Architecture Agent → 
    ├── Backend API Specialist
    ├── Real-time Communication Specialist
    └── Microservices Design Specialist
        ↓
Results synthesized back up the tree:
    ← Specialists provide detailed designs
    ← Department agents integrate their domains
    ← Lead agent creates final comprehensive plan
```

## Key Features

### 🔄 **Dynamic Scaling**
- Agents spawn only when needed
- Tree can grow to any depth based on task complexity
- Automatic resource cleanup

### 🎯 **Specialization**
- Each agent focuses on its domain expertise
- Personas defined at runtime for maximum flexibility
- No hard-coded roles or limitations

### 🔍 **Full Traceability**
- Every decision and delegation logged
- Complete audit trail of reasoning
- Easy debugging and process improvement

### 🤝 **Collaborative Intelligence**
- Agents build on each other's work
- Cross-domain consistency checking
- Collective problem-solving approach

### 🛡️ **Fault Tolerance**
- Failed agents don't crash the system
- Automatic retries and error handling
- Graceful degradation

## Configuration

### Environment Variables
- `OPENAI_API_KEY`: Required for LLM access
- `DOCKER_HOST`: Override Docker connection (optional)

### Model Configuration
Modify in `agents/generic_agent/agent.py`:
- **Decision Model**: `gpt-4o` (for complex reasoning)
- **Execution Model**: `gpt-4-turbo` (for task execution)

### Container Configuration  
Modify in `orchestrator/docker/manager.go`:
- **Port Range**: Starting from 50060
- **Startup Delay**: 3 seconds for container readiness
- **Image Name**: `agentic-engineering-system_generic_agent`

## Monitoring & Debugging

### Logs
The system provides comprehensive logging:
- Container spawning/stopping
- Task delegation decisions  
- gRPC communication
- Result synthesis

### Task Tree Inspection
The task tree maintains full state and can be inspected for debugging:
- Task hierarchy visualization
- Status tracking (pending, running, delegated, completed, failed)
- Result aggregation

## Extending the System

### Adding New Personas
Simply modify the initial prompt or add specialized personas in the main.go file.

### Custom Task Types
Add new task types by modifying the agent decision logic in `agent.py`.

### Additional Infrastructure
Extend the Docker manager to support other container orchestration platforms.

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

5. **Agent containers not stopping**
   - System includes automatic cleanup
   - Manual cleanup: `docker stop $(docker ps -q --filter ancestor=agentic-engineering-system_generic_agent)`

6. **gRPC connection timeouts with many sub-tasks**
   - This is expected behavior when spawning many containers simultaneously
   - The system will retry and eventually succeed with the synthesis
   - For better performance, use a simpler initial task or increase timeout values
   - Sub-task failures don't prevent the root synthesis from completing

## Performance Considerations

- **Parallel Execution**: Sub-tasks run concurrently when possible
- **Resource Management**: Containers cleaned up immediately after use
- **Model Selection**: Balance between capability and cost/speed
- **Caching**: Results cached in task tree for potential reuse

## Security Notes

- Never commit API keys to version control
- Container isolation provides security boundaries
- gRPC communication is local-only by default
- Consider using secrets management for production

## Future Enhancements

- **Persistent Task Storage**: Database backend for task trees
- **Web UI**: Visual task tree and progress monitoring
- **Multi-Cloud Support**: Deploy across cloud providers
- **Custom Agents**: Specialized agent types beyond generic
- **Real-time Collaboration**: Human-in-the-loop capabilities

## License

This project demonstrates the principles outlined in the task description and steps documents. It showcases a production-ready implementation of a hierarchical multi-agent AI system.
