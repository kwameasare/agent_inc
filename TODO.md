Implementation Plan: Agentic System V2Objective: To implement the architectural changes outlined in CR-AGENTSYS-20250710-01. This document provides a detailed, file-by-file guide for a developer to follow.Prerequisites: Before starting, ensure you have the protoc compiler installed and have run go mod tidy and npm install in the respective orchestrator and ui directories.Task 1: Implement State Persistence with BoltDB (CR Item 2.1)Goal: Make the orchestrator stateful so that task progress is not lost on restart.Step 1.1: Add DependencyIn your terminal, navigate to the orchestrator directory and run:go get go.etcd.io/bbolt
Step 1.2: Modify orchestrator/main.goWe will integrate BoltDB to save and load task state.// orchestrator/main.go

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

	"go.etcd.io/bbolt" // <-- ADD THIS IMPORT
)

// Global state
var (
	dockerManager *docker.Manager
	// The in-memory map now acts as a cache for active tasks. The DB is the source of truth.
	currentTasks = make(map[string]*TaskExecution)
	tasksMutex   sync.RWMutex
	db           *bbolt.DB // <-- ADD THIS
)

// ... (TaskExecution, ProjectPhase, etc. structs remain the same)

func main() {
	// ... (check for API key)

	// --- MODIFICATION: Initialize BoltDB ---
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
	// --- END MODIFICATION ---

	// ... (Docker manager init)

	// Setup HTTP routes
	http.HandleFunc("/api/task", enableCORS(handleTask))
	http.HandleFunc("/api/task/", enableCORS(handleTaskStatus)) // This will now read from DB
	http.HandleFunc("/api/phases/approve", enableCORS(handlePhaseApproval))
	// ... (rest of main)
}

// --- NEW FUNCTIONS: Database Operations ---
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
// --- END NEW FUNCTIONS ---

// --- MODIFICATION: Update Handlers to Use DB ---
func handleTask(w http.ResponseWriter, r *http.Request) {
	// ... (inside POST case)
	// After creating the `execution` object...
	tasksMutex.Lock()
	currentTasks[taskID] = execution
	tasksMutex.Unlock()

	// Save the initial state to the database
	if err := saveTaskState(execution); err != nil {
		log.Printf("ERROR: Failed to save initial state for task %s: %v", taskID, err)
		http.Error(w, "Failed to persist task", http.StatusInternalServerError)
		return
	}

	go executeTask(execution)
	// ... (rest of function)
}

func handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	// ... (get taskID from URL)
	// MODIFICATION: Load directly from the database
	execution, err := loadTaskState(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	// ... (create response and encode)
}
// --- END MODIFICATION ---

// --- MODIFICATION: Persist state changes throughout execution ---
// In every function that modifies the `execution` object, add a call to `saveTaskState`.
// Example in `executeTask`:
func executeTask(execution *TaskExecution) {
    tasksMutex.Lock()
    execution.Status = "running"
    tasksMutex.Unlock()
    if err := saveTaskState(execution); err != nil {
        log.Printf("ERROR: Failed to save running status for %s: %v", execution.ID, err)
    }
    // ... rest of the function
}
Verification: Run the orchestrator. A file named orchestrator.db should be created. Submit a task, then stop and restart the server. You should be able to query the task status via the API and see its last known state.Task 2: Overhaul UI/API (CR Item 2.2)Goal: Make the UI genuinely useful by showing expert results and using efficient real-time updates.Step 2.1: Add API Endpoint for Phase DetailsIn orchestrator/main.go, add the new handler.// orchestrator/main.go

func main() {
    // ...
    http.HandleFunc("/api/task/{taskId}/phase/{phaseId}", enableCORS(handlePhaseDetails))
    // ...
}

func handlePhaseDetails(w http.ResponseWriter, r *http.Request) {
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
Step 2.2: Enhance UI to Display Expert ResultsModify ui/src/components/TaskResult.tsx to fetch and display the results.// ui/src/components/TaskResult.tsx

import { useState, useEffect } from 'react'; // <-- Add useEffect
// ... other imports

// ... TaskResultProps interface

// NEW component for displaying a single expert
const ExpertDetails = ({ expert }: { expert: DomainExpert }) => {
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <div className="bg-white bg-opacity-50 rounded-lg p-3 my-2">
      <div
        className="flex items-center justify-between cursor-pointer"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center space-x-3">
          {getExpertStatusIcon(expert.status)}
          <div>
            <p className="font-semibold text-gray-800">{expert.role}</p>
            <p className="text-sm text-gray-600">{expert.expertise}</p>
          </div>
        </div>
        {/* ... status badge ... */}
      </div>
      {isExpanded && expert.status === 'completed' && (
        <div className="mt-3 pt-3 border-t border-gray-200">
          <h6 className="font-bold text-gray-800 text-sm mb-1">Expert's Report:</h6>
          <pre className="text-xs text-gray-700 whitespace-pre-wrap font-mono bg-gray-50 p-2 rounded">
            {expert.result || "No result was provided by the agent."}
          </pre>
        </div>
      )}
    </div>
  );
};

// ... inside TaskResult component
// ... phase.experts.map(...)
{phase.experts.map((expert, expertIndex) => (
  <ExpertDetails key={expertIndex} expert={expert} />
))}
Step 2.3: Implement Server-Sent Events (SSE)This is a significant change to how the UI gets updates.In orchestrator/main.go:// orchestrator/main.go

// Add a new global map to manage SSE client channels
var sseClients = make(map[string]chan string)
var sseMutex sync.RWMutex

func main() {
    // ...
    http.HandleFunc("/api/task/{taskId}/events", enableCORS(handleTaskEvents))
    // ...
}

// NEW FUNCTION: SSE Handler
func handleTaskEvents(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	messageChan := make(chan string)
	
	sseMutex.Lock()
	sseClients[taskID] = messageChan
	sseMutex.Unlock()

	defer func() {
		sseMutex.Lock()
		delete(sseClients, taskID)
		sseMutex.Unlock()
		close(messageChan)
	}()

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

// NEW FUNCTION: Broadcast updates to SSE clients
func broadcastUpdate(taskID string) {
	sseMutex.RLock()
	clientChan, ok := sseClients[taskID]
	sseMutex.RUnlock()

	if ok {
		// Load the latest state from DB and send it
		execution, err := loadTaskState(taskID)
		if err == nil {
			jsonData, _ := json.Marshal(execution)
			clientChan <- string(jsonData)
		}
	}
}

// MODIFICATION: Call broadcastUpdate whenever state changes.
// Example in `checkPhaseCompletion`:
func checkPhaseCompletion(taskID string, phase *ProjectPhase) {
    // ... after updating phase status
    broadcastUpdate(taskID)
}

// Also call it in `executeTask`, `handlePhaseApproval`, etc.
In ui/src/App.tsx:// ui/src/App.tsx

// ...
function App() {
  // ... state declarations

  // MODIFICATION: Replace polling with SSE
  useEffect(() => {
    if (currentTask?.status === 'running' && currentTask.orchestratorId) {
      const eventSource = new EventSource(`http://localhost:8080/api/task/${currentTask.orchestratorId}/events`);
      
      eventSource.onmessage = (event) => {
        const updatedTaskData = JSON.parse(event.data);
        const taskToUpdate: Task = {
            ...updatedTaskData,
            timestamp: new Date(updatedTaskData.Started),
        };
        
        setCurrentTask(taskToUpdate);
        setTasks(prev => prev.map(t => t.orchestratorId === taskToUpdate.orchestratorId ? taskToUpdate : t));

        if (taskToUpdate.status === 'completed' || taskToUpdate.status === 'failed' || taskToUpdate.status === 'error') {
            eventSource.close();
        }
      };

      eventSource.onerror = () => {
        // Handle error, maybe switch back to polling as a fallback
        eventSource.close();
      };

      return () => {
        eventSource.close();
      };
    }
  }, [currentTask?.orchestratorId, currentTask?.status]);

  const submitTask = async (description: string) => {
    // ... existing logic to POST the task
    // The useEffect above will handle updates automatically once the task is running.
    // The original polling `while` loop can be completely removed.
  };

  // ... rest of component
}
Task 3: Enforce Programmatic Agent Constraints (CR Item 2.3)Goal: Move delegation rules from prompts into code.Step 3.1: Modify proto/agent.proto// proto/agent.proto

message TaskRequest {
  string task_id = 1;
  string persona_prompt = 2;
  string task_instructions = 3;
  map<string, string> context_data = 4;
  bool can_delegate = 5; // <-- ADD THIS LINE
}
Action: After saving this, you MUST regenerate the gRPC files for both Go and Python.Step 3.2: Update Python Agent (agents/generic_agent/agent.py)# agents/generic_agent/agent.py

# ... in ExecuteTask method
            # ... after getting the `decision` from the LLM
            if decision == "delegate":
                # --- MODIFICATION: Enforce the rule ---
                if not request.can_delegate:
                    logger.warning(f"Task {request.task_id}: Agent chose 'delegate' but was not permitted. Overriding to 'execute'.")
                    decision = "execute"
                else:
                    # (original delegation logic here)
                    # ...
                    return result
            
            if decision == "execute": # Note: now an `if` instead of `elif`
                # (original execution logic here)
Step 3.3: Update Go OrchestratorFirst, update the client function signature in orchestrator/tasks/client.go.// orchestrator/tasks/client.go
func ExecuteTaskOnAgent(address, taskID, persona, instructions string, contextData map[string]string, canDelegate bool) (*pb.TaskResult, error) { // <-- Add canDelegate
    // ...
    request := &pb.TaskRequest{
        // ...
        CanDelegate: canDelegate, // <-- Add this field
    }
    // ...
}
Then, set the flag when calling the experts in orchestrator/main.go.// orchestrator/main.go
// Inside executeDomainExpert function

// For Phase 1, delegation is not allowed.
isPhaseOne := phase.ID == "phase_1_planning" // Or a better check
canDelegate := !isPhaseOne

result, err := tasks.ExecuteTaskOnAgent(
    agentContainer.Address, 
    expert.Role, 
    expert.Persona, 
    expert.Task, 
    contextData, 
    canDelegate, // <-- Pass the flag
)
Task 4: Implement Resilient Agent Logic (CR Item 2.4)Goal: Make the agent smarter about handling LLM errors.Step 4.1: Add Self-Correction to Python AgentThis requires a more robust approach to calling the LLM.# agents/generic_agent/agent.py
import pydantic # Add pydantic to requirements.txt

# NEW: Define Pydantic models for validation
class SubTask(pydantic.BaseModel):
    requested_persona: str
    task_details: str

class LLMDecision(pydantic.BaseModel):
    decision: str
    reason: str
    sub_tasks: list[SubTask] = []

# ... in GenericAgentServicer class
def get_validated_decision(self, analysis_prompt: str) -> LLMDecision:
    for i in range(3): # Retry loop
        try:
            response = completion(model="gpt-4o", messages=[{"role": "user", "content": analysis_prompt}], response_format={"type": "json_object"})
            data = json.loads(response.choices[0].message.content)
            validated_data = LLMDecision.model_validate(data)
            return validated_data
        except (json.JSONDecodeError, pydantic.ValidationError) as e:
            logger.warning(f"Validation failed (attempt {i+1}): {e}. Re-prompting with correction.")
            analysis_prompt += f"\n\nYour last response failed validation with this error: {e}. Please correct your response to match the required JSON schema."
    raise ValueError("Failed to get valid decision from LLM after 3 attempts.")

# ... in ExecuteTask method
try:
    decision_data = self.get_validated_decision(analysis_prompt)
    decision = decision_data.decision
    # ... use decision_data.sub_tasks directly
except ValueError as e:
    # return a gRPC error
Action: Add pydantic to agents/generic_agent/requirements.txt.This change request plan is comprehensive and directly addresses the critical flaws in the system. Executing these steps will result in a significantly more robust, usable, and production-ready application.