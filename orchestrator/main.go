package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"agentic-engineering-system/docker"
	"agentic-engineering-system/tasks"
	"agentic-engineering-system/tasktree"
)

func main() {
	// Check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	ctx := context.Background()
	dockerManager, err := docker.NewManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create docker manager: %v", err)
	}
	defer dockerManager.CleanupAllAgents()

	taskTree := tasktree.NewTree()
	var wg sync.WaitGroup

	// 1. Initial decomposition
	initialPersona := "You are an elite Engineering Project Manager and Technical Lead with expertise in software architecture, system design, and cross-functional team coordination. You excel at breaking down complex technical projects into manageable components and coordinating specialized teams."
	initialTask := "Design and architect a comprehensive e-commerce platform that can handle 1 million concurrent users. The platform should include: user authentication system, product catalog with search capabilities, shopping cart and checkout system, payment processing integration, order management system, inventory tracking, admin dashboard, mobile API, security implementation, and deployment strategy with auto-scaling capabilities."

	rootNode := taskTree.AddNode("", initialPersona, initialTask)
	log.Printf("Starting workflow with root task: %s", rootNode.ID)

	wg.Add(1)
	go executeNode(ctx, &wg, taskTree, dockerManager, rootNode)

	wg.Wait()
	log.Println("--- Entire workflow completed ---")
	log.Printf("Final Result for Root Task (%s):\n%s\n", rootNode.ID, rootNode.Result)
}

// executeNode is the core recursive function of the orchestrator.
func executeNode(ctx context.Context, wg *sync.WaitGroup, tree *tasktree.Tree, dm *docker.Manager, node *tasktree.Node) {
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
				defer subTaskWg.Done()
				executeNode(ctx, &sync.WaitGroup{}, tree, dm, child)
			}(childNode, i+1)
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
