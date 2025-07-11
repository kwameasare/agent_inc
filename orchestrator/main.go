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
	"agentic-engineering-system/websocket"

	"go.etcd.io/bbolt"
)

// Global state for the orchestrator
var (
	dockerManager *docker.Manager
	currentTasks  = make(map[string]*TaskExecution)
	tasksMutex    sync.RWMutex
	db            *bbolt.DB
	wsHub         *websocket.Hub
	sseClients    = make(map[string]chan string)
	sseMutex      sync.RWMutex
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
	CreatedAt            time.Time          `json:"createdAt"`
	UpdatedAt            time.Time          `json:"updatedAt"`
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

// BoltDB Functions
func saveTaskState(execution *TaskExecution) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		encoded, err := json.Marshal(execution)
		if err != nil {
			return fmt.Errorf("failed to serialize task %s: %w", execution.ID, err)
		}
		return b.Put([]byte(execution.ID), encoded)
	})
}

func loadTaskState(taskID string) (*TaskExecution, error) {
	var execution TaskExecution
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		data := b.Get([]byte(taskID))
		if data == nil {
			return fmt.Errorf("task %s not found in DB", taskID)
		}
		if err := json.Unmarshal(data, &execution); err != nil {
			return fmt.Errorf("failed to deserialize task %s: %w", taskID, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

func loadAllTasks() ([]*TaskExecution, error) {
	var tasks []*TaskExecution
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		return b.ForEach(func(k, v []byte) error {
			var execution TaskExecution
			if err := json.Unmarshal(v, &execution); err != nil {
				log.Printf("Warning: Failed to deserialize task %s: %v", string(k), err)
				return nil // Continue with other tasks
			}
			tasks = append(tasks, &execution)
			return nil
		})
	})
	return tasks, err
}

// Broadcast updates to WebSocket clients (similar to SSE concept)
func broadcastUpdate(taskID string) {
	execution, err := loadTaskState(taskID)
	if err == nil {
		if wsHub != nil {
			wsHub.BroadcastMessage("task_updated", execution, taskID, "")
		}

		// Also broadcast to SSE clients
		sseMutex.RLock()
		clientChan, ok := sseClients[taskID]
		sseMutex.RUnlock()

		if ok {
			jsonData, _ := json.Marshal(execution)
			select {
			case clientChan <- string(jsonData):
			default:
				// Channel is full or closed, skip
			}
		}
	}
}

// SSE Handler
func handleTaskEvents(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	messageChan := make(chan string, 10) // Buffered channel

	sseMutex.Lock()
	sseClients[taskID] = messageChan
	sseMutex.Unlock()

	defer func() {
		sseMutex.Lock()
		delete(sseClients, taskID)
		sseMutex.Unlock()
		close(messageChan)
	}()

	// Send initial state
	if execution, err := loadTaskState(taskID); err == nil {
		if jsonData, err := json.Marshal(execution); err == nil {
			fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}

	for {
		select {
		case msg := <-messageChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func main() {
	// Check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Get port from environment variable, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize BoltDB
	var err error
	db, err = bbolt.Open("orchestrator.db", 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("FATAL: Could not open database: %v", err)
	}
	defer db.Close()

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("tasks"))
		return err
	})
	if err != nil {
		log.Fatalf("FATAL: Could not create tasks bucket: %v", err)
	}

	log.Printf("✅ BoltDB initialized successfully")

	// Load existing tasks from database
	loadTasksFromDB()

	// Initialize WebSocket hub
	wsHub = websocket.NewHub()
	go wsHub.Run()

	// Initialize Docker manager
	ctx := context.Background()
	dockerManager, err = docker.NewManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create docker manager: %v", err)
	}
	defer dockerManager.CleanupAllAgents()

	// Setup HTTP routes
	http.HandleFunc("/api/task", enableCORS(handleTask))
	http.HandleFunc("/api/task/", enableCORS(handleTaskStatus))
	http.HandleFunc("/api/task/{taskId}/phase/{phaseId}", enableCORS(handlePhaseDetails))
	http.HandleFunc("/api/task/{taskId}/events", enableCORS(handleTaskEvents))
	http.HandleFunc("/api/phases/approve", enableCORS(handlePhaseApproval))
	http.HandleFunc("/api/phase/", enableCORS(handlePhaseResults))
	http.HandleFunc("/ws", wsHub.HandleWebSocket)
	http.HandleFunc("/health", handleHealth)

	// Serve static files for the UI
	fs := http.FileServer(http.Dir("./ui/dist"))
	http.Handle("/", http.StripPrefix("/", fs))

	log.Println("🚀 Orchestrator starting...")
	log.Printf("📡 HTTP API server listening on :%s", port)
	log.Printf("🌐 UI available at http://localhost:%s", port)
	log.Println("📊 API endpoints:")
	log.Println("   POST /api/task - Submit new task")
	log.Println("   GET  /api/task/{id} - Get task status")
	log.Println("   GET  /api/phase/{taskId}/{phaseId} - Get phase results")
	log.Println("   POST /api/phases/approve - Approve/reject phase")
	log.Println("   WS   /ws - WebSocket for real-time updates")
	log.Println("   GET  /health - Health check")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func loadTasksFromDB() {
	tasks, err := loadAllTasks()
	if err != nil {
		log.Printf("Warning: Failed to load tasks from database: %v", err)
		return
	}

	tasksMutex.Lock()
	defer tasksMutex.Unlock()

	for _, task := range tasks {
		currentTasks[task.ID] = task
	}

	log.Printf("✅ Loaded %d tasks from database", len(tasks))
}

func saveTaskToDB(task *TaskExecution) {
	if err := saveTaskState(task); err != nil {
		log.Printf("Warning: Failed to save task to database: %v", err)
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

		now := time.Now()
		execution := &TaskExecution{
			ID:                   taskID,
			Task:                 req.Task,
			Status:               "pending",
			Started:              now,
			Context:              ctx,
			Cancel:               cancel,
			RequiresUserApproval: true,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		tasksMutex.Lock()
		currentTasks[taskID] = execution
		tasksMutex.Unlock()

		// Save the initial state to the database
		if err := saveTaskState(execution); err != nil {
			log.Printf("ERROR: Failed to save initial state for task %s: %v", taskID, err)
			http.Error(w, "Failed to persist task", http.StatusInternalServerError)
			return
		}

		// Broadcast task creation via WebSocket
		if wsHub != nil {
			wsHub.BroadcastMessage("task_created", execution, taskID, "")
		}

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
				ID:                   task.ID,
				Task:                 task.Task,
				Status:               task.Status,
				Result:               task.Result,
				Error:                task.Error,
				Started:              task.Started,
				Phases:               task.Phases,
				CurrentPhase:         task.CurrentPhase,
				RequiresUserApproval: task.RequiresUserApproval,
				CreatedAt:            task.CreatedAt,
				UpdatedAt:            task.UpdatedAt,
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
		// Try to load from database if not in memory
		execution, err := loadTaskState(taskID)
		if err != nil {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}

		// Cache in memory
		tasksMutex.Lock()
		currentTasks[taskID] = execution
		tasksMutex.Unlock()
	}

	// Create response without context
	response := TaskExecution{
		ID:                   execution.ID,
		Task:                 execution.Task,
		Status:               execution.Status,
		Result:               execution.Result,
		Error:                execution.Error,
		Started:              execution.Started,
		Phases:               execution.Phases,
		CurrentPhase:         execution.CurrentPhase,
		RequiresUserApproval: execution.RequiresUserApproval,
		CreatedAt:            execution.CreatedAt,
		UpdatedAt:            execution.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handlePhaseDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskID := r.PathValue("taskId")
	phaseID := r.PathValue("phaseId")

	execution, err := loadTaskState(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	var phase *ProjectPhase
	for i := range execution.Phases {
		if execution.Phases[i].ID == phaseID {
			phase = &execution.Phases[i]
			break
		}
	}

	if phase == nil {
		http.Error(w, "Phase not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(phase)
}

func handlePhaseResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract task ID and phase ID from URL path: /api/phase/{taskId}/{phaseId}
	path := r.URL.Path[len("/api/phase/"):]
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		http.Error(w, "Invalid URL format. Use /api/phase/{taskId}/{phaseId}", http.StatusBadRequest)
		return
	}

	taskID := parts[0]
	phaseID := parts[1]

	tasksMutex.RLock()
	execution, exists := currentTasks[taskID]
	tasksMutex.RUnlock()

	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Find the phase
	var phase *ProjectPhase
	for i := range execution.Phases {
		if execution.Phases[i].ID == phaseID {
			phase = &execution.Phases[i]
			break
		}
	}

	if phase == nil {
		http.Error(w, "Phase not found", http.StatusNotFound)
		return
	}

	// Get detailed results for each expert in the phase
	detailedResults := make(map[string]interface{})
	for _, expert := range phase.Experts {
		detailedResults[expert.Role] = map[string]interface{}{
			"expertise": expert.Expertise,
			"task":      expert.Task,
			"status":    expert.Status,
			"result":    expert.Result,
		}
	}

	response := map[string]interface{}{
		"phase":           phase,
		"detailedResults": detailedResults,
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

func updateTaskStatus(execution *TaskExecution, status string, result, error string) {
	tasksMutex.Lock()
	execution.Status = status
	execution.UpdatedAt = time.Now()
	if result != "" {
		execution.Result = result
	}
	if error != "" {
		execution.Error = error
	}
	tasksMutex.Unlock()

	// Save to database
	saveTaskToDB(execution)

	// Broadcast status update via WebSocket
	if wsHub != nil {
		wsHub.BroadcastMessage("task_status_updated", map[string]interface{}{
			"id":     execution.ID,
			"status": status,
			"result": result,
			"error":  error,
		}, execution.ID, "")
	}
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
	execution.UpdatedAt = time.Now()

	if req.Approved {
		phase.Status = "approved"
		log.Printf("✅ [%s] Phase %s approved by user", req.TaskID, req.PhaseID)

		// Broadcast phase approval via WebSocket
		if wsHub != nil {
			wsHub.BroadcastMessage("phase_approved", map[string]interface{}{
				"taskId":  req.TaskID,
				"phaseId": req.PhaseID,
				"phase":   phase,
			}, req.TaskID, req.PhaseID)
		}

		// Continue with the next phase if there is one
		if execution.CurrentPhase < len(execution.Phases)-1 {
			execution.CurrentPhase++
			go startNextPhase(execution)
		} else {
			execution.Status = "completed"
			log.Printf("🎉 [%s] All phases completed", req.TaskID)

			// Broadcast task completion
			if wsHub != nil {
				wsHub.BroadcastMessage("task_completed", execution, req.TaskID, "")
			}
		}
	} else {
		phase.Status = "rejected"
		execution.Status = "failed"
		execution.Error = "Phase rejected by user: " + req.UserFeedback
		log.Printf("❌ [%s] Phase %s rejected by user: %s", req.TaskID, req.PhaseID, req.UserFeedback)

		// Broadcast phase rejection via WebSocket
		if wsHub != nil {
			wsHub.BroadcastMessage("phase_rejected", map[string]interface{}{
				"taskId":  req.TaskID,
				"phaseId": req.PhaseID,
				"phase":   phase,
				"reason":  req.UserFeedback,
			}, req.TaskID, req.PhaseID)
		}
	}

	tasksMutex.Unlock()

	// Save to database
	saveTaskToDB(execution)

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
	execution.UpdatedAt = time.Now()

	log.Printf("🚀 [%s] Starting phase %d: %s", execution.ID, execution.CurrentPhase+1, currentPhase.Name)

	// Save to database
	saveTaskToDB(execution)

	// Broadcast phase start via WebSocket
	if wsHub != nil {
		wsHub.BroadcastMessage("phase_started", map[string]interface{}{
			"taskId": execution.ID,
			"phase":  currentPhase,
		}, execution.ID, currentPhase.ID)
	}

	// Execute the domain experts in this phase
	for i := range currentPhase.Experts {
		go executeDomainExpert(execution.ID, currentPhase, &currentPhase.Experts[i])
	}
}

func executeDomainExpert(taskID string, phase *ProjectPhase, expert *DomainExpert) {
	log.Printf("👨‍💼 [%s] Starting domain expert: %s", taskID, expert.Role)

	expert.Status = "running"

	// Broadcast expert start via WebSocket
	if wsHub != nil {
		wsHub.BroadcastMessage("expert_started", map[string]interface{}{
			"taskId":  taskID,
			"phaseId": phase.ID,
			"expert":  expert,
		}, taskID, phase.ID)
	}

	// Create agent container
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	agentContainer, err := dockerManager.SpawnAgent(ctx)
	if err != nil {
		expert.Status = "failed"
		expert.Result = fmt.Sprintf("Error spawning agent: %v", err)
		log.Printf("❌ [%s] Failed to spawn agent for domain expert %s: %v", taskID, expert.Role, err)

		// Broadcast expert failure
		if wsHub != nil {
			wsHub.BroadcastMessage("expert_failed", map[string]interface{}{
				"taskId":  taskID,
				"phaseId": phase.ID,
				"expert":  expert,
				"error":   err.Error(),
			}, taskID, phase.ID)
		}
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

	// For Phase 1, delegation is not allowed. For others, it is.
	// This is a simple check; a more robust system might have this as a property of the phase itself.
	isPhaseOne := phase.ID == "phase_1_planning" || strings.HasPrefix(phase.ID, "phase-1")
	canDelegate := !isPhaseOne

	result, err := tasks.ExecuteTaskOnAgent(agentContainer.Address, expert.Role, expert.Persona, expert.Task, contextData, canDelegate)
	if err != nil {
		expert.Status = "failed"
		expert.Result = fmt.Sprintf("Error: %v", err)
		log.Printf("❌ [%s] Domain expert %s failed: %v", taskID, expert.Role, err)

		// Broadcast expert failure
		if wsHub != nil {
			wsHub.BroadcastMessage("expert_failed", map[string]interface{}{
				"taskId":  taskID,
				"phaseId": phase.ID,
				"expert":  expert,
				"error":   err.Error(),
			}, taskID, phase.ID)
		}
		return
	}

	// Check for agent-reported errors
	if !result.Success {
		expert.Status = "failed"
		expert.Result = "AGENT ERROR: " + result.ErrorMessage
		log.Printf("❌ [%s] Domain expert %s reported failure: %s", taskID, expert.Role, result.ErrorMessage)

		// Broadcast expert failure
		if wsHub != nil {
			wsHub.BroadcastMessage("expert_failed", map[string]interface{}{
				"taskId":  taskID,
				"phaseId": phase.ID,
				"expert":  expert,
				"error":   result.ErrorMessage,
			}, taskID, phase.ID)
		}
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

	// Broadcast expert completion via WebSocket
	if wsHub != nil {
		wsHub.BroadcastMessage("expert_completed", map[string]interface{}{
			"taskId":  taskID,
			"phaseId": phase.ID,
			"expert":  expert,
		}, taskID, phase.ID)
	}

	// Check if all experts in this phase are done
	checkPhaseCompletion(taskID, phase)
}

func checkPhaseCompletion(taskID string, phase *ProjectPhase) {
	// Check if all experts are completed
	allCompleted := true
	for _, expert := range phase.Experts {
		if expert.Status != "completed" && expert.Status != "failed" {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		tasksMutex.Lock()
		defer tasksMutex.Unlock()

		// Important: check the specific execution object
		execution, exists := currentTasks[taskID]
		if !exists {
			return
		}

		phase.Status = "completed"
		phase.EndTime = &[]time.Time{time.Now()}[0]
		execution.UpdatedAt = time.Now()

		// Save to database
		if err := saveTaskState(execution); err != nil {
			log.Printf("ERROR: Failed to save phase completion for %s: %v", taskID, err)
		}

		// Broadcast phase completion via WebSocket
		if wsHub != nil {
			wsHub.BroadcastMessage("phase_completed", map[string]interface{}{
				"taskId": taskID,
				"phase":  phase,
			}, taskID, phase.ID)
		}

		// Broadcast update
		broadcastUpdate(taskID)

		// This is the key logic for pausing
		if execution.RequiresUserApproval {
			phase.Status = "awaiting_approval"
			log.Printf("⏳ [%s] Phase '%s' completed. Awaiting user approval.", taskID, phase.Name)

			// Broadcast awaiting approval via WebSocket
			if wsHub != nil {
				wsHub.BroadcastMessage("phase_awaiting_approval", map[string]interface{}{
					"taskId": taskID,
					"phase":  phase,
				}, taskID, phase.ID)
			}
		} else {
			// Auto-approve if user approval is not required for this task
			phase.Approved = true
			phase.Status = "approved"
			log.Printf("✅ [%s] Phase '%s' auto-approved.", taskID, phase.Name)
			if execution.CurrentPhase < len(execution.Phases)-1 {
				execution.CurrentPhase++
				go startNextPhase(execution)
			} else {
				execution.Status = "completed"
				log.Printf("🎉 [%s] All phases completed.", taskID)

				// Broadcast task completion
				if wsHub != nil {
					wsHub.BroadcastMessage("task_completed", execution, taskID, "")
				}
			}
		}
	}
}

func executeTask(execution *TaskExecution) {
	log.Printf("🚀 [%s] Starting task execution: %s", execution.ID, execution.Task)

	tasksMutex.Lock()
	execution.Status = "planning"
	tasksMutex.Unlock()
	if err := saveTaskState(execution); err != nil {
		log.Printf("ERROR: Failed to save planning status for %s: %v", execution.ID, err)
	}

	updateTaskStatus(execution, "planning", "", "")

	// Step 1: Use the Lead Agent to generate a phased plan.
	err := generatePhasedPlan(execution)
	if err != nil {
		updateTaskStatus(execution, "failed", "", err.Error())
		log.Printf("❌ [%s] Failed to generate phased plan: %v", execution.ID, err)
		return
	}

	// Step 2: If a plan was generated successfully, start the first phase.
	if len(execution.Phases) > 0 {
		log.Printf("🎬 [%s] Phased plan generated. Starting first phase.", execution.ID)

		// Broadcast plan generation completion
		if wsHub != nil {
			wsHub.BroadcastMessage("plan_generated", map[string]interface{}{
				"taskId": execution.ID,
				"phases": execution.Phases,
			}, execution.ID, "")
		}

		startNextPhase(execution)
	} else {
		// Fallback for simple tasks that don't need phases.
		log.Printf("🌳 [%s] No phases generated, executing as a single task.", execution.ID)
		executeTaskWithTree(execution) // Keep the original logic as a fallback
	}
}

// MODIFICATION 2: Create a new function to generate the phased plan.
func generatePhasedPlan(execution *TaskExecution) error {
	log.Printf("📋 [%s] Generating phased project plan...", execution.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	agentContainer, err := dockerManager.SpawnAgent(ctx)
	if err != nil {
		return fmt.Errorf("failed to spawn lead agent: %v", err)
	}
	defer dockerManager.CleanupAllAgents()

	// This is the updated prompt for the Lead Agent to create phases.
	planningPrompt := fmt.Sprintf(`
You are a world-class AI Project Manager. Your job is to break down a complex user request into a sequence of logical PHASES.

**Constraint Checklist:**
1.  **Phase 1 Restriction**: The first phase MUST NOT contain more than 10 domain experts.
2.  **No Delegation in Phase 1**: The tasks for experts in the first phase must be self-contained. You must explicitly instruct them NOT to delegate further.
3.  **Logical Progression**: Subsequent phases should build upon the results of the previous one (e.g., Phase 1: Planning, Phase 2: Implementation).

**User Task:** "%s"

**Your Output MUST be ONLY valid JSON in this exact format:**
{
  "phases": [
    {
      "id": "phase_1_planning",
      "name": "Initial Design and Planning",
      "description": "Define the architecture, requirements, and user experience.",
      "experts": [
        {
          "role": "Lead Architect",
          "expertise": "Overall system design and technology stack selection.",
          "persona": "You are a Lead Architect. Your task is to produce a high-level system architecture document. **You must execute this task yourself and are not allowed to delegate it further.**",
          "task": "Based on the user request, create a detailed technical architecture document, including diagrams and technology choices."
        }
      ]
    },
    {
      "id": "phase_2_implementation",
      "name": "Core Feature Implementation",
      "description": "Develop the key components defined in the planning phase.",
      "experts": [
        {
          "role": "Backend Developer",
          "expertise": "API and database development.",
          "persona": "You are a senior backend developer. You will receive design documents and must implement the corresponding API endpoints.",
          "task": "Implement the core user authentication and profile management API endpoints according to the architecture document from Phase 1."
        }
      ]
    }
  ]
}
`, execution.Task)

	leadPersona := "You are a JSON response generator. You ONLY output valid JSON. You never include explanations, comments, or any text outside the JSON structure."

	result, err := tasks.ExecuteTaskOnAgent(agentContainer.Address, execution.ID+"-planner", leadPersona, planningPrompt, make(map[string]string), true)
	if err != nil || !result.Success {
		return fmt.Errorf("lead agent failed to generate a plan. Error: %v, Agent Message: %s", err, result.GetErrorMessage())
	}

	// Unmarshal the phased plan from the agent's response
	var planResponse struct {
		Phases []ProjectPhase `json:"phases"`
	}
	// Sanitize the response to ensure it's valid JSON
	jsonContent := strings.TrimSpace(result.FinalContent)
	if strings.HasPrefix(jsonContent, "```json") {
		jsonContent = strings.TrimPrefix(jsonContent, "```json")
		jsonContent = strings.TrimSuffix(jsonContent, "```")
		jsonContent = strings.TrimSpace(jsonContent)
	}

	if err := json.Unmarshal([]byte(jsonContent), &planResponse); err != nil {
		return fmt.Errorf("failed to parse phased plan from lead agent: %v. Raw content: %s", err, jsonContent)
	}

	if len(planResponse.Phases) == 0 {
		return fmt.Errorf("lead agent returned a plan with no phases")
	}

	// Update the execution object with the new plan
	tasksMutex.Lock()
	execution.Phases = planResponse.Phases
	execution.CurrentPhase = 0
	execution.UpdatedAt = time.Now()
	tasksMutex.Unlock()

	// Save to database
	saveTaskToDB(execution)

	log.Printf("✅ [%s] Successfully generated a plan with %d phases.", execution.ID, len(execution.Phases))
	return nil
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
	result, err := tasks.ExecuteTaskOnAgent(agentContainer.Address, node.ID, node.Persona, node.Instructions, contextData, true)
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
		synthesisResult, err := tasks.ExecuteTaskOnAgent(agentContainer.Address, node.ID+"-synthesis", node.Persona, synthesisInstructions, synthesisContext, true)
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
