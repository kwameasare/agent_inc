package main

import (
	"context"
	"log"
	"os"

	"agentic-engineering-system/docker"
)

func debugContainer() {
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
	log.Printf("Container ID: %s", agentContainer.ID)

	// Get container logs
	if logs, logErr := dockerManager.GetContainerLogs(ctx, agentContainer.ID); logErr == nil {
		log.Printf("🔍 Container logs:\n%s", logs)
	} else {
		log.Printf("⚠️ Could not retrieve container logs: %v", logErr)
	}

	log.Printf("Press Enter to continue...")
	os.Stdin.Read(make([]byte, 1))
}
