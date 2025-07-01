package tasktree

import (
	"fmt"
	"sync"
	"time"
)

// Node represents a single task in the hierarchy.
type Node struct {
	ID                string
	ParentID          string
	Persona           string
	Instructions      string
	Status            string // e.g., "pending", "running", "delegated", "completed", "failed"
	Result            string
	SubTaskIDs        []string
	RequiredSubTasks  int
	CompletedSubTasks int
	SubTaskResults    map[string]string // Map of SubTaskID to its result
	lock              sync.Mutex
}

// Tree manages the entire task hierarchy.
type Tree struct {
	Nodes map[string]*Node // Map of TaskID to Node
	lock  sync.RWMutex
}

func NewTree() *Tree {
	return &Tree{
		Nodes: make(map[string]*Node),
	}
}

func (t *Tree) AddNode(parentID, persona, instructions string) *Node {
	t.lock.Lock()
	defer t.lock.Unlock()

	node := &Node{
		ID:             fmt.Sprintf("task-%d", time.Now().UnixNano()), // Unique ID
		ParentID:       parentID,
		Persona:        persona,
		Instructions:   instructions,
		Status:         "pending",
		SubTaskResults: make(map[string]string),
	}
	t.Nodes[node.ID] = node

	if parentID != "" {
		parentNode := t.Nodes[parentID]
		if parentNode != nil {
			parentNode.lock.Lock()
			parentNode.SubTaskIDs = append(parentNode.SubTaskIDs, node.ID)
			parentNode.lock.Unlock()
		}
	}

	return node
}

func (t *Tree) GetNode(taskID string) *Node {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.Nodes[taskID]
}

func (t *Tree) UpdateNodeStatus(taskID, status string) {
	t.lock.RLock()
	node := t.Nodes[taskID]
	t.lock.RUnlock()

	if node != nil {
		node.lock.Lock()
		node.Status = status
		node.lock.Unlock()
	}
}

func (t *Tree) UpdateNodeResult(taskID, result string) {
	t.lock.RLock()
	node := t.Nodes[taskID]
	t.lock.RUnlock()

	if node != nil {
		node.lock.Lock()
		node.Result = result
		node.Status = "completed"
		node.lock.Unlock()
	}
}

func (t *Tree) SetRequiredSubTasks(taskID string, count int) {
	t.lock.RLock()
	node := t.Nodes[taskID]
	t.lock.RUnlock()

	if node != nil {
		node.lock.Lock()
		node.RequiredSubTasks = count
		node.lock.Unlock()
	}
}

func (t *Tree) GetNodeStatus(taskID string) string {
	t.lock.RLock()
	node := t.Nodes[taskID]
	t.lock.RUnlock()

	if node != nil {
		node.lock.Lock()
		defer node.lock.Unlock()
		return node.Status
	}
	return "unknown"
}

// GetSubTaskResults returns the results from all sub-tasks as a context map
func (t *Tree) GetSubTaskResults(taskID string) map[string]string {
	t.lock.RLock()
	node := t.Nodes[taskID]
	t.lock.RUnlock()

	if node == nil {
		return nil
	}

	node.lock.Lock()
	defer node.lock.Unlock()

	results := make(map[string]string)
	for _, subTaskID := range node.SubTaskIDs {
		subNode := t.Nodes[subTaskID]
		if subNode != nil && subNode.Status == "completed" {
			results[subNode.Persona] = subNode.Result
		}
	}

	return results
}

// GetFailedSubTasks returns the IDs of all failed sub-tasks
func (t *Tree) GetFailedSubTasks(taskID string) []string {
	t.lock.RLock()
	node := t.Nodes[taskID]
	t.lock.RUnlock()

	if node == nil {
		return nil
	}

	node.lock.Lock()
	defer node.lock.Unlock()

	var failed []string
	for _, subTaskID := range node.SubTaskIDs {
		subNode := t.Nodes[subTaskID]
		if subNode != nil && subNode.Status == "failed" {
			failed = append(failed, subTaskID)
		}
	}

	return failed
}

// GetCompletedSubTasks returns the IDs of all completed sub-tasks
func (t *Tree) GetCompletedSubTasks(taskID string) []string {
	t.lock.RLock()
	node := t.Nodes[taskID]
	t.lock.RUnlock()

	if node == nil {
		return nil
	}

	node.lock.Lock()
	defer node.lock.Unlock()

	var completed []string
	for _, subTaskID := range node.SubTaskIDs {
		subNode := t.Nodes[subTaskID]
		if subNode != nil && subNode.Status == "completed" {
			completed = append(completed, subTaskID)
		}
	}

	return completed
}
