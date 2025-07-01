# Quick Start Guide

Get the Agentic Engineering System running in 5 minutes.

## 1. Prerequisites Check

Ensure you have these installed:
- **Docker**: `docker --version`
- **Go**: `go version` (1.21+)
- **Python**: `python3 --version` (3.10+)

## 2. Get API Key

1. Visit https://platform.openai.com/api-keys
2. Create a new API key
3. Copy the key (starts with `sk-`)

## 3. Set Environment Variable

```bash
export OPENAI_API_KEY=sk-your-actual-api-key-here
```

## 4. Build and Run

```bash
# Make build script executable (if needed)
chmod +x build.sh

# Build the system
./build.sh

# Run the orchestrator
cd orchestrator
go run .
```

## 5. Watch the Magic

You'll see logs like:
```
Starting workflow with root task: task-1698765432123
Spawned agent container abc123 on port 50060
Task task-1698765432123: Decision is to delegate.
Task task-1698765432123 delegated into 5 sub-tasks.
Spawned agent container def456 on port 50061
...
```

The system will:
1. 🤖 Spawn the first agent to analyze the initial task
2. 🔀 Agent decides to delegate into specialized sub-tasks
3. 🚀 Multiple agents spawn in parallel for each sub-task
4. 📊 Sub-agents may further delegate if tasks are complex
5. 🔄 Results synthesize back up the hierarchy
6. 📋 Final comprehensive report is generated
7. 🧹 All containers are automatically cleaned up

## Example Output

The system will produce a comprehensive analysis like:

```
=== Final Result ===
# Comprehensive Chat Application System Design

## Executive Summary
This document presents a complete architecture for a scalable real-time chat application supporting 100,000+ concurrent users...

## 1. Database Architecture
[Detailed database schema and scaling strategy]

## 2. Backend API Design  
[RESTful API specifications and microservices architecture]

## 3. Frontend Architecture
[React/Vue component structure and state management]

## 4. Security Framework
[Authentication, authorization, and data protection]

## 5. Deployment Strategy
[Kubernetes, monitoring, and CI/CD pipeline]
...
```

## Troubleshooting

**"Docker not found"**: Install Docker Desktop from https://docker.com

**"Permission denied"**: Run `chmod +x build.sh`

**"API key invalid"**: Check your OpenAI API key is correct

**"Port already in use"**: Restart Docker or wait for containers to clean up

## What's Happening Under the Hood

1. **Initial Task**: System starts with a complex engineering task
2. **Decision Making**: Lead agent uses GPT-4o to decide if task is simple or complex
3. **Delegation**: If complex, breaks down into specialist sub-tasks
4. **Recursive Spawning**: Each sub-task gets its own container and agent
5. **Parallel Execution**: Independent tasks run simultaneously
6. **Synthesis**: Results combine back up the hierarchy
7. **Final Output**: Comprehensive, integrated deliverable

## Customization

Want to try a different task? Edit the `initialTask` variable in `orchestrator/main.go`:

```go
initialTask := "Design a blockchain-based supply chain system"
// or
initialTask := "Create a machine learning pipeline for fraud detection"
// or  
initialTask := "Develop an IoT sensor network for smart cities"
```

The system will adapt automatically to any complex engineering challenge!

## Next Steps

- Read the full [README.md](README.md) for complete documentation
- Check [IMPLEMENTATION.md](IMPLEMENTATION.md) for technical details
- Explore the code to understand the architecture
- Try modifying agent personas or task examples
- Consider contributing improvements or extensions

---

**Need Help?** Check the troubleshooting section in README.md or review the logs for specific error messages.
