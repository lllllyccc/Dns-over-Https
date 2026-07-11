# DNS-over-HTTPS Server

A lightweight, high-performance DNS-over-HTTPS (DoH) forwarder built with Go. Encrypt your DNS queries via HTTPS to enhance privacy and bypass DNS-based restrictions.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## Why?

Traditional DNS queries are sent in plaintext, making them vulnerable to eavesdropping and manipulation. This DoH server:

- **Encrypts** your DNS queries via HTTPS (RFC 8484)
- **Forwards** to trusted upstream resolvers (Google, Cloudflare)
- **Caches** responses locally to reduce latency
- **Blocks** ads and trackers at the DNS level

## Features

| Feature | Description |
|---------|-------------|
| RFC 8484 Compliant | Full DoH support with GET and POST methods |
| Multi-Upstream | Google DNS, Cloudflare DNS with automatic failover |
| Persistent Cache | BoltDB-backed cache with TTL-aware eviction |
| Ad/Tracker Blocking | Hosts-file based filtering with runtime management |
| Admin Dashboard | Web UI for stats, logs, filter & cache management |
| Apple Profiles | `.mobileconfig` for iOS/macOS one-click setup |
| Systemd Service | Auto-start, auto-restart, zero-downtime updates |

## Quick Start

```bash
git clone https://github.com/lllllyccc/Dns-over-Https.git
cd Dns-over-Https

cp config.example.yaml config.yaml
# Edit config.yaml with your settings

go build -o doh-server ./cmd/doh-server
./doh-server config.yaml
```

## Architecture

```
Client (DoH) ──► Nginx (443/TLS) ──► Go Server (8053) ──► Google DNS
                                          │               Cloudflare DNS
                                          ▼
                                     Cache (BoltDB)
```

## Configuration

```yaml
listen: "127.0.0.1:8053"
admin_listen: "127.0.0.1:8054"
domain: "doh.yourdomain.com"

upstreams:
  - name: "google"
    address: "8.8.8.8:53"
    protocol: "udp"
    weight: 1
  - name: "cloudflare"
    address: "1.1.1.1:53"
    protocol: "udp"
    weight: 1

cache:
  enabled: true
  max_entries: 10000
  default_ttl: 3600

filter:
  enabled: true
  blocklist_path: "./blocklist.txt"

admin:
  username: "admin"
  password: "changeme"
```

See [DEPLOY.md](DEPLOY.md) for full deployment guide.

## Admin Dashboard

- **Real-time stats**: query count, cache hit rate, blocked queries
- **Server status**: CPU, memory, disk usage, uptime, load average
- **Upstream health**: live monitoring of DNS resolver status
- **Query log**: last 50 queries with source, type, and latency
- **Filter management**: add/remove domains, toggle blocking on/off
- **Cache control**: view entries, purge cache

## Apple Devices

1. Open `https://doh.yourdomain.com/doh.mobileconfig` in Safari
2. Allow the profile download
3. Settings → General → VPN & Device Management → Install

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/dns-query` | GET/POST | DoH DNS resolution |
| `/health` | GET | Health check |
| `/admin/` | GET | Admin dashboard |
| `/admin/api/stats` | GET | Server statistics |
| `/admin/api/system` | GET | System info (CPU, RAM, disk) |
| `/admin/api/health` | GET | Upstream DNS health |
| `/admin/api/logs` | GET | Query logs |
| `/admin/api/filter` | GET/POST/DELETE | Blocklist CRUD |
| `/admin/api/cache/purge` | POST | Purge DNS cache |

## Testing

```bash
curl https://doh.yourdomain.com/health

curl -H "accept: application/dns-message" \
  "https://doh.yourdomain.com/dns-query?dns=qqoBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE"
```

## License

[MIT](LICENSE)
