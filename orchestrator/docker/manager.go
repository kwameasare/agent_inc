package docker

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type Manager struct {
	cli         *client.Client
	ctx         context.Context
	nextPort    int
	activePorts map[string]bool
	containers  map[string]string // Map container ID to port
	lock        sync.Mutex
}

type AgentContainer struct {
	ID      string
	Address string // e.g., "localhost:50060"
	Port    string
}

func NewManager(ctx context.Context) (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Manager{
		cli:         cli,
		ctx:         ctx,
		nextPort:    50060, // Start from a high port number
		activePorts: make(map[string]bool),
		containers:  make(map[string]string),
	}, nil
}

func (m *Manager) SpawnAgent(ctx context.Context) (*AgentContainer, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	port := strconv.Itoa(m.nextPort)
	m.nextPort++

	// Use the exact same approach as manual Docker run that works
	// docker run --rm -d -p PORT:PORT -e OPENAI_API_KEY=$OPENAI_API_KEY agentic-engineering-system_generic_agent python agent.py PORT

	hostBinding := nat.PortBinding{
		HostIP:   "", // Use default (empty) instead of "0.0.0.0"
		HostPort: port,
	}
	containerPort, err := nat.NewPort("tcp", port)
	if err != nil {
		return nil, fmt.Errorf("failed to create port: %v", err)
	}

	portBindings := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	// Get the current OPENAI_API_KEY - ensure we get the fresh value
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}
	log.Printf("🔑 Using OpenAI API key ending in: ...%s", openaiKey[len(openaiKey)-4:])
	log.Printf("🔑 Full API key length: %d characters", len(openaiKey))
	log.Printf("🔑 API key starts with: %s...", openaiKey[:20])
	
	// Prepare environment variables for the container
	envVars := []string{"OPENAI_API_KEY=" + openaiKey}
	log.Printf("🔑 Environment variable being passed: OPENAI_API_KEY=%s...%s (length: %d)", 
		openaiKey[:20], openaiKey[len(openaiKey)-4:], len(openaiKey))
	
	// Create with minimal configuration that matches manual approach
	resp, err := m.cli.ContainerCreate(ctx, &container.Config{
		Image:        "agentic-engineering-system_generic_agent",
		Cmd:          []string{"python", "agent.py", port},
		Env:          envVars,
		ExposedPorts: nat.PortSet{containerPort: struct{}{}}, // Explicitly expose the port
	}, &container.HostConfig{
		PortBindings: portBindings,
		AutoRemove:   false, // Disable for debugging - keep containers around to inspect
	}, nil, nil, "")
	if err != nil {
		return nil, err
	}

	if err := m.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	log.Printf("Spawned agent container %s on port %s", resp.ID[:12], port)
	m.containers[resp.ID] = port
	m.activePorts[port] = true

	// Give the container more time to start its gRPC server and initialize
	log.Printf("Waiting for agent in container %s to initialize...", resp.ID[:12])

	// Instead of fixed wait, do health checks
	maxWaitTime := 30 * time.Second
	checkInterval := 1 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		// Try to connect to the port to see if it's accepting connections
		conn, err := net.DialTimeout("tcp", "host.docker.internal:"+port, 2*time.Second)
		if err == nil {
			conn.Close()
			log.Printf("✅ Agent container %s is ready and accepting connections", resp.ID[:12])
			// Give the gRPC server extra time to fully initialize HTTP/2 handling
			time.Sleep(5 * time.Second)
			break
		}

		// Check if we've reached the maximum wait time
		if time.Since(startTime) >= maxWaitTime {
			log.Printf("⚠️ Agent container %s did not become ready within %v", resp.ID[:12], maxWaitTime)
			break
		}

		time.Sleep(checkInterval)
	}

	return &AgentContainer{
		ID:      resp.ID,
		Address: "host.docker.internal:" + port, // Use Docker host reference to reach host-bound ports
		Port:    port,
	}, nil
}

func (m *Manager) StopAgent(ctx context.Context, containerID string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if port, exists := m.containers[containerID]; exists {
		delete(m.activePorts, port)
		delete(m.containers, containerID)
	}

	timeout := 10
	err := m.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		log.Printf("Failed to stop container %s: %v", containerID[:12], err)
		return err
	}

	err = m.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Printf("Failed to remove container %s: %v", containerID[:12], err)
		return err
	}

	log.Printf("Stopped and removed agent container %s", containerID[:12])
	return nil
}

func (m *Manager) GetContainerLogs(ctx context.Context, containerID string) (string, error) {
	out, err := m.cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "50", // Get last 50 lines
	})
	if err != nil {
		return "", err
	}
	defer out.Close()

	buf := make([]byte, 4096)
	n, err := out.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return "", err
	}

	return string(buf[:n]), nil
}

func (m *Manager) CleanupAllAgents() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for containerID := range m.containers {
		timeout := 5
		_ = m.cli.ContainerStop(m.ctx, containerID, container.StopOptions{Timeout: &timeout})
		_ = m.cli.ContainerRemove(m.ctx, containerID, types.ContainerRemoveOptions{})
		log.Printf("Cleaned up container %s", containerID[:12])
	}

	m.containers = make(map[string]string)
	m.activePorts = make(map[string]bool)
}
