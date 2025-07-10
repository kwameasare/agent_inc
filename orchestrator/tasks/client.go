package tasks

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "agentic-engineering-system/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ExecuteTaskOnAgent sends a task to an agent via gRPC and returns the result
func ExecuteTaskOnAgent(address, taskID, persona, instructions string, contextData map[string]string, canDelegate bool) (*pb.TaskResult, error) {
	// Retry logic for connection issues
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🔄 [%s] Attempt %d/%d: Connecting to agent at %s", taskID, attempt, maxRetries, address)

		result, err := attemptTaskExecution(address, taskID, persona, instructions, contextData, canDelegate)
		if err == nil {
			return result, nil
		}

		log.Printf("⚠️ [%s] Attempt %d failed: %v", taskID, attempt, err)

		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * 2 * time.Second
			log.Printf("⏳ [%s] Waiting %v before retry...", taskID, waitTime)
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts", maxRetries)
}

func attemptTaskExecution(address, taskID, persona, instructions string, contextData map[string]string, canDelegate bool) (*pb.TaskResult, error) {
	// Connect to the agent using the exact pattern that works in minimal test
	log.Printf("🔌 [%s] Establishing gRPC connection to %s", taskID, address)

	// Use the exact same pattern as the working minimal test
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %s: %v", address, err)
	}
	defer conn.Close()

	log.Printf("✅ [%s] gRPC client created", taskID)

	// Create gRPC client
	client := pb.NewGenericAgentClient(conn)

	// Prepare the request - ensure ContextData is never nil
	if contextData == nil {
		contextData = make(map[string]string)
	}

	request := &pb.TaskRequest{
		TaskId:           taskID,
		PersonaPrompt:    persona,
		TaskInstructions: instructions,
		ContextData:      contextData,
		CanDelegate:      canDelegate,
	}

	// Execute the task using the same context as connection
	log.Printf("📤 [%s] Sending task to agent...", taskID)
	result, err := client.ExecuteTask(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %v", err)
	}

	log.Printf("✅ [%s] Task completed successfully. Success: %v", taskID, result.Success)
	return result, nil
}
