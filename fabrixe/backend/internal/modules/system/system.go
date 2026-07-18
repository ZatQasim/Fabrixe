package system

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fabrixe/fabrixe/pkg/models"
)

// ─────────────────────────────────────────────
// CPU sampling (requires two readings ~100ms apart)
// ─────────────────────────────────────────────

type cpuStat struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func readCPUStats() ([]cpuStat, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var stats []cpuStat
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		var s cpuStat
		s.user, _ = strconv.ParseUint(fields[1], 10, 64)
		s.nice, _ = strconv.ParseUint(fields[2], 10, 64)
		s.system, _ = strconv.ParseUint(fields[3], 10, 64)
		s.idle, _ = strconv.ParseUint(fields[4], 10, 64)
		s.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
		s.irq, _ = strconv.ParseUint(fields[6], 10, 64)
		s.softirq, _ = strconv.ParseUint(fields[7], 10, 64)
		if len(fields) > 8 {
			s.steal, _ = strconv.ParseUint(fields[8], 10, 64)
		}
		stats = append(stats, s)
	}
	return stats, scanner.Err()
}

func cpuPercent(a, b cpuStat) float64 {
	idleDelta := (b.idle + b.iowait) - (a.idle + a.iowait)
	totalA := a.user + a.nice + a.system + a.idle + a.iowait + a.irq + a.softirq + a.steal
	totalB := b.user + b.nice + b.system + b.idle + b.iowait + b.irq + b.softirq + b.steal
	total := totalB - totalA
	if total == 0 {
		return 0
	}
	return math.Round((1.0-float64(idleDelta)/float64(total))*10000) / 100
}

// GetCPUInfo reads live CPU usage from /proc/stat.
func GetCPUInfo() (models.CPUInfo, error) {
	a, err := readCPUStats()
	if err != nil {
		return models.CPUInfo{}, err
	}
	time.Sleep(200 * time.Millisecond)
	b, err := readCPUStats()
	if err != nil {
		return models.CPUInfo{}, err
	}

	info := models.CPUInfo{
		Cores:   runtime.NumCPU(),
		Threads: runtime.NumCPU(),
	}

	// Index 0 is the aggregate "cpu" line
	if len(a) > 0 && len(b) > 0 {
		info.UsageTotal = cpuPercent(a[0], b[0])
	}
	for i := 1; i < len(a) && i < len(b); i++ {
		info.PerCore = append(info.PerCore, cpuPercent(a[i], b[i]))
	}

	// CPU model from /proc/cpuinfo
	info.Model = readCPUModel()

	// CPU frequency
	info.Frequency = readCPUFrequency()

	return info, nil
}

func readCPUModel() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "Unknown"
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
}

func readCPUFrequency() float64 {
	// Try /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq
	data, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq")
	if err == nil {
		val, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		if err == nil {
			return val / 1000 // kHz → MHz
		}
	}
	// Fall back to /proc/cpuinfo
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu MHz") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				return val
			}
		}
	}
	return 0
}

// ─────────────────────────────────────────────
// Memory
// ─────────────────────────────────────────────

func GetMemoryInfo() (models.MemoryInfo, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return models.MemoryInfo{}, err
	}
	defer f.Close()

	kv := make(map[string]uint64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		kv[key] = val * 1024 // kB → bytes
	}

	total := kv["MemTotal"]
	free := kv["MemFree"]
	cached := kv["Cached"] + kv["SReclaimable"]
	buffers := kv["Buffers"]

	// Avoid uint64 underflow: clamp if the subtrahends exceed total
	overhead := free + cached + buffers
	var used uint64
	if overhead < total {
		used = total - overhead
	}

	info := models.MemoryInfo{
		TotalBytes:   total,
		UsedBytes:    used,
		FreeBytes:    free,
		CachedBytes:  cached,
		BuffersBytes: buffers,
		SwapTotal:    kv["SwapTotal"],
		SwapUsed:     kv["SwapTotal"] - kv["SwapFree"],
	}
	if total > 0 {
		info.UsagePercent = math.Round(float64(used)/float64(total)*10000) / 100
	}
	return info, scanner.Err()
}

// ─────────────────────────────────────────────
// Disk
// ─────────────────────────────────────────────

func GetDiskInfo() ([]models.DiskInfo, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := make(map[string]bool)
	var disks []models.DiskInfo

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		device, mountpoint, fstype := fields[0], fields[1], fields[2]

		// Skip non-physical filesystems
		if strings.HasPrefix(device, "tmpfs") || strings.HasPrefix(device, "devtmpfs") ||
			strings.HasPrefix(device, "sysfs") || strings.HasPrefix(device, "proc") ||
			strings.HasPrefix(device, "cgroup") || strings.HasPrefix(device, "pstore") ||
			strings.HasPrefix(fstype, "squashfs") || strings.HasPrefix(device, "overlay") ||
			strings.HasPrefix(device, "none") || strings.HasPrefix(mountpoint, "/sys") ||
			strings.HasPrefix(mountpoint, "/proc") || strings.HasPrefix(mountpoint, "/dev/pts") ||
			strings.HasPrefix(mountpoint, "/run/user") {
			continue
		}

		if seen[mountpoint] {
			continue
		}
		seen[mountpoint] = true

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mountpoint, &stat); err != nil {
			continue
		}

		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		used := total - free
		var pct float64
		if total > 0 {
			pct = math.Round(float64(used)/float64(total)*10000) / 100
		}

		disks = append(disks, models.DiskInfo{
			Device:     device,
			Mountpoint: mountpoint,
			FSType:     fstype,
			Total:      total,
			Used:       used,
			Free:       free,
			UsePercent: pct,
		})
	}
	return disks, scanner.Err()
}

// ─────────────────────────────────────────────
// Network
// ─────────────────────────────────────────────

func GetNetworkInfo() ([]models.NetworkInterface, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Build IP lookup from system interfaces
	ifaceAddrs := make(map[string][]string)
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ifaceAddrs[iface.Name] = append(ifaceAddrs[iface.Name], addr.String())
		}
	}

	var result []models.NetworkInterface
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue // skip header lines
		}
		line := scanner.Text()
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:colonIdx])
		fields := strings.Fields(line[colonIdx+1:])
		if len(fields) < 10 {
			continue
		}
		parse := func(s string) uint64 {
			v, _ := strconv.ParseUint(s, 10, 64)
			return v
		}
		result = append(result, models.NetworkInterface{
			Name:        name,
			BytesRecv:   parse(fields[0]),
			PacketsRecv: parse(fields[1]),
			ErrorsIn:    parse(fields[2]),
			BytesSent:   parse(fields[8]),
			PacketsSent: parse(fields[9]),
			ErrorsOut:   parse(fields[10]),
			IPAddresses: ifaceAddrs[name],
		})
	}
	return result, scanner.Err()
}

// ─────────────────────────────────────────────
// System Info
// ─────────────────────────────────────────────

func GetSystemInfo() (models.SystemInfo, error) {
	hostname, _ := os.Hostname()

	// Uptime from /proc/uptime
	var uptimeSec uint64
	data, err := os.ReadFile("/proc/uptime")
	if err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			f, _ := strconv.ParseFloat(fields[0], 64)
			uptimeSec = uint64(f)
		}
	}

	// Load averages from /proc/loadavg
	var load1, load5, load15 float64
	var procCount int
	data, err = os.ReadFile("/proc/loadavg")
	if err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			load1, _ = strconv.ParseFloat(fields[0], 64)
			load5, _ = strconv.ParseFloat(fields[1], 64)
			load15, _ = strconv.ParseFloat(fields[2], 64)
		}
		if len(fields) >= 4 {
			parts := strings.Split(fields[3], "/")
			if len(parts) == 2 {
				procCount, _ = strconv.Atoi(parts[1])
			}
		}
	}

	// OS release
	osName, osVersion := readOSRelease()

	// Kernel version
	kernel := ""
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		kernel = strings.TrimSpace(string(out))
	}

	arch := runtime.GOARCH
	if out, err := exec.Command("uname", "-m").Output(); err == nil {
		arch = strings.TrimSpace(string(out))
	}

	bootTime := time.Now().Unix() - int64(uptimeSec)

	return models.SystemInfo{
		Hostname:  hostname,
		OS:        osName,
		OSVersion: osVersion,
		Kernel:    kernel,
		Arch:      arch,
		Uptime:    uptimeSec,
		BootTime:  bootTime,
		LoadAvg1:  load1,
		LoadAvg5:  load5,
		LoadAvg15: load15,
		Processes: procCount,
	}, nil
}

func readOSRelease() (name, version string) {
	files := []string{"/etc/os-release", "/usr/lib/os-release"}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		kv := parseKeyValue(string(data))
		n := strings.Trim(kv["NAME"], `"`)
		v := strings.Trim(kv["VERSION"], `"`)
		if v == "" {
			v = strings.Trim(kv["VERSION_ID"], `"`)
		}
		if n != "" {
			return n, v
		}
	}
	return "Linux", ""
}

func parseKeyValue(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}

// ─────────────────────────────────────────────
// Services (systemd via systemctl)
// ─────────────────────────────────────────────

func GetServices() ([]models.ServiceInfo, error) {
	out, err := exec.Command("systemctl", "list-units", "--type=service",
		"--all", "--no-pager", "--no-legend", "--plain").Output()
	if err != nil {
		// systemctl may not be available (e.g., container without systemd)
		return []models.ServiceInfo{}, nil
	}

	var services []models.ServiceInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ".service")
		desc := ""
		if len(fields) > 4 {
			desc = strings.Join(fields[4:], " ")
		}
		services = append(services, models.ServiceInfo{
			Name:        name,
			Description: desc,
			LoadState:   fields[1],
			ActiveState: fields[2],
			SubState:    fields[3],
		})
		if len(services) >= 200 {
			break
		}
	}
	return services, nil
}

func ServiceAction(name, action string) (string, error) {
	allowed := map[string]bool{"start": true, "stop": true, "restart": true, "reload": true, "status": true}
	if !allowed[action] {
		return "", fmt.Errorf("action %q not allowed", action)
	}
	// Validate service name (no shell injection)
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return "", fmt.Errorf("invalid service name")
		}
	}
	out, err := exec.Command("systemctl", action, name+".service").CombinedOutput()
	return string(out), err
}

// ─────────────────────────────────────────────
// Full snapshot (parallelised)
// ─────────────────────────────────────────────

func GetSnapshot() (models.SystemSnapshot, error) {
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		snap   models.SystemSnapshot
		errors []error
	)
	snap.Timestamp = time.Now().Unix()

	type job struct {
		name string
		fn   func() error
	}
	jobs := []job{
		{"system", func() error {
			s, err := GetSystemInfo()
			mu.Lock(); snap.System = s; mu.Unlock()
			return err
		}},
		{"cpu", func() error {
			c, err := GetCPUInfo()
			mu.Lock(); snap.CPU = c; mu.Unlock()
			return err
		}},
		{"memory", func() error {
			m, err := GetMemoryInfo()
			mu.Lock(); snap.Memory = m; mu.Unlock()
			return err
		}},
		{"disk", func() error {
			d, err := GetDiskInfo()
			mu.Lock(); snap.Disks = d; mu.Unlock()
			return err
		}},
		{"network", func() error {
			n, err := GetNetworkInfo()
			mu.Lock(); snap.Network = n; mu.Unlock()
			return err
		}},
	}

	wg.Add(len(jobs))
	for _, j := range jobs {
		j := j
		go func() {
			defer wg.Done()
			if err := j.fn(); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", j.name, err))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(errors) > 0 {
		return snap, errors[0]
	}
	return snap, nil
}

// ─────────────────────────────────────────────
// Hardware inventory
// ─────────────────────────────────────────────

type HardwareInfo struct {
	CPU         string   `json:"cpu"`
	CPUCores    int      `json:"cpu_cores"`
	MemoryTotal uint64   `json:"memory_total_bytes"`
	Disks       []string `json:"disk_models"`
	NetworkCards []string `json:"network_cards"`
	Vendor      string   `json:"vendor"`
	Product     string   `json:"product"`
	SerialNumber string  `json:"serial_number"`
}

func GetHardwareInfo() HardwareInfo {
	hw := HardwareInfo{
		CPU:      readCPUModel(),
		CPUCores: runtime.NumCPU(),
	}

	// Memory
	if mem, err := GetMemoryInfo(); err == nil {
		hw.MemoryTotal = mem.TotalBytes
	}

	// Disk models from /sys/block
	entries, _ := os.ReadDir("/sys/block")
	for _, e := range entries {
		name := e.Name()
		modelPath := filepath.Join("/sys/block", name, "device/model")
		data, err := os.ReadFile(modelPath)
		if err == nil {
			hw.Disks = append(hw.Disks, fmt.Sprintf("%s: %s", name, strings.TrimSpace(string(data))))
		} else if !strings.HasPrefix(name, "loop") && !strings.HasPrefix(name, "ram") {
			hw.Disks = append(hw.Disks, name)
		}
	}

	// DMI data (requires root)
	hw.Vendor = readDMI("sys_vendor")
	hw.Product = readDMI("product_name")
	hw.SerialNumber = readDMI("product_serial")

	return hw
}

func readDMI(field string) string {
	data, err := os.ReadFile("/sys/class/dmi/id/" + field)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
