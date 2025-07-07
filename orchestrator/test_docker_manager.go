package main

import (
	"context"
	"log"
	"time"

	"agentic-engineering-system/docker"
	pb "agentic-engineering-system/proto/agentic-engineering-system/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func dockerManagerTest() {
	ctx := context.Background()
	dockerManager, err := docker.NewManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create docker manager: %v", err)
	}
	defer dockerManager.CleanupAllAgents()

	// Spawn agent
	log.Printf("🐳 Spawning agent container...")
	agentContainer, err := dockerManager.SpawnAgent(ctx)
	if err != nil {
		log.Fatalf("Failed to spawn agent container: %v", err)
	}
	defer func() {
		log.Printf("🧹 Cleaning up agent container %s", agentContainer.ID[:12])
		if err := dockerManager.StopAgent(ctx, agentContainer.ID); err != nil {
			log.Printf("⚠️ Failed to cleanup container: %v", err)
		}
	}()

	log.Printf("✅ Agent container spawned: %s at %s", agentContainer.ID[:12], agentContainer.Address)

	// Try to connect and send a simple request
	address := agentContainer.Address

	log.Printf("🔌 Establishing gRPC connection to %s", address)

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
		TaskId:           "test-docker-manager",
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
	if result.ErrorMessage != "" {
		log.Printf("❌ Error Message: %s", result.ErrorMessage)
	}
}
