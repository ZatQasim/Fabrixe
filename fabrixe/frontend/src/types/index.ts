// ─────────────────────────────────────────────
// Auth
// ─────────────────────────────────────────────

export interface User {
  id: number
  username: string
  email: string
  full_name: string
  role: 'administrator' | 'operator' | 'viewer'
  last_login?: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  expires_at: string
  token_type: string
  user?: User
}

// ─────────────────────────────────────────────
// System
// ─────────────────────────────────────────────

export interface CPUInfo {
  model: string
  cores: number
  threads: number
  usage_total_percent: number
  per_core_percent: number[]
  frequency_mhz: number
}

export interface MemoryInfo {
  total_bytes: number
  used_bytes: number
  free_bytes: number
  cached_bytes: number
  buffers_bytes: number
  swap_total_bytes: number
  swap_used_bytes: number
  usage_percent: number
}

export interface DiskInfo {
  device: string
  mountpoint: string
  fstype: string
  total_bytes: number
  used_bytes: number
  free_bytes: number
  use_percent: number
}

export interface NetworkInterface {
  name: string
  bytes_recv: number
  bytes_sent: number
  packets_recv: number
  packets_sent: number
  errors_in: number
  errors_out: number
  ip_addresses: string[]
}

export interface SystemInfo {
  hostname: string
  os: string
  os_version: string
  kernel: string
  arch: string
  uptime_seconds: number
  boot_time_unix: number
  load_avg_1: number
  load_avg_5: number
  load_avg_15: number
  process_count: number
}

export interface SystemSnapshot {
  timestamp: number
  system: SystemInfo
  cpu: CPUInfo
  memory: MemoryInfo
  disks: DiskInfo[]
  network: NetworkInterface[]
}

export interface HardwareInfo {
  cpu: string
  cpu_cores: number
  memory_total_bytes: number
  disk_models: string[]
  network_cards: string[]
  vendor: string
  product: string
  serial_number: string
}

export interface ServiceInfo {
  name: string
  description: string
  load_state: string
  active_state: string
  sub_state: string
  unit_file: string
  pid?: number
}

// ─────────────────────────────────────────────
// Alerts
// ─────────────────────────────────────────────

export interface Alert {
  id: number
  level: 'info' | 'warning' | 'critical'
  source: string
  message: string
  is_resolved: boolean
  resolved_by?: number
  resolved_at?: string
  created_at: string
}

// ─────────────────────────────────────────────
// Deployments
// ─────────────────────────────────────────────

export interface Deployment {
  id: number
  name: string
  description: string
  deploy_type: 'script' | 'docker' | 'systemd'
  config: string
  status: 'idle' | 'running' | 'success' | 'failed'
  last_run?: string
  last_output: string
  created_by?: number
  created_at: string
  updated_at: string
}

export interface ScheduledTask {
  id: number
  name: string
  description: string
  command: string
  schedule: string
  is_active: boolean
  last_run?: string
  last_status: 'pending' | 'running' | 'success' | 'failed'
  last_output: string
  next_run?: string
  created_by?: number
  created_at: string
  updated_at: string
}

// ─────────────────────────────────────────────
// Security
// ─────────────────────────────────────────────

export interface UserRecord {
  id: number
  username: string
  email: string
  role: string
  full_name: string
  last_login?: string
  is_active: boolean
  failed_logins: number
  locked_until?: string
  created_at: string
  updated_at: string
}

export interface AuditLog {
  id: number
  event_type: string
  description: string
  user_id?: number
  username: string
  ip_address: string
  resource: string
  outcome: 'success' | 'failure' | 'warning'
  metadata: string
  created_at: string
}

export interface DeviceRecord {
  id: number
  name: string
  fingerprint: string
  ip_address: string
  mac_address: string
  device_type: string
  is_trusted: boolean
  first_seen: string
  last_seen: string
  notes: string
}

export interface Session {
  id: string
  user_id: number
  username: string
  ip_address: string
  user_agent: string
  created_at: string
  expires_at: string
  last_seen: string
}

// ─────────────────────────────────────────────
// Communication
// ─────────────────────────────────────────────

export interface CommunicationNode {
  id: number
  node_id: string
  display_name: string
  endpoint: string
  public_key: string
  fingerprint: string
  is_trusted: boolean
  status: 'online' | 'offline' | 'unknown'
  last_seen?: string
  created_at: string
  updated_at: string
}

export interface NodeIdentity {
  node_id: string
  hostname: string
  public_key: string
  fingerprint: string
  version: string
}

// ─────────────────────────────────────────────
// API
// ─────────────────────────────────────────────

export interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  error?: string
  meta?: { total: number; limit: number; offset: number }
}

// ─────────────────────────────────────────────
// WebSocket
// ─────────────────────────────────────────────

export interface WsMessage {
  type: 'snapshot'
  payload: SystemSnapshot
}
