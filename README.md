# DNS-over-HTTPS Server

A full-featured DoH forwarder written in Go. Forwards DNS queries to upstream providers (Google, Cloudflare) with caching, ad/tracker blocking, and an admin dashboard.

## Features

- **RFC 8484 compliant** DoH server (GET + POST)
- **Multiple upstreams**: Google DNS, Cloudflare DNS, with failover
- **DNS caching**: BoltDB-backed persistent cache with TTL
- **Ad/tracker blocking**: Hosts-file based filtering
- **Admin dashboard**: Real-time stats, query log, filter management
- **Structured logging**: JSON logs with query details
- **Let's Encrypt TLS**: Automatic certificate provisioning

## Quick Start

```bash
# Copy and edit config
cp config.example.yaml config.yaml
vim config.yaml

# Build
go build -o doh-server ./cmd/doh-server

# Run
./do-server config.yaml
```

## Configuration

Edit `config.yaml`:

```yaml
listen: "0.0.0.0:443"
admin_listen: "127.0.0.1:8443"
domain: "dns.yourdomain.com"

upstreams:
  - name: "google"
    address: "8.8.8.8:53"
    protocol: "udp"
    weight: 1
  - name: "cloudflare"
    address: "1.1.1.1:53"
    protocol: "udp"
    weight: 1
```

## Usage

### DoH Client

```bash
# GET request
curl -H "accept: application/dns-message" \
  "https://dns.yourdomain.com/dns-query?dns=AAABAAABAAAAAAAAB2V4YW1wbGUDAQEA"

# With dig (if supported)
dig @dns.yourdomain.com -p 443 example.com
```

### Admin Dashboard

Open `https://127.0.0.1:8443` in browser (local access only).

## Oracle Free Tier Deployment

1. Provision Ubuntu 22.04 ARM instance
2. Install Go:
   ```bash
   wget https://go.dev/dl/go1.22.0.linux-arm64.tar.gz
   sudo tar -C /usr/local -xzf go1.22.0.linux-arm64.tar.gz
   export PATH=$PATH:/usr/local/go/bin
   ```
3. Build for ARM:
   ```bash
   GOOS=linux GOARCH=arm64 go build -o doh-server ./cmd/doh-server
   ```
4. Configure DNS: Point `dns.yourdomain.com` → server IP
5. Run with systemd (see below)

### Systemd Service

```ini
[Unit]
Description=DNS-over-HTTPS Server
After=network.target

[Service]
Type=simple
User=doh
WorkingDirectory=/opt/doh-server
ExecStart=/opt/doh-server/doh-server /opt/doh-server/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Docker

```bash
docker build -t doh-server .
docker run -p 443:443 -p 8443:8443 -v ./config.yaml:/app/config.yaml doh-server
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/dns-query` | GET/POST | DoH endpoint |
| `/health` | GET | Health check |
| `/admin/` | GET | Dashboard |
| `/admin/api/stats` | GET | Server statistics |
| `/admin/api/health` | GET | Upstream health |
| `/admin/api/logs` | GET | Query logs |
| `/admin/api/filter` | GET/POST/DELETE | Blocklist management |
| `/admin/api/filter/toggle` | POST | Toggle filter |
| `/admin/api/cache/purge` | POST | Purge cache |

## License

MIT
