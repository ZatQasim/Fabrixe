package models

import "time"

// ─────────────────────────────────────────────
// User
// ─────────────────────────────────────────────

type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	FullName     string     `json:"full_name"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	FailedLogins int        `json:"-"`
	LockedUntil  *time.Time `json:"-"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ─────────────────────────────────────────────
// Device
// ─────────────────────────────────────────────

type Device struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Fingerprint string     `json:"fingerprint"`
	IPAddress   string     `json:"ip_address"`
	MACAddress  string     `json:"mac_address"`
	DeviceType  string     `json:"device_type"`
	IsTrusted   bool       `json:"is_trusted"`
	FirstSeen   time.Time  `json:"first_seen"`
	LastSeen    time.Time  `json:"last_seen"`
	AddedBy     *int64     `json:"added_by,omitempty"`
	Notes       string     `json:"notes"`
}

// ─────────────────────────────────────────────
// AuditLog
// ─────────────────────────────────────────────

type AuditLog struct {
	ID          int64     `json:"id"`
	EventType   string    `json:"event_type"`
	Description string    `json:"description"`
	UserID      *int64    `json:"user_id,omitempty"`
	Username    string    `json:"username"`
	IPAddress   string    `json:"ip_address"`
	Resource    string    `json:"resource"`
	Outcome     string    `json:"outcome"`
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─────────────────────────────────────────────
// ScheduledTask
// ─────────────────────────────────────────────

type ScheduledTask struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Command     string     `json:"command"`
	Schedule    string     `json:"schedule"`
	IsActive    bool       `json:"is_active"`
	LastRun     *time.Time `json:"last_run,omitempty"`
	LastStatus  string     `json:"last_status"`
	LastOutput  string     `json:"last_output"`
	NextRun     *time.Time `json:"next_run,omitempty"`
	CreatedBy   *int64     `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ─────────────────────────────────────────────
// Deployment
// ─────────────────────────────────────────────

type Deployment struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	DeployType  string     `json:"deploy_type"`
	Config      string     `json:"config"`
	Status      string     `json:"status"`
	LastRun     *time.Time `json:"last_run,omitempty"`
	LastOutput  string     `json:"last_output"`
	CreatedBy   *int64     `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ─────────────────────────────────────────────
// CommunicationNode
// ─────────────────────────────────────────────

type CommunicationNode struct {
	ID          int64      `json:"id"`
	NodeID      string     `json:"node_id"`
	DisplayName string     `json:"display_name"`
	Endpoint    string     `json:"endpoint"`
	PublicKey   string     `json:"public_key"`
	Fingerprint string     `json:"fingerprint"`
	IsTrusted   bool       `json:"is_trusted"`
	Status      string     `json:"status"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
	AddedBy     *int64     `json:"added_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ─────────────────────────────────────────────
// Alert
// ─────────────────────────────────────────────

type Alert struct {
	ID         int64      `json:"id"`
	Level      string     `json:"level"`
	Source     string     `json:"source"`
	Message    string     `json:"message"`
	IsResolved bool       `json:"is_resolved"`
	ResolvedBy *int64     `json:"resolved_by,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ─────────────────────────────────────────────
// System Metrics (real-time, not persisted)
// ─────────────────────────────────────────────

type CPUInfo struct {
	Model      string    `json:"model"`
	Cores      int       `json:"cores"`
	Threads    int       `json:"threads"`
	UsageTotal float64   `json:"usage_total_percent"`
	PerCore    []float64 `json:"per_core_percent"`
	Frequency  float64   `json:"frequency_mhz"`
}

type MemoryInfo struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	FreeBytes      uint64  `json:"free_bytes"`
	CachedBytes    uint64  `json:"cached_bytes"`
	BuffersBytes   uint64  `json:"buffers_bytes"`
	SwapTotal      uint64  `json:"swap_total_bytes"`
	SwapUsed       uint64  `json:"swap_used_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

type DiskInfo struct {
	Device     string  `json:"device"`
	Mountpoint string  `json:"mountpoint"`
	FSType     string  `json:"fstype"`
	Total      uint64  `json:"total_bytes"`
	Used       uint64  `json:"used_bytes"`
	Free       uint64  `json:"free_bytes"`
	UsePercent float64 `json:"use_percent"`
}

type NetworkInterface struct {
	Name        string `json:"name"`
	BytesRecv   uint64 `json:"bytes_recv"`
	BytesSent   uint64 `json:"bytes_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	ErrorsIn    uint64 `json:"errors_in"`
	ErrorsOut   uint64 `json:"errors_out"`
	IPAddresses []string `json:"ip_addresses"`
}

type SystemInfo struct {
	Hostname   string  `json:"hostname"`
	OS         string  `json:"os"`
	OSVersion  string  `json:"os_version"`
	Kernel     string  `json:"kernel"`
	Arch       string  `json:"arch"`
	Uptime     uint64  `json:"uptime_seconds"`
	BootTime   int64   `json:"boot_time_unix"`
	LoadAvg1   float64 `json:"load_avg_1"`
	LoadAvg5   float64 `json:"load_avg_5"`
	LoadAvg15  float64 `json:"load_avg_15"`
	Processes  int     `json:"process_count"`
}

type SystemSnapshot struct {
	Timestamp  int64              `json:"timestamp"`
	System     SystemInfo         `json:"system"`
	CPU        CPUInfo            `json:"cpu"`
	Memory     MemoryInfo         `json:"memory"`
	Disks      []DiskInfo         `json:"disks"`
	Network    []NetworkInterface `json:"network"`
}

// ─────────────────────────────────────────────
// Service (systemd / SysV)
// ─────────────────────────────────────────────

type ServiceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LoadState   string `json:"load_state"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
	UnitFile    string `json:"unit_file"`
	PID         int    `json:"pid,omitempty"`
}

// ─────────────────────────────────────────────
// API Envelope
// ─────────────────────────────────────────────

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *PageMeta   `json:"meta,omitempty"`
}

type PageMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func OK(data interface{}) Response {
	return Response{Success: true, Data: data}
}

func Fail(msg string) Response {
	return Response{Success: false, Error: msg}
}
