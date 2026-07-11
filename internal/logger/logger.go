package logger

import (
	"log/slog"
	"sync"
	"time"
)

type QueryLog struct {
	entries []QueryEntry
	maxSize int
	mu      sync.RWMutex
}

type QueryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	ClientIP  string    `json:"client_ip"`
	QueryName string    `json:"query_name"`
	QueryType string    `json:"query_type"`
	Upstream  string    `json:"upstream"`
	CacheHit  bool      `json:"cache_hit"`
	Blocked   bool      `json:"blocked"`
	RTT       int64     `json:"rtt_ms"`
}

type Stats struct {
	TotalQueries   int64            `json:"total_queries"`
	CacheHits      int64            `json:"cache_hits"`
	BlockedQueries int64            `json:"blocked_queries"`
	Errors         int64            `json:"errors"`
	StartTime      time.Time        `json:"start_time"`
	QueryTypes     map[string]int64 `json:"query_types"`
	UpstreamStats  map[string]int64 `json:"upstream_stats"`
}

type stats struct {
	Stats
	mu sync.RWMutex
}

type Logger struct {
	queryLog *QueryLog
	stats    *stats
	slog     *slog.Logger
}

func New(maxLogEntries int, level string) *Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(nil, &slog.HandlerOptions{Level: lvl})
	logger := slog.New(handler)

	return &Logger{
		queryLog: &QueryLog{
			entries: make([]QueryEntry, 0),
			maxSize: maxLogEntries,
		},
		stats: &stats{
			Stats: Stats{
				StartTime:     time.Now(),
				QueryTypes:    make(map[string]int64),
				UpstreamStats: make(map[string]int64),
			},
		},
		slog: logger,
	}
}

func (l *Logger) LogQuery(entry QueryEntry) {
	l.slog.Info("dns_query",
		"client_ip", entry.ClientIP,
		"query_name", entry.QueryName,
		"query_type", entry.QueryType,
		"upstream", entry.Upstream,
		"cache_hit", entry.CacheHit,
		"blocked", entry.Blocked,
		"rtt_ms", entry.RTT,
	)

	l.queryLog.mu.Lock()
	l.queryLog.entries = append(l.queryLog.entries, entry)
	if len(l.queryLog.entries) > l.queryLog.maxSize {
		l.queryLog.entries = l.queryLog.entries[1:]
	}
	l.queryLog.mu.Unlock()

	l.stats.mu.Lock()
	l.stats.TotalQueries++
	if entry.CacheHit {
		l.stats.CacheHits++
	}
	if entry.Blocked {
		l.stats.BlockedQueries++
	}
	l.stats.QueryTypes[entry.QueryType]++
	l.stats.UpstreamStats[entry.Upstream]++
	l.stats.mu.Unlock()
}

func (l *Logger) LogError(msg string, args ...any) {
	l.slog.Error(msg, args...)
	l.stats.mu.Lock()
	l.stats.Errors++
	l.stats.mu.Unlock()
}

func (l *Logger) QueryEntries() []QueryEntry {
	l.queryLog.mu.RLock()
	defer l.queryLog.mu.RUnlock()

	entries := make([]QueryEntry, len(l.queryLog.entries))
	copy(entries, l.queryLog.entries)
	return entries
}

func (l *Logger) GetStats() Stats {
	l.stats.mu.RLock()
	defer l.stats.mu.RUnlock()

	queryTypes := make(map[string]int64)
	for k, v := range l.stats.QueryTypes {
		queryTypes[k] = v
	}
	upstreamStats := make(map[string]int64)
	for k, v := range l.stats.UpstreamStats {
		upstreamStats[k] = v
	}

	return Stats{
		TotalQueries:   l.stats.TotalQueries,
		CacheHits:      l.stats.CacheHits,
		BlockedQueries: l.stats.BlockedQueries,
		Errors:         l.stats.Errors,
		StartTime:      l.stats.StartTime,
		QueryTypes:     queryTypes,
		UpstreamStats:  upstreamStats,
	}
}
