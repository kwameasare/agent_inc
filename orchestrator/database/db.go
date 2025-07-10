package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

type TaskExecution struct {
	ID                   string         `json:"id"`
	Task                 string         `json:"task"`
	Status               string         `json:"status"`
	Result               string         `json:"result,omitempty"`
	Error                string         `json:"error,omitempty"`
	Started              time.Time      `json:"started"`
	Phases               []ProjectPhase `json:"phases,omitempty"`
	CurrentPhase         int            `json:"currentPhase"`
	RequiresUserApproval bool           `json:"requiresUserApproval"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

type ProjectPhase struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Status       string            `json:"status"`
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
	Status    string `json:"status"`
	Result    string `json:"result,omitempty"`
}

func NewDB(databaseURL string) (*DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS task_executions (
		id VARCHAR(255) PRIMARY KEY,
		task TEXT NOT NULL,
		status VARCHAR(50) NOT NULL,
		result TEXT,
		error TEXT,
		started TIMESTAMP NOT NULL,
		phases JSONB,
		current_phase INTEGER DEFAULT 0,
		requires_user_approval BOOLEAN DEFAULT true,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_task_executions_status ON task_executions(status);
	CREATE INDEX IF NOT EXISTS idx_task_executions_created_at ON task_executions(created_at);

	CREATE TABLE IF NOT EXISTS phase_results (
		id SERIAL PRIMARY KEY,
		task_id VARCHAR(255) NOT NULL,
		phase_id VARCHAR(255) NOT NULL,
		expert_role VARCHAR(255) NOT NULL,
		expert_result TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		FOREIGN KEY (task_id) REFERENCES task_executions(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_phase_results_task_phase ON phase_results(task_id, phase_id);
	`

	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) SaveTask(task *TaskExecution) error {
	now := time.Now()
	task.UpdatedAt = now

	phasesJSON, err := json.Marshal(task.Phases)
	if err != nil {
		return fmt.Errorf("failed to marshal phases: %v", err)
	}

	query := `
	INSERT INTO task_executions (id, task, status, result, error, started, phases, current_phase, requires_user_approval, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	ON CONFLICT (id) DO UPDATE SET
		status = EXCLUDED.status,
		result = EXCLUDED.result,
		error = EXCLUDED.error,
		phases = EXCLUDED.phases,
		current_phase = EXCLUDED.current_phase,
		requires_user_approval = EXCLUDED.requires_user_approval,
		updated_at = EXCLUDED.updated_at
	`

	_, err = db.conn.Exec(query, task.ID, task.Task, task.Status, task.Result, task.Error,
		task.Started, phasesJSON, task.CurrentPhase, task.RequiresUserApproval, task.CreatedAt, now)

	return err
}

func (db *DB) GetTask(id string) (*TaskExecution, error) {
	query := `
	SELECT id, task, status, result, error, started, phases, current_phase, requires_user_approval, created_at, updated_at
	FROM task_executions WHERE id = $1
	`

	var task TaskExecution
	var phasesJSON []byte

	err := db.conn.QueryRow(query, id).Scan(
		&task.ID, &task.Task, &task.Status, &task.Result, &task.Error,
		&task.Started, &phasesJSON, &task.CurrentPhase, &task.RequiresUserApproval,
		&task.CreatedAt, &task.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if len(phasesJSON) > 0 {
		if err := json.Unmarshal(phasesJSON, &task.Phases); err != nil {
			return nil, fmt.Errorf("failed to unmarshal phases: %v", err)
		}
	}

	return &task, nil
}

func (db *DB) GetAllTasks() ([]*TaskExecution, error) {
	query := `
	SELECT id, task, status, result, error, started, phases, current_phase, requires_user_approval, created_at, updated_at
	FROM task_executions ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*TaskExecution
	for rows.Next() {
		var task TaskExecution
		var phasesJSON []byte

		err := rows.Scan(
			&task.ID, &task.Task, &task.Status, &task.Result, &task.Error,
			&task.Started, &phasesJSON, &task.CurrentPhase, &task.RequiresUserApproval,
			&task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(phasesJSON) > 0 {
			if err := json.Unmarshal(phasesJSON, &task.Phases); err != nil {
				return nil, fmt.Errorf("failed to unmarshal phases: %v", err)
			}
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

func (db *DB) SavePhaseResult(taskID, phaseID, expertRole, result string) error {
	query := `
	INSERT INTO phase_results (task_id, phase_id, expert_role, expert_result)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (task_id, phase_id, expert_role) DO UPDATE SET
		expert_result = EXCLUDED.expert_result,
		created_at = NOW()
	`

	_, err := db.conn.Exec(query, taskID, phaseID, expertRole, result)
	return err
}

func (db *DB) GetPhaseResults(taskID, phaseID string) (map[string]string, error) {
	query := `
	SELECT expert_role, expert_result 
	FROM phase_results 
	WHERE task_id = $1 AND phase_id = $2
	`

	rows, err := db.conn.Query(query, taskID, phaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]string)
	for rows.Next() {
		var role, result string
		if err := rows.Scan(&role, &result); err != nil {
			return nil, err
		}
		results[role] = result
	}

	return results, nil
}

func (db *DB) DeleteTask(id string) error {
	query := `DELETE FROM task_executions WHERE id = $1`
	_, err := db.conn.Exec(query, id)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}
