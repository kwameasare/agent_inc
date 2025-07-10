package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
	ID                   string             `json:"id"`
	Task                 string             `json:"task"`
	Status               string             `json:"status"`
	Result               string             `json:"result,omitempty"`
	Error                string             `json:"error,omitempty"`
	Started              time.Time          `json:"started"`
	Tree                 *tasktree.Tree     `json:"-"`
	RootNode             *tasktree.Node     `json:"-"`
	Context              context.Context    `json:"-"`
	Cancel               context.CancelFunc `json:"-"`
	Phases               []ProjectPhase     `json:"phases,omitempty"`
	CurrentPhase         int                `json:"currentPhase"`
	RequiresUserApproval bool               `json:"requiresUserApproval"`
}

type ProjectPhase struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Status       string            `json:"status"` // "pending", "approved", "running", "completed", "rejected"
	Experts      []DomainExpert    `json:"experts"`
	Results      map[string]string `json:"results,omitempty"`
	StartTime    *time.Time        `json:"startTime,omitempty"`
	EndTime      *time.Time        `json:"endTime,omitempty"`
	Approved     bool              `json:"approved"`
	UserFeedback string            `json:"userFeedback,omitempty"`
}

type DomainExpert struct {
	Role      string `json:"role"`
	Expertise string `json:"expertise"`
	Persona   string `json:"persona"`
	Task      string `json:"task"`
	Status    string `json:"status"` // "pending", "running", "completed", "failed"
	Result    string `json:"result,omitempty"`
}

type PhaseApprovalRequest struct {
	TaskID       string `json:"taskId"`
	PhaseID      string `json:"phaseId"`
	Approved     bool   `json:"approved"`
	UserFeedback string `json:"userFeedback,omitempty"`
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
	http.HandleFunc("/api/phases/approve", enableCORS(handlePhaseApproval))
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

func handlePhaseApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PhaseApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find the task execution
	tasksMutex.Lock()
	execution, exists := currentTasks[req.TaskID]
	if !exists {
		tasksMutex.Unlock()
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Find the phase by ID
	var phase *ProjectPhase
	for i := range execution.Phases {
		if execution.Phases[i].ID == req.PhaseID {
			phase = &execution.Phases[i]
			break
		}
	}

	if phase == nil {
		tasksMutex.Unlock()
		http.Error(w, "Phase not found", http.StatusNotFound)
		return
	}

	// Update phase approval status
	phase.Approved = req.Approved
	phase.UserFeedback = req.UserFeedback

	if req.Approved {
		phase.Status = "approved"
		log.Printf("✅ [%s] Phase %s approved by user", req.TaskID, req.PhaseID)

		// Continue with the next phase if there is one
		if execution.CurrentPhase < len(execution.Phases)-1 {
			execution.CurrentPhase++
			go startNextPhase(execution)
		} else {
			execution.Status = "completed"
			log.Printf("🎉 [%s] All phases completed", req.TaskID)
		}
	} else {
		phase.Status = "rejected"
		execution.Status = "failed"
		execution.Error = "Phase rejected by user: " + req.UserFeedback
		log.Printf("❌ [%s] Phase %s rejected by user: %s", req.TaskID, req.PhaseID, req.UserFeedback)
	}

	tasksMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"phase":   phase,
		"task":    execution,
	})
}

func startNextPhase(execution *TaskExecution) {
	tasksMutex.Lock()
	defer tasksMutex.Unlock()

	if execution.CurrentPhase >= len(execution.Phases) {
		log.Printf("⚠️ [%s] No more phases to execute", execution.ID)
		return
	}

	currentPhase := &execution.Phases[execution.CurrentPhase]
	currentPhase.Status = "running"
	currentPhase.StartTime = &[]time.Time{time.Now()}[0]

	log.Printf("🚀 [%s] Starting phase %d: %s", execution.ID, execution.CurrentPhase+1, currentPhase.Name)

	// Execute the domain experts in this phase
	for i := range currentPhase.Experts {
		go executeDomainExpert(execution.ID, currentPhase, &currentPhase.Experts[i])
	}
}

func executeDomainExpert(taskID string, phase *ProjectPhase, expert *DomainExpert) {
	log.Printf("👨‍💼 [%s] Starting domain expert: %s", taskID, expert.Role)

	expert.Status = "running"

	// Create agent container
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	agentContainer, err := dockerManager.SpawnAgent(ctx)
	if err != nil {
		expert.Status = "failed"
		expert.Result = fmt.Sprintf("Error spawning agent: %v", err)
		log.Printf("❌ [%s] Failed to spawn agent for domain expert %s: %v", taskID, expert.Role, err)
		return
	}

	// Cleanup agent when done
	defer func() {
		log.Printf("🧹 [%s] Cleaning up agent container for %s", taskID, expert.Role)
		if err := dockerManager.StopAgent(ctx, agentContainer.ID); err != nil {
			log.Printf("⚠️ Failed to cleanup agent container: %v", err)
		}
	}()

	log.Printf("✅ [%s] Agent container spawned for %s: %s at %s", taskID, expert.Role, agentContainer.ID[:12], agentContainer.Address)

	// Execute the expert's task with empty context data
	contextData := make(map[string]string)
	result, err := tasks.ExecuteTaskOnAgent(agentContainer.Address, expert.Role, expert.Persona, expert.Task, contextData)
	if err != nil {
		expert.Status = "failed"
		expert.Result = fmt.Sprintf("Error: %v", err)
		log.Printf("❌ [%s] Domain expert %s failed: %v", taskID, expert.Role, err)
		return
	}

	// Check for agent-reported errors
	if !result.Success {
		expert.Status = "failed"
		expert.Result = "AGENT ERROR: " + result.ErrorMessage
		log.Printf("❌ [%s] Domain expert %s reported failure: %s", taskID, expert.Role, result.ErrorMessage)
		return
	}

	expert.Status = "completed"
	expert.Result = result.FinalContent

	// Store result in phase results
	tasksMutex.Lock()
	if phase.Results == nil {
		phase.Results = make(map[string]string)
	}
	phase.Results[expert.Role] = result.FinalContent
	tasksMutex.Unlock()

	log.Printf("✅ [%s] Domain expert %s completed", taskID, expert.Role)

	// Check if all experts in this phase are done
	checkPhaseCompletion(taskID, phase)
}

func checkPhaseCompletion(taskID string, phase *ProjectPhase) {
	tasksMutex.Lock()
	defer tasksMutex.Unlock()

	allCompleted := true
	for _, expert := range phase.Experts {
		if expert.Status != "completed" && expert.Status != "failed" {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		phase.Status = "completed"
		phase.EndTime = &[]time.Time{time.Now()}[0]

		// Check if this phase requires user approval
		execution := currentTasks[taskID]
		if execution != nil && execution.RequiresUserApproval {
			phase.Status = "awaiting_approval"
			log.Printf("⏳ [%s] Phase %s completed, awaiting user approval", taskID, phase.ID)
		} else {
			// Auto-approve and continue
			phase.Approved = true
			phase.Status = "approved"
			if execution.CurrentPhase < len(execution.Phases)-1 {
				execution.CurrentPhase++
				go startNextPhase(execution)
			} else {
				execution.Status = "completed"
				log.Printf("🎉 [%s] All phases completed", taskID)
			}
		}
	}
}

func executeTask(execution *TaskExecution) {
	log.Printf("🚀 [%s] Starting task execution: %s", execution.ID, execution.Task[:min(100, len(execution.Task))])

	// Update status to running
	tasksMutex.Lock()
	execution.Status = "running"
	execution.RequiresUserApproval = true // Enable phased execution with user approval
	tasksMutex.Unlock()

	// First, use a lead agent to break down the task into phases with domain experts
	if len(execution.Phases) == 0 {
		log.Printf("📋 [%s] Breaking down task into phases with domain experts", execution.ID)

		// Create a lead agent to analyze and break down the task
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		agentContainer, err := dockerManager.SpawnAgent(ctx)
		if err != nil {
			tasksMutex.Lock()
			execution.Status = "failed"
			execution.Error = fmt.Sprintf("Failed to spawn lead agent: %v", err)
			tasksMutex.Unlock()
			log.Printf("❌ [%s] Failed to spawn lead agent: %v", execution.ID, err)
			return
		}

		defer func() {
			log.Printf("🧹 [%s] Cleaning up lead agent container", execution.ID)
			if err := dockerManager.StopAgent(ctx, agentContainer.ID); err != nil {
				log.Printf("⚠️ Failed to cleanup lead agent container: %v", err)
			}
		}()

		// Task breakdown prompt for the lead agent - focused on getting valid JSON
		breakdownTask := fmt.Sprintf(`You are a senior project manager. Break down this task into phases with domain experts.

TASK: %s

Your response MUST be ONLY this JSON format, no other text:

{
  "phases": [
    {
      "id": "phase-1",
      "name": "Requirements and Planning",
      "description": "Analyze requirements and create detailed plans",
      "experts": [
        {
          "role": "Business Analyst",
          "expertise": "Requirements analysis and stakeholder management",
          "persona": "Senior business analyst with 10+ years experience translating business needs into technical requirements",
          "task": "Analyze the task requirements and define detailed functional specifications"
        },
        {
          "role": "Technical Architect",
          "expertise": "System architecture and technology selection",
          "persona": "Expert system architect specializing in scalable, maintainable system design",
          "task": "Define the technical architecture and technology stack recommendations"
        },
        {
          "role": "UX Designer",
          "expertise": "User experience design and interface planning",
          "persona": "Lead UX designer with expertise in user-centered design and accessibility",
          "task": "Create user experience framework and interface design guidelines"
        }
      ]
    }
  ]
}

RULES:
- Maximum 10 experts in phase 1
- Each expert gets ONE specific task
- Experts do NOT delegate further
- Focus on planning/design before implementation
- Return ONLY the JSON, nothing else`, execution.Task)

		leadPersona := "You are a JSON response generator. You ONLY output valid JSON. You never include explanations, comments, or any text outside the JSON structure. You are expert at project planning and always return exactly the requested JSON format."

		contextData := make(map[string]string)
		result, err := tasks.ExecuteTaskOnAgent(agentContainer.Address, "lead-planner", leadPersona, breakdownTask, contextData)
		if err != nil {
			tasksMutex.Lock()
			execution.Status = "failed"
			execution.Error = fmt.Sprintf("Lead agent planning failed: %v", err)
			tasksMutex.Unlock()
			log.Printf("❌ [%s] Lead agent planning failed: %v", execution.ID, err)
			return
		}

		if !result.Success {
			tasksMutex.Lock()
			execution.Status = "failed"
			execution.Error = "Lead agent reported planning failure: " + result.ErrorMessage
			tasksMutex.Unlock()
			log.Printf("❌ [%s] Lead agent reported planning failure: %s", execution.ID, result.ErrorMessage)
			return
		}

		// Parse the JSON response to extract phases
		var planResponse struct {
			Phases []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Experts     []struct {
					Role      string `json:"role"`
					Expertise string `json:"expertise"`
					Persona   string `json:"persona"`
					Task      string `json:"task"`
				} `json:"experts"`
			} `json:"phases"`
		}

		// Try to extract JSON from the response (in case agent included extra text)
		content := strings.TrimSpace(result.FinalContent)
		log.Printf("🔍 [%s] Raw planning response: '%s'", execution.ID, content[:min(300, len(content))])

		// Clean the content - remove any markdown formatting
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)

		// Look for JSON object boundaries
		startIdx := -1
		endIdx := -1
		braceCount := 0

		for i, char := range content {
			if char == '{' {
				if startIdx == -1 {
					startIdx = i
				}
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 && startIdx != -1 {
					endIdx = i + 1
					break
				}
			}
		}

		var jsonContent string
		if startIdx != -1 && endIdx != -1 {
			jsonContent = content[startIdx:endIdx]
		} else if len(content) > 0 && content[0] == '{' {
			// If starts with { but no complete object found, try the whole content
			jsonContent = content
		} else {
			log.Printf("⚠️ [%s] No valid JSON structure found in response, falling back to tree execution", execution.ID)
			log.Printf("🔍 [%s] Full response content: '%s'", execution.ID, content)
			executeTaskWithTree(execution)
			return
		}

		log.Printf("🔍 [%s] Attempting to parse JSON: %s", execution.ID, jsonContent[:min(200, len(jsonContent))])

		if err := json.Unmarshal([]byte(jsonContent), &planResponse); err != nil {
			log.Printf("⚠️ [%s] Failed to parse JSON, falling back to tree execution: %v", execution.ID, err)
			log.Printf("🔍 [%s] JSON content was: %s", execution.ID, jsonContent)
			executeTaskWithTree(execution)
			return
		}

		// Validate the response structure
		if len(planResponse.Phases) == 0 {
			log.Printf("⚠️ [%s] No phases in response, falling back to tree execution", execution.ID)
			executeTaskWithTree(execution)
			return
		}

		// Convert to our phase structure
		tasksMutex.Lock()
		execution.Phases = make([]ProjectPhase, len(planResponse.Phases))
		for i, phase := range planResponse.Phases {
			execution.Phases[i] = ProjectPhase{
				ID:          phase.ID,
				Name:        phase.Name,
				Description: phase.Description,
				Status:      "pending",
				Results:     make(map[string]string),
			}

			// Convert experts
			execution.Phases[i].Experts = make([]DomainExpert, len(phase.Experts))
			for j, expert := range phase.Experts {
				execution.Phases[i].Experts[j] = DomainExpert{
					Role:      expert.Role,
					Expertise: expert.Expertise,
					Persona:   expert.Persona,
					Task:      expert.Task,
					Status:    "pending",
				}
			}
		}
		execution.CurrentPhase = 0
		tasksMutex.Unlock()

		log.Printf("✅ [%s] Task broken down into %d phases", execution.ID, len(execution.Phases))
		for i, phase := range execution.Phases {
			log.Printf("📋 [%s] Phase %d: %s (%d experts)", execution.ID, i+1, phase.Name, len(phase.Experts))
		}
	}

	// Start execution of the first phase
	if len(execution.Phases) > 0 {
		log.Printf("🎬 [%s] Starting phased execution", execution.ID)
		startNextPhase(execution)
	} else {
		log.Printf("⚠️ [%s] No phases created, falling back to tree execution", execution.ID)
		executeTaskWithTree(execution)
	}
}

func executeTaskWithTree(execution *TaskExecution) {
	log.Printf("🌳 [%s] Using tree-based execution", execution.ID)

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
		log.Printf("❌ [%s] Task failed: %s", execution.ID, execution.RootNode.Result)
	}
	tasksMutex.Unlock()
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
