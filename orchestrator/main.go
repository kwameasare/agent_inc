package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"agentic-engineering-system/docker"
	"agentic-engineering-system/tasks"
	"agentic-engineering-system/tasktree"
)

// Global state for the orchestrator
var (
	dockerManager *docker.Manager
	currentTasks  = make(map[string]*TaskExecution)
	tasksMutex    sync.RWMutex
)

type TaskExecution struct {
	ID       string                `json:"id"`
	Task     string                `json:"task"`
	Status   string                `json:"status"`
	Result   string                `json:"result,omitempty"`
	Error    string                `json:"error,omitempty"`
	Started  time.Time             `json:"started"`
	Tree     *tasktree.Tree        `json:"-"`
	RootNode *tasktree.Node        `json:"-"`
	Context  context.Context       `json:"-"`
	Cancel   context.CancelFunc    `json:"-"`
}

type TaskRequest struct {
	Task string `json:"task"`
}

type TaskResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func main() {
	// Check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Initialize Docker manager
	ctx := context.Background()
	var err error
	dockerManager, err = docker.NewManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create docker manager: %v", err)
	}
	defer dockerManager.CleanupAllAgents()

	// Setup HTTP routes
	http.HandleFunc("/api/task", enableCORS(handleTask))
	http.HandleFunc("/api/task/", enableCORS(handleTaskStatus))
	http.HandleFunc("/health", handleHealth)

	// Serve static files for the UI
	fs := http.FileServer(http.Dir("../ui/dist"))
	http.Handle("/", http.StripPrefix("/", fs))

	log.Println("🚀 Orchestrator starting...")
	log.Println("📡 HTTP API server listening on :8080")
	log.Println("🌐 UI available at http://localhost:8080")
	log.Println("📊 API endpoints:")
	log.Println("   POST /api/task - Submit new task")
	log.Println("   GET  /api/task/{id} - Get task status")
	log.Println("   GET  /health - Health check")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func enableCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

func handleTask(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var req TaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Task == "" {
			http.Error(w, "Task is required", http.StatusBadRequest)
			return
		}

		// Create new task execution
		taskID := fmt.Sprintf("task_%d", time.Now().Unix())
		ctx, cancel := context.WithCancel(context.Background())
		
		execution := &TaskExecution{
			ID:      taskID,
			Task:    req.Task,
			Status:  "pending",
			Started: time.Now(),
			Context: ctx,
			Cancel:  cancel,
		}

		tasksMutex.Lock()
		currentTasks[taskID] = execution
		tasksMutex.Unlock()

		// Start task execution in background
		go executeTask(execution)

		response := TaskResponse{
			ID:     taskID,
			Status: "pending",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case "GET":
		tasksMutex.RLock()
		tasks := make([]TaskExecution, 0, len(currentTasks))
		for _, task := range currentTasks {
			// Create a copy without context
			taskCopy := TaskExecution{
				ID:      task.ID,
				Task:    task.Task,
				Status:  task.Status,
				Result:  task.Result,
				Error:   task.Error,
				Started: task.Started,
			}
			tasks = append(tasks, taskCopy)
		}
		tasksMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract task ID from URL path
	taskID := r.URL.Path[len("/api/task/"):]
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	tasksMutex.RLock()
	execution, exists := currentTasks[taskID]
	tasksMutex.RUnlock()

	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Create response without context
	response := TaskExecution{
		ID:      execution.ID,
		Task:    execution.Task,
		Status:  execution.Status,
		Result:  execution.Result,
		Error:   execution.Error,
		Started: execution.Started,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"tasks":     len(currentTasks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func executeTask(execution *TaskExecution) {
	log.Printf("🚀 [%s] Starting task execution: %s", execution.ID, execution.Task[:min(100, len(execution.Task))])

	// Update status to running
	tasksMutex.Lock()
	execution.Status = "running"
	tasksMutex.Unlock()

	// Create task tree
	execution.Tree = tasktree.NewTree()
	
	// Default persona for general tasks
	persona := "You are an elite AI assistant with expertise across multiple domains including technology, business, science, and creative fields. You excel at analyzing complex problems, breaking them down into manageable components, and coordinating specialized approaches when needed."
	
	execution.RootNode = execution.Tree.AddNode("", persona, execution.Task)
	log.Printf("📋 [%s] Created root node: %s", execution.ID, execution.RootNode.ID)

	var wg sync.WaitGroup
	wg.Add(1)
	go executeNode(execution.Context, &wg, execution.Tree, dockerManager, execution.RootNode, execution.ID)
	wg.Wait()

	// Update final status
	tasksMutex.Lock()
	if execution.RootNode.Status == "completed" {
		execution.Status = "completed"
		execution.Result = execution.RootNode.Result
		log.Printf("✅ [%s] Task completed successfully", execution.ID)
	} else {
		execution.Status = "error"
		execution.Error = execution.RootNode.Result
		log.Printf("❌ [%s] Task failed: %s", execution.ID, execution.Error)
	}
	tasksMutex.Unlock()

	log.Printf("🏁 [%s] Task execution finished with status: %s", execution.ID, execution.Status)
}

// executeNode is the core recursive function of the orchestrator.
func executeNode(ctx context.Context, wg *sync.WaitGroup, tree *tasktree.Tree, dm *docker.Manager, node *tasktree.Node, taskID string) {
	defer wg.Done()
	log.Printf("🚀 [%s] Starting execution: %s", node.ID, node.Instructions[:min(100, len(node.Instructions))])

	tree.UpdateNodeStatus(node.ID, "running")

	// 1. Spawn a generic agent container for this task.
	log.Printf("🐳 [%s] Spawning agent container...", node.ID)
	agentContainer, err := dm.SpawnAgent(ctx)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to spawn agent container: %v", err)
		log.Printf("❌ [%s] %s", node.ID, errorMsg)
		tree.UpdateNodeStatus(node.ID, "failed")
		tree.UpdateNodeResult(node.ID, "ERROR: "+errorMsg)
		return
	}
	defer func() {
		log.Printf("🧹 [%s] Cleaning up agent container %s", node.ID, agentContainer.ID[:12])
		if err := dm.StopAgent(ctx, agentContainer.ID); err != nil {
			log.Printf("⚠️ [%s] Failed to cleanup container: %v", node.ID, err)
		}
	}()

	log.Printf("✅ [%s] Agent container spawned: %s at %s", node.ID, agentContainer.ID[:12], agentContainer.Address)

	// 2. Get context from completed sub-tasks if this is a synthesis
	contextData := tree.GetSubTaskResults(node.ID)
	if len(contextData) > 0 {
		log.Printf("📋 [%s] Using context from %d completed sub-tasks", node.ID, len(contextData))
	}

	// 3. Execute the task on the spawned agent via gRPC.
	log.Printf("📡 [%s] Sending task to agent...", node.ID)
	result, err := tasks.ExecuteTaskOnAgent(agentContainer.Address, node.ID, node.Persona, node.Instructions, contextData)
	if err != nil {
		errorMsg := fmt.Sprintf("gRPC communication failed: %v", err)
		log.Printf("❌ [%s] %s", node.ID, errorMsg)

		// Try to get container logs for debugging
		if logs, logErr := dm.GetContainerLogs(ctx, agentContainer.ID); logErr == nil {
			log.Printf("🔍 [%s] Container logs:\n%s", node.ID, logs)
		} else {
			log.Printf("⚠️ [%s] Could not retrieve container logs: %v", node.ID, logErr)
		}

		tree.UpdateNodeStatus(node.ID, "failed")
		tree.UpdateNodeResult(node.ID, "ERROR: "+errorMsg)
		return
	}

	// 4. Check for agent-reported errors
	if !result.Success {
		log.Printf("❌ [%s] Agent reported failure: %s", node.ID, result.ErrorMessage)
		tree.UpdateNodeStatus(node.ID, "failed")
		tree.UpdateNodeResult(node.ID, "AGENT ERROR: "+result.ErrorMessage)
		return
	}

	log.Printf("✅ [%s] Agent completed task successfully", node.ID)

	// 5. Check if the agent decided to delegate.
	if len(result.SubTasks) > 0 {
		log.Printf("🔀 [%s] Agent delegated into %d sub-tasks", node.ID, len(result.SubTasks))
		tree.UpdateNodeStatus(node.ID, "delegated")
		tree.SetRequiredSubTasks(node.ID, len(result.SubTasks))

		// Log sub-task details
		for i, subTask := range result.SubTasks {
			log.Printf("📝 [%s] Sub-task %d: %s -> %s", node.ID, i+1,
				subTask.RequestedPersona[:min(50, len(subTask.RequestedPersona))],
				subTask.TaskDetails[:min(100, len(subTask.TaskDetails))])
		}

		var subTaskWg sync.WaitGroup

		for i, subTaskReq := range result.SubTasks {
			// Create a new node in the tree for the sub-task.
			childNode := tree.AddNode(node.ID, subTaskReq.RequestedPersona, subTaskReq.TaskDetails)
			log.Printf("🌱 [%s] Created sub-task %d: %s", node.ID, i+1, childNode.ID)

			// Recursively call executeNode for the child.
			subTaskWg.Add(1)
			go func(child *tasktree.Node, taskNum int) {
				executeNode(ctx, &subTaskWg, tree, dm, child, taskID)
			}(childNode, i+1)

			// Add a longer delay between container starts to reduce resource contention
			if i < len(result.SubTasks)-1 { // Don't delay after the last one
				time.Sleep(2 * time.Second) // Increased from 500ms
			}
		}
		subTaskWg.Wait() // Wait for all children to finish.

		// Get failed and completed sub-tasks
		failedSubTasks := tree.GetFailedSubTasks(node.ID)
		completedSubTasks := tree.GetCompletedSubTasks(node.ID)

		log.Printf("📊 [%s] Sub-task summary: %d successful, %d failed",
			node.ID, len(completedSubTasks), len(failedSubTasks))

		// Check if any sub-tasks failed
		if len(failedSubTasks) > 0 {
			errorMsg := fmt.Sprintf("Sub-task failures: %v", failedSubTasks)
			log.Printf("❌ [%s] %s", node.ID, errorMsg)

			// Log detailed error information for each failed sub-task
			for _, failedID := range failedSubTasks {
				failedNode := tree.GetNode(failedID)
				if failedNode != nil {
					log.Printf("💥 [%s] Failed sub-task %s error: %s", node.ID, failedID, failedNode.Result)
				}
			}

			tree.UpdateNodeStatus(node.ID, "failed")
			tree.UpdateNodeResult(node.ID, "ERROR: "+errorMsg)
			return
		}

		// All sub-tasks are done. Now, we need to synthesize the results.
		log.Printf("🔄 [%s] All sub-tasks completed successfully. Starting synthesis...", node.ID)

		// Collate results from children.
		synthesisContext := tree.GetSubTaskResults(node.ID)

		if len(synthesisContext) == 0 {
			errorMsg := "No completed sub-tasks found for synthesis"
			log.Printf("❌ [%s] %s", node.ID, errorMsg)
			tree.UpdateNodeStatus(node.ID, "failed")
			tree.UpdateNodeResult(node.ID, "ERROR: "+errorMsg)
			return
		}

		log.Printf("📝 [%s] Synthesis context has %d sub-task results", node.ID, len(synthesisContext))

		synthesisInstructions := `All your sub-agents have completed their tasks. Their reports are provided in the context data. 

Your final task is to synthesize these reports into a single, cohesive document that fulfills your original objective. Create a comprehensive final report that:

1. Integrates all the sub-task results logically
2. Ensures consistency across all components
3. Identifies any gaps or inconsistencies 
4. Provides a final, actionable deliverable
5. Includes an executive summary

Original Task: ` + node.Instructions

		// Call the SAME agent again, but this time with the synthesis task.
		log.Printf("🔬 [%s] Sending synthesis task to agent...", node.ID)
		synthesisResult, err := tasks.ExecuteTaskOnAgent(agentContainer.Address, node.ID+"-synthesis", node.Persona, synthesisInstructions, synthesisContext)
		if err != nil {
			errorMsg := fmt.Sprintf("Synthesis gRPC failed: %v", err)
			log.Printf("❌ [%s] %s", node.ID, errorMsg)
			tree.UpdateNodeStatus(node.ID, "failed")
			tree.UpdateNodeResult(node.ID, "ERROR: "+errorMsg)
			return
		}

		if !synthesisResult.Success {
			errorMsg := fmt.Sprintf("Synthesis agent error: %s", synthesisResult.ErrorMessage)
			log.Printf("❌ [%s] %s", node.ID, errorMsg)
			tree.UpdateNodeStatus(node.ID, "failed")
			tree.UpdateNodeResult(node.ID, "ERROR: "+errorMsg)
			return
		}

		log.Printf("✅ [%s] Synthesis completed successfully", node.ID)
		tree.UpdateNodeResult(node.ID, synthesisResult.FinalContent)

	} else {
		// The agent executed the task directly.
		log.Printf("⚡ [%s] Agent executed task directly (no delegation)", node.ID)
		tree.UpdateNodeResult(node.ID, result.FinalContent)
	}

	tree.UpdateNodeStatus(node.ID, "completed")
	log.Printf("🎉 [%s] Task completed successfully!", node.ID)
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
