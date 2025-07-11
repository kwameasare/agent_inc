package main

import (
	"context"
	"log"
	"time"

	pb "agentic-engineering-system/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main2() {
	address := "127.0.0.1:50061"

	log.Printf("🔌 Establishing gRPC connection to %s", address)

	// Use the older gRPC Dial API for better compatibility
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to create gRPC client for %s: %v", address, err)
	}
	defer conn.Close()

	log.Printf("✅ gRPC client created")

	// Create gRPC client
	client := pb.NewGenericAgentClient(conn)

	// Prepare the request
	request := &pb.TaskRequest{
		TaskId:           "test-go-client",
		PersonaPrompt:    "You are a helpful assistant.",
		TaskInstructions: "Say hello and confirm you can receive this message.",
		ContextData:      make(map[string]string),
	}

	// Execute the task with a timeout for task execution
	taskCtx, taskCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer taskCancel()

	// Execute the task
	log.Printf("📤 Sending task to agent...")
	result, err := client.ExecuteTask(taskCtx, request)
	if err != nil {
		log.Fatalf("gRPC call failed: %v", err)
	}

	log.Printf("✅ Task completed. Success: %v", result.Success)
	log.Printf("📄 Final Content: %s", result.FinalContent)
	log.Printf("❌ Error Message: %s", result.ErrorMessage)
}
