package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// ─────────────────────────────────────────────
// Deployments
// ─────────────────────────────────────────────

func (h *Handler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Conn().Query(`
		SELECT id, name, description, deploy_type, config, status, last_run, last_output, created_by, created_at, updated_at
		FROM deployments ORDER BY updated_at DESC`)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, description, deployType, config, status, createdAt, updatedAt string
		var lastRun, lastOutput sql.NullString
		var createdBy sql.NullInt64
		if err := rows.Scan(&id, &name, &description, &deployType, &config, &status, &lastRun, &lastOutput, &createdBy, &createdAt, &updatedAt); err != nil {
			continue
		}
		items = append(items, map[string]interface{}{
			"id": id, "name": name, "description": description,
			"deploy_type": deployType, "config": config, "status": status,
			"last_run": lastRun.String, "last_output": lastOutput.String,
			"created_by": createdBy.Int64, "created_at": createdAt, "updated_at": updatedAt,
		})
	}
	if items == nil {
		items = []map[string]interface{}{}
	}
	ok(w, items)
}

func (h *Handler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid deployment id")
		return
	}
	var name, description, deployType, config, status, createdAt, updatedAt string
	var lastRun, lastOutput sql.NullString
	var createdBy sql.NullInt64
	err = h.db.Conn().QueryRow(`
		SELECT name, description, deploy_type, config, status, last_run, last_output, created_by, created_at, updated_at
		FROM deployments WHERE id = ?`, id).
		Scan(&name, &description, &deployType, &config, &status, &lastRun, &lastOutput, &createdBy, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		notFound(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	ok(w, map[string]interface{}{
		"id": id, "name": name, "description": description,
		"deploy_type": deployType, "config": config, "status": status,
		"last_run": lastRun.String, "last_output": lastOutput.String,
		"created_by": createdBy.Int64, "created_at": createdAt, "updated_at": updatedAt,
	})
}

type deploymentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DeployType  string `json:"deploy_type"`
	Config      string `json:"config"`
}

func (h *Handler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req deploymentRequest
	if err := decode(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	if req.Name == "" {
		badRequest(w, "name is required")
		return
	}
	validTypes := map[string]bool{"script": true, "docker": true, "systemd": true}
	if !validTypes[req.DeployType] {
		badRequest(w, "deploy_type must be script, docker, or systemd")
		return
	}
	if req.Config == "" {
		req.Config = "{}"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := h.db.Conn().Exec(`
		INSERT INTO deployments (name, description, deploy_type, config, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'idle', ?, ?, ?)`,
		req.Name, req.Description, req.DeployType, req.Config, claims.UserID, now, now)
	if err != nil {
		internalError(w, err)
		return
	}
	newID, _ := res.LastInsertId()
	h.writeAuditLog("deployment.created", fmt.Sprintf("deployment %q created by %s", req.Name, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	created(w, map[string]interface{}{"id": newID})
}

func (h *Handler) UpdateDeployment(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid id")
		return
	}
	var req deploymentRequest
	if err := decode(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`
		UPDATE deployments SET name=?, description=?, deploy_type=?, config=?, updated_at=? WHERE id=?`,
		req.Name, req.Description, req.DeployType, req.Config, now, id)
	h.writeAuditLog("deployment.updated", fmt.Sprintf("deployment id=%d updated by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "deployment updated"})
}

func (h *Handler) DeleteDeployment(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	_, _ = h.db.Conn().Exec(`DELETE FROM deployments WHERE id = ?`, id)
	h.writeAuditLog("deployment.deleted", fmt.Sprintf("deployment id=%d deleted by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "deployment deleted"})
}

// POST /api/deployment/{id}/run
func (h *Handler) RunDeployment(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid id")
		return
	}

	var name, deployType, config string
	err = h.db.Conn().QueryRow(`SELECT name, deploy_type, config FROM deployments WHERE id = ?`, id).
		Scan(&name, &deployType, &config)
	if err == sql.ErrNoRows {
		notFound(w)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`UPDATE deployments SET status='running', last_run=? WHERE id=?`, now, id)

	h.writeAuditLog("deployment.run_started", fmt.Sprintf("deployment %q started by %s", name, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")

	go func() {
		output, runErr := executeDeployment(deployType, config)
		status := "success"
		if runErr != nil {
			status = "failed"
			output += "\nERROR: " + runErr.Error()
		}
		fin := time.Now().UTC().Format(time.RFC3339)
		_, _ = h.db.Conn().Exec(`
			UPDATE deployments SET status=?, last_output=?, updated_at=? WHERE id=?`,
			status, output, fin, id)
		outcome := "success"
		if runErr != nil {
			outcome = "failure"
		}
		h.writeAuditLog("deployment.run_finished",
			fmt.Sprintf("deployment %q finished with status %s", name, status),
			&claims.UserID, claims.Username, "", outcome)
	}()

	ok(w, map[string]string{"message": "deployment started", "status": "running"})
}

func executeDeployment(deployType, config string) (string, error) {
	switch deployType {
	case "script":
		// Config is expected to contain a shell script
		if strings.TrimSpace(config) == "" || config == "{}" {
			return "", fmt.Errorf("no script configured")
		}
		cmd := exec.Command("bash", "-c", config)
		out, err := cmd.CombinedOutput()
		return string(out), err
	case "docker":
		// Config should be a docker command
		parts := strings.Fields(config)
		if len(parts) == 0 {
			return "", fmt.Errorf("no docker command configured")
		}
		cmd := exec.Command("docker", parts...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	case "systemd":
		// Config should be a service name
		svcName := strings.TrimSpace(config)
		if svcName == "" || svcName == "{}" {
			return "", fmt.Errorf("no service name configured")
		}
		cmd := exec.Command("systemctl", "restart", svcName)
		out, err := cmd.CombinedOutput()
		return string(out), err
	default:
		return "", fmt.Errorf("unknown deploy type: %s", deployType)
	}
}

// ─────────────────────────────────────────────
// Scheduled Tasks
// ─────────────────────────────────────────────

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Conn().Query(`
		SELECT id, name, description, command, schedule, is_active, last_run, last_status, last_output, next_run, created_by, created_at, updated_at
		FROM scheduled_tasks ORDER BY name`)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, description, command, schedule, lastStatus, createdAt, updatedAt string
		var isActive int
		var lastRun, lastOutput, nextRun sql.NullString
		var createdBy sql.NullInt64
		if err := rows.Scan(&id, &name, &description, &command, &schedule, &isActive, &lastRun, &lastStatus, &lastOutput, &nextRun, &createdBy, &createdAt, &updatedAt); err != nil {
			continue
		}
		items = append(items, map[string]interface{}{
			"id": id, "name": name, "description": description, "command": command, "schedule": schedule,
			"is_active": isActive == 1, "last_run": lastRun.String, "last_status": lastStatus,
			"last_output": lastOutput.String, "next_run": nextRun.String,
			"created_by": createdBy.Int64, "created_at": createdAt, "updated_at": updatedAt,
		})
	}
	if items == nil {
		items = []map[string]interface{}{}
	}
	ok(w, items)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid id")
		return
	}
	var name, description, command, schedule, lastStatus, createdAt, updatedAt string
	var isActive int
	var lastRun, lastOutput, nextRun sql.NullString
	var createdBy sql.NullInt64
	err = h.db.Conn().QueryRow(`
		SELECT name, description, command, schedule, is_active, last_run, last_status, last_output, next_run, created_by, created_at, updated_at
		FROM scheduled_tasks WHERE id = ?`, id).
		Scan(&name, &description, &command, &schedule, &isActive, &lastRun, &lastStatus, &lastOutput, &nextRun, &createdBy, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		notFound(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	ok(w, map[string]interface{}{
		"id": id, "name": name, "description": description, "command": command, "schedule": schedule,
		"is_active": isActive == 1, "last_run": lastRun.String, "last_status": lastStatus,
		"last_output": lastOutput.String, "next_run": nextRun.String,
		"created_by": createdBy.Int64, "created_at": createdAt, "updated_at": updatedAt,
	})
}

type taskRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Schedule    string `json:"schedule"`
	IsActive    *bool  `json:"is_active"`
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req taskRequest
	if err := decode(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	if req.Name == "" || req.Command == "" || req.Schedule == "" {
		badRequest(w, "name, command, and schedule are required")
		return
	}
	isActive := 1
	if req.IsActive != nil && !*req.IsActive {
		isActive = 0
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := h.db.Conn().Exec(`
		INSERT INTO scheduled_tasks (name, description, command, schedule, is_active, last_status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?)`,
		req.Name, req.Description, req.Command, req.Schedule, isActive, claims.UserID, now, now)
	if err != nil {
		internalError(w, err)
		return
	}
	newID, _ := res.LastInsertId()
	h.writeAuditLog("deployment.task_created", fmt.Sprintf("task %q created by %s", req.Name, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	created(w, map[string]interface{}{"id": newID})
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	var req taskRequest
	if err := decode(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	isActive := 1
	if req.IsActive != nil && !*req.IsActive {
		isActive = 0
	}
	_, _ = h.db.Conn().Exec(`
		UPDATE scheduled_tasks SET name=?, description=?, command=?, schedule=?, is_active=?, updated_at=? WHERE id=?`,
		req.Name, req.Description, req.Command, req.Schedule, isActive, now, id)
	h.writeAuditLog("deployment.task_updated", fmt.Sprintf("task id=%d updated by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "task updated"})
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	_, _ = h.db.Conn().Exec(`DELETE FROM scheduled_tasks WHERE id = ?`, id)
	h.writeAuditLog("deployment.task_deleted", fmt.Sprintf("task id=%d deleted by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "task deleted"})
}

func (h *Handler) RunTask(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	var name, command string
	err := h.db.Conn().QueryRow(`SELECT name, command FROM scheduled_tasks WHERE id = ?`, id).Scan(&name, &command)
	if err == sql.ErrNoRows {
		notFound(w)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`UPDATE scheduled_tasks SET last_status='running', last_run=?, updated_at=? WHERE id=?`, now, now, id)

	h.writeAuditLog("deployment.task_run", fmt.Sprintf("task %q manually triggered by %s", name, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")

	go func() {
		cmd := exec.Command("bash", "-c", command)
		out, runErr := cmd.CombinedOutput()
		status := "success"
		if runErr != nil {
			status = "failed"
		}
		fin := time.Now().UTC().Format(time.RFC3339)
		_, _ = h.db.Conn().Exec(`
			UPDATE scheduled_tasks SET last_status=?, last_output=?, updated_at=? WHERE id=?`,
			status, string(out), fin, id)
	}()

	ok(w, map[string]string{"message": "task started", "status": "running"})
}
