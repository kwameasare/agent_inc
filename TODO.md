Implementation Guide: Agentic System V2.1Objective: This guide provides the exact steps and code needed to fix the two most critical flaws in the current system. By completing these tasks, the system will become significantly more robust and user-friendly.Developer: Junior LevelEstimated Time: 2-3 hoursTask 1: Fix the UI - Display Expert Results for ApprovalProblem: The user cannot see the work produced by the AI experts, making the "Approve Phase" button a blind action. We need to display the results from each expert within the UI.Step 1.1: Create a New ExpertDetails ComponentThis new component will manage the display of a single expert, including an expandable section for their results.Action: Create a new file: ui/src/components/ExpertDetails.tsxContent: Copy and paste the following code into the new file.// ui/src/components/ExpertDetails.tsx

import { useState } from 'react';
import { CheckCircle, XCircle, Clock, Loader2, ChevronDown, ChevronUp } from 'lucide-react';
import type { DomainExpert } from '../App';

// Helper to get the right icon based on status
const getExpertStatusIcon = (status: string) => {
  switch (status) {
    case 'completed':
      return <CheckCircle className="w-5 h-5 text-green-600" />;
    case 'running':
      return <Loader2 className="w-5 h-5 text-blue-600 animate-spin" />;
    case 'failed':
      return <XCircle className="w-5 h-5 text-red-600" />;
    default:
      return <Clock className="w-5 h-5 text-gray-500" />;
  }
};

interface ExpertDetailsProps {
  expert: DomainExpert;
}

export function ExpertDetails({ expert }: ExpertDetailsProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  // An expert's result can only be viewed if they have completed their task
  const canViewResult = expert.status === 'completed' && expert.result;

  return (
    <div className="activity-item activity-info p-4 my-2">
      <div
        className={`flex items-center justify-between ${canViewResult ? 'cursor-pointer' : ''}`}
        onClick={() => canViewResult && setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center space-x-3">
          {getExpertStatusIcon(expert.status)}
          <div>
            <p className="font-semibold text-gray-800">{expert.role}</p>
            <p className="text-sm text-gray-600">{expert.expertise}</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
            <div className={`px-2 py-1 rounded text-xs font-bold ${
                expert.status === 'completed' ? 'bg-green-100 text-green-800' :
                expert.status === 'running' ? 'bg-blue-100 text-blue-800' :
                expert.status === 'failed' ? 'bg-red-100 text-red-800' :
                'bg-gray-100 text-gray-600'
            }`}>
                {expert.status.toUpperCase()}
            </div>
            {canViewResult && (
                isExpanded ? <ChevronUp className="w-5 h-5 text-gray-600" /> : <ChevronDown className="w-5 h-5 text-gray-600" />
            )}
        </div>
      </div>

      {/* This section will only render if the expert is expanded and has a result */}
      {isExpanded && canViewResult && (
        <div className="mt-4 pt-3 border-t border-gray-200">
          <h6 className="font-bold text-gray-800 text-sm mb-2">Expert's Report:</h6>
          <pre className="text-sm text-gray-700 whitespace-pre-wrap font-mono bg-gray-50 p-3 rounded-lg border border-gray-200">
            {expert.result}
          </pre>
        </div>
      )}
    </div>
  );
}
Step 1.2: Update TaskResult.tsx to Use the New ComponentNow, we'll replace the old expert display logic with our new, more detailed component.Action: Open the file ui/src/components/TaskResult.tsx and apply the following changes.Changes:Import the new ExpertDetails component.Remove the old getExpertStatusIcon function, as it's now inside ExpertDetails.In the JSX, replace the old div that mapped over phase.experts with a call to the <ExpertDetails /> component.// ui/src/components/TaskResult.tsx

import { CheckCircle, XCircle, Clock, Loader2, FileText, AlertTriangle, Users, Pause } from 'lucide-react';
import { useState } from 'react';
import type { Task } from '../App';
import { ExpertDetails } from './ExpertDetails'; // <-- 1. IMPORT THE NEW COMPONENT

interface TaskResultProps {
  task: Task | null;
  onApprovePhase?: (taskId: string, phaseId: string, approved: boolean, feedback?: string) => Promise<any>;
}

export function TaskResult({ task, onApprovePhase }: TaskResultProps) {
  // ... (keep existing state and helper functions like getStatusIcon, getStatusText, etc.)

  // 2. REMOVE the old getExpertStatusIcon function. It's no longer needed here.

  return (
    // ... (keep the outer div and header)
    
        // ... (inside the task.phases.map function)
        {task.phases && task.phases.length > 0 && (
          <div>
            {/* ... */}
            <div className="space-y-2"> {/* Changed from space-y-6 */}
              {task.phases.map((phase, phaseIndex) => (
                <div 
                  key={phase.id}
                  className={`activity-item p-6 ${
                    // ... (existing class logic)
                  }`}
                >
                  {/* ... (existing phase header logic) */}
                  
                  {/* 3. REPLACE the old expert mapping logic */}
                  {phase.experts && phase.experts.length > 0 && (
                    <div className="ml-4 mt-4 space-y-1">
                      <h5 className="text-lg font-bold text-gray-700 mb-2">Domain Experts Assigned:</h5>
                      {phase.experts.map((expert, expertIndex) => (
                        <ExpertDetails key={expertIndex} expert={expert} />
                      ))}
                    </div>
                  )}
                  {/* END REPLACEMENT */}
                  
                  {/* ... (keep the rest of the component for Phase Results, Approval UI, etc.) */}
                </div>
              ))}
            </div>
          </div>
        )}
    // ...
  );
}
Verification: Run the application (npm run dev in the ui directory). Submit a task. When a phase completes and is awaiting approval, you should now be able to click on each completed expert to expand a section and view the text report they generated.Task 2: Implement Programmatic Agent ConstraintsProblem: The system's workflow relies on "instructing" the LLM not to delegate. This is unreliable. We will add a programmatic flag to enforce this rule in code.Step 2.1: Modify the gRPC Protocol DefinitionAction: Open proto/agent.proto and add the can_delegate field.// proto/agent.proto

message TaskRequest {
  string task_id = 1;
  string persona_prompt = 2;
  string task_instructions = 3;
  map<string, string> context_data = 4;
  bool can_delegate = 5; // <-- ADD THIS LINE
}
Step 2.2: Regenerate gRPC CodeThis step is mandatory. After changing the .proto file, you must regenerate the code for both Go and Python.Action: From the root directory of your project (agent_inc), run these two commands:# For Go
protoc --proto_path=proto --go_out=orchestrator --go-grpc_out=orchestrator proto/agent.proto

# For Python
python3 -m grpc_tools.protoc -I./proto --python_out=./agents/generic_agent --grpc_python_out=./agents/generic_agent proto/agent.proto
(Note: You may need to adjust paths based on your exact structure, but this should work for the provided repo.)Step 2.3: Update the Go Orchestrator to Send the FlagWe need to modify the gRPC client to send the new can_delegate flag.Action: Open orchestrator/tasks/client.go and update the ExecuteTaskOnAgent function.// orchestrator/tasks/client.go

// 1. Add the new boolean parameter to the function signature
func ExecuteTaskOnAgent(address, taskID, persona, instructions string, contextData map[string]string, canDelegate bool) (*pb.TaskResult, error) {
    // ... (keep retry logic)
}

func attemptTaskExecution(address, taskID, persona, instructions string, contextData map[string]string, canDelegate bool) (*pb.TaskResult, error) {
    // ... (keep connection logic)

    request := &pb.TaskRequest{
		TaskId:           taskID,
		PersonaPrompt:    persona,
		TaskInstructions: instructions,
		ContextData:      contextData,
		CanDelegate:      canDelegate, // <-- 2. SET THE FLAG in the request object
	}

    // ... (rest of the function)
}
Action: Now, open orchestrator/main.go and pass the flag when calling the experts.// orchestrator/main.go

// Inside the executeDomainExpert function:
func executeDomainExpert(taskID string, phase *ProjectPhase, expert *DomainExpert) {
    // ... (spawn agent container)

    // For Phase 1, delegation is not allowed. For others, it is.
    // This is a simple check; a more robust system might have this as a property of the phase itself.
    isPhaseOne := phase.ID == "phase_1_planning" || strings.HasPrefix(phase.ID, "phase-1")
    canDelegate := !isPhaseOne

    // Pass the new `canDelegate` flag in the function call
    result, err := tasks.ExecuteTaskOnAgent(
        agentContainer.Address, 
        expert.Role, 
        expert.Persona, 
        expert.Task, 
        contextData, 
        canDelegate, // <-- PASS THE FLAG HERE
    )
    
    // ... (rest of the function)
}
Step 2.4: Update the Python Agent to Enforce the FlagFinally, the Python agent must check the flag and change its behavior accordingly.Action: Open agents/generic_agent/agent.py and modify the ExecuteTask method.# agents/generic_agent/agent.py

# ... inside the ExecuteTask method, after you get the `decision` from the LLM
            decision = decision_data.get("decision")
            reason = decision_data.get("reason", "No reason provided")
            
            logger.info(f"🎯 Task {request.task_id}: LLM decision = {decision}, Reason = {reason}")

            # --- MODIFICATION: Enforce the can_delegate flag ---
            if decision == "delegate" and not request.can_delegate:
                logger.warning(f"Task {request.task_id}: Agent decided to delegate, but was not permitted by the orchestrator. Forcing execution.")
                decision = "execute" # Override the LLM's decision
            # --- END MODIFICATION ---

            # Step 2: Act on the (potentially overridden) decision.
            if decision == "delegate":
                # ... (existing delegation logic)
            elif decision == "execute":
                # ... (existing execution logic)
Verification: Run the system and submit a task. Check the orchestrator logs. You should see logs indicating that the Phase 1 experts are being called with can_delegate=false. Check the agent logs. If an agent for a Phase 1 task ever tries to delegate, you should see the "Overriding to 'execute'" warning message.By completing these two tasks, you will have fundamentally improved the system's reliability and user experience, moving it much closer to a production-ready state.