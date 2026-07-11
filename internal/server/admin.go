package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"dns-over-https/internal/filter"
	"dns-over-https/internal/logger"
	"dns-over-https/internal/resolver"
)

type AdminHandler struct {
	fwd    *resolver.Forwarder
	filter *filter.Filter
	log    *logger.Logger
	cache  *resolver.Cache
}

func NewAdminHandler(fwd *resolver.Forwarder, flt *filter.Filter, log *logger.Logger, cache *resolver.Cache) *AdminHandler {
	return &AdminHandler{
		fwd:    fwd,
		filter: flt,
		log:    log,
		cache:  cache,
	}
}

func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/admin" || path == "/admin/":
		h.serveDashboard(w, r)
	case path == "/admin/api/stats":
		h.serveStats(w, r)
	case path == "/admin/api/health":
		h.serveHealth(w, r)
	case path == "/admin/api/logs":
		h.serveLogs(w, r)
	case path == "/admin/api/filter" && r.Method == "GET":
		h.serveFilterList(w, r)
	case path == "/admin/api/filter" && r.Method == "POST":
		h.serveFilterAdd(w, r)
	case path == "/admin/api/filter" && r.Method == "DELETE":
		h.serveFilterRemove(w, r)
	case path == "/admin/api/filter/toggle" && r.Method == "POST":
		h.serveFilterToggle(w, r)
	case path == "/admin/api/cache/purge" && r.Method == "POST":
		h.serveCachePurge(w, r)
	default:
		http.NotFound(w, r)
	}
}

var dashboardTmpl = template.Must(template.New("dashboard").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DNS-over-HTTPS Admin</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f172a; color: #e2e8f0; padding: 20px; }
        .header { text-align: center; margin-bottom: 30px; }
        .header h1 { font-size: 24px; color: #38bdf8; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 20px; max-width: 1200px; margin: 0 auto; }
        .card { background: #1e293b; border-radius: 12px; padding: 20px; border: 1px solid #334155; }
        .card h2 { font-size: 14px; color: #94a3b8; text-transform: uppercase; margin-bottom: 15px; }
        .stat { font-size: 32px; font-weight: bold; color: #38bdf8; }
        .stat-label { font-size: 12px; color: #64748b; margin-top: 4px; }
        .upstream { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #334155; }
        .upstream:last-child { border: none; }
        .status-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; margin-right: 8px; }
        .status-ok { background: #22c55e; }
        .status-err { background: #ef4444; }
        .log-table { width: 100%; border-collapse: collapse; font-size: 12px; }
        .log-table th { text-align: left; padding: 8px; color: #94a3b8; border-bottom: 1px solid #334155; }
        .log-table td { padding: 8px; border-bottom: 1px solid #1e293b; }
        .badge { padding: 2px 8px; border-radius: 4px; font-size: 11px; }
        .badge-cache { background: #166534; color: #86efac; }
        .badge-filter { background: #7f1d1d; color: #fca5a5; }
        .badge-upstream { background: #1e3a5f; color: #93c5fd; }
        .full-width { grid-column: 1 / -1; }
        .btn { padding: 8px 16px; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; margin: 4px; }
        .btn-primary { background: #38bdf8; color: #0f172a; }
        .btn-danger { background: #ef4444; color: white; }
        .input { padding: 8px 12px; border: 1px solid #334155; border-radius: 6px; background: #0f172a; color: #e2e8f0; font-size: 13px; }
    </style>
</head>
<body>
    <div class="header"><h1>DNS-over-HTTPS Dashboard</h1></div>
    <div class="grid">
        <div class="card"><h2>Total Queries</h2><div class="stat" id="total-queries">-</div><div class="stat-label">Since server start</div></div>
        <div class="card"><h2>Cache Hit Rate</h2><div class="stat" id="cache-rate">-</div><div class="stat-label">Cached responses</div></div>
        <div class="card"><h2>Blocked</h2><div class="stat" id="blocked">-</div><div class="stat-label">Filtered queries</div></div>
        <div class="card"><h2>Upstreams</h2><div id="upstream-list"></div></div>
        <div class="card"><h2>Filter ({{.FilterCount}} domains)</h2>
            <div style="margin-bottom:10px">
                <input class="input" id="new-domain" placeholder="domain.com">
                <button class="btn btn-primary" onclick="addDomain()">Add</button>
            </div>
            <div id="filter-status" style="margin-bottom:10px"></div>
            <div id="filter-list" style="max-height:200px;overflow-y:auto"></div>
        </div>
        <div class="card"><h2>Cache</h2>
            <div id="cache-stats"></div>
            <button class="btn btn-danger" onclick="purgeCache()" style="margin-top:10px">Purge Cache</button>
        </div>
        <div class="card full-width"><h2>Recent Queries</h2>
            <table class="log-table">
                <thead><tr><th>Time</th><th>Client</th><th>Query</th><th>Type</th><th>Source</th><th>RTT</th></tr></thead>
                <tbody id="log-body"></tbody>
            </table>
        </div>
    </div>
    <script>
        async function refresh() {
            const stats = await (await fetch('/admin/api/stats')).json();
            document.getElementById('total-queries').textContent = stats.total_queries;
            const rate = stats.total_queries > 0 ? ((stats.cache_hits / stats.total_queries) * 100).toFixed(1) : '0.0';
            document.getElementById('cache-rate').textContent = rate + '%';
            document.getElementById('blocked').textContent = stats.blocked_queries;

            const health = await (await fetch('/admin/api/health')).json();
            const ul = document.getElementById('upstream-list');
            ul.innerHTML = Object.entries(health).map(([name, ok]) =>
                '<div class="upstream"><span><span class="status-dot ' + (ok ? 'status-ok' : 'status-err') + '"></span>' + name + '</span><span>' + (ok ? 'OK' : 'DOWN') + '</span></div>'
            ).join('');

            const logs = await (await fetch('/admin/api/logs')).json();
            const tbody = document.getElementById('log-body');
            tbody.innerHTML = logs.reverse().slice(0, 50).map(e => {
                let badge = e.cache_hit ? '<span class="badge badge-cache">cache</span>' :
                    e.blocked ? '<span class="badge badge-filter">blocked</span>' :
                    '<span class="badge badge-upstream">' + e.upstream + '</span>';
                return '<tr><td>' + new Date(e.timestamp).toLocaleTimeString() + '</td><td>' + e.client_ip + '</td><td>' + e.query_name + '</td><td>' + e.query_type + '</td><td>' + badge + '</td><td>' + e.rtt_ms + 'ms</td></tr>';
            }).join('');

            const filters = await (await fetch('/admin/api/filter')).json();
            document.getElementById('filter-list').innerHTML = filters.domains.map(d =>
                '<div style="display:flex;justify-content:space-between;padding:4px 0;border-bottom:1px solid #334155"><span>' + d + '</span><button class="btn btn-danger" style="padding:2px 8px;font-size:11px" onclick="removeDomain(\''+d+'\')">x</button></div>'
            ).join('');
            document.getElementById('filter-status').innerHTML = 'Filter: ' + (filters.enabled ? '<span style="color:#22c55e">ON</span>' : '<span style="color:#ef4444">OFF</span>') +
                ' <button class="btn btn-primary" onclick="toggleFilter()" style="padding:4px 8px;font-size:11px">Toggle</button>';

            document.getElementById('cache-stats').innerHTML = '<div style="font-size:13px;margin:5px 0">Entries: ' + (stats.cache_total || 0) + '</div>';
        }
        async function addDomain() {
            const d = document.getElementById('new-domain').value.trim();
            if (!d) return;
            await fetch('/admin/api/filter', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({domain:d})});
            document.getElementById('new-domain').value = '';
            refresh();
        }
        async function removeDomain(d) {
            await fetch('/admin/api/filter', {method:'DELETE', headers:{'Content-Type':'application/json'}, body:JSON.stringify({domain:d})});
            refresh();
        }
        async function toggleFilter() {
            await fetch('/admin/api/filter/toggle', {method:'POST'});
            refresh();
        }
        async function purgeCache() {
            await fetch('/admin/api/cache/purge', {method:'POST'});
            refresh();
        }
        refresh();
        setInterval(refresh, 5000);
    </script>
</body>
</html>`))

func (h *AdminHandler) serveDashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"FilterCount": h.filter.Count(),
	}
	dashboardTmpl.Execute(w, data)
}

func (h *AdminHandler) serveStats(w http.ResponseWriter, r *http.Request) {
	stats := h.log.GetStats()
	cacheTotal, cacheMem := h.cache.Stats()

	result := map[string]interface{}{
		"total_queries":   stats.TotalQueries,
		"cache_hits":      stats.CacheHits,
		"blocked_queries": stats.BlockedQueries,
		"cache_total":     cacheTotal,
		"cache_memory":    cacheMem,
	}

	json.NewEncoder(w).Encode(result)
}

func (h *AdminHandler) serveHealth(w http.ResponseWriter, r *http.Request) {
	health := h.fwd.HealthCheck()
	json.NewEncoder(w).Encode(health)
}

func (h *AdminHandler) serveLogs(w http.ResponseWriter, r *http.Request) {
	entries := h.log.QueryEntries()
	json.NewEncoder(w).Encode(entries)
}

func (h *AdminHandler) serveFilterList(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"enabled": h.filter.IsEnabled(),
		"domains": h.filter.Domains(),
		"count":   h.filter.Count(),
	}
	json.NewEncoder(w).Encode(result)
}

func (h *AdminHandler) serveFilterAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	req.Domain = strings.TrimSpace(strings.ToLower(req.Domain))
	if req.Domain == "" {
		http.Error(w, "domain required", http.StatusBadRequest)
		return
	}
	h.filter.AddDomain(req.Domain)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *AdminHandler) serveFilterRemove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	h.filter.RemoveDomain(req.Domain)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *AdminHandler) serveFilterToggle(w http.ResponseWriter, r *http.Request) {
	h.filter.SetEnabled(!h.filter.IsEnabled())
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"enabled": h.filter.IsEnabled()})
}

func (h *AdminHandler) serveCachePurge(w http.ResponseWriter, r *http.Request) {
	h.cache.Purge()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
