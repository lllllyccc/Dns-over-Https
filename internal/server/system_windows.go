package server

import (
	"fmt"
	"runtime"
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
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		Uptime:    time.Since(time.Now().Add(-time.Duration(m.Sys) * time.Nanosecond)).Truncate(time.Second).String(),
		CPUUsage:  "-",
		MemTotal:  formatBytes(int64(m.Sys)),
		MemUsed:   formatBytes(int64(m.Alloc)),
		MemUsage:  fmt.Sprintf("%.1f", float64(m.Alloc)/float64(m.Sys)*100),
		DiskTotal: "-",
		DiskUsed:  "-",
		DiskUsage: "-",
		Load1:     "-",
		Load5:     "-",
		Load15:    "-",
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
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
