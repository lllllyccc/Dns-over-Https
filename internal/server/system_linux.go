package server

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type SystemInfo struct {
	Uptime    string `json:"uptime"`
	CPUUsage  string `json:"cpu_usage"`
	MemTotal  string `json:"mem_total"`
	MemUsed   string `json:"mem_used"`
	MemUsage  string `json:"mem_usage"`
	DiskTotal string `json:"disk_total"`
	DiskUsed  string `json:"disk_used"`
	DiskUsage string `json:"disk_usage"`
	Load1     string `json:"load_1"`
	Load5     string `json:"load_5"`
	Load15    string `json:"load_15"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func getSystemInfo() SystemInfo {
	info := SystemInfo{
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if secs, err := strconv.ParseFloat(fields[0], 64); err == nil {
				d := time.Duration(secs * float64(time.Second))
				days := int(d.Hours()) / 24
				hours := int(d.Hours()) % 24
				mins := int(d.Minutes()) % 60
				if days > 0 {
					info.Uptime = fmt.Sprintf("%dd %dh %dm", days, hours, mins)
				} else if hours > 0 {
					info.Uptime = fmt.Sprintf("%dh %dm", hours, mins)
				} else {
					info.Uptime = fmt.Sprintf("%dm", mins)
				}
			}
		}
	}

	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			info.Load1 = fields[0]
			info.Load5 = fields[1]
			info.Load15 = fields[2]
		}
	}

	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		var memTotal, memAvail int64
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "MemTotal:") {
				memTotal = parseMemKB(line)
			} else if strings.HasPrefix(line, "MemAvailable:") {
				memAvail = parseMemKB(line)
			}
		}
		if memTotal > 0 {
			memUsed := memTotal - memAvail
			info.MemTotal = formatBytes(memTotal * 1024)
			info.MemUsed = formatBytes(memUsed * 1024)
			info.MemUsage = fmt.Sprintf("%.1f", float64(memUsed)/float64(memTotal)*100)
		}
	}

	if data, err := os.ReadFile("/proc/stat"); err == nil {
		lines := strings.SplitN(string(data), "\n", 2)
		if len(lines) > 0 {
			fields := strings.Fields(lines[0])
			if len(fields) >= 5 {
				user, _ := strconv.ParseInt(fields[1], 10, 64)
				nice, _ := strconv.ParseInt(fields[2], 10, 64)
				system, _ := strconv.ParseInt(fields[3], 10, 64)
				idle, _ := strconv.ParseInt(fields[4], 10, 64)
				total := user + nice + system + idle
				if total > 0 {
					info.CPUUsage = fmt.Sprintf("%.1f", float64(total-idle)/float64(total)*100)
				}
			}
		}
	}

	info.DiskTotal, info.DiskUsed, info.DiskUsage = getDiskUsage("/")

	return info
}

func parseMemKB(line string) int64 {
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		if val, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
			return val
		}
	}
	return 0
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func getDiskUsage(path string) (total, used, usage string) {
	var stat syscallStatfs
	if err := syscallStatfsFn(path, &stat); err != nil {
		return "-", "-", "-"
	}
	totalBytes := int64(stat.blocks) * int64(stat.bsize)
	freeBytes := int64(stat.bavail) * int64(stat.bsize)
	usedBytes := totalBytes - freeBytes
	usagePct := 0.0
	if totalBytes > 0 {
		usagePct = float64(usedBytes) / float64(totalBytes) * 100
	}
	return formatBytes(totalBytes), formatBytes(usedBytes), fmt.Sprintf("%.1f", usagePct)
}
