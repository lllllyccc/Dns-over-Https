# DNS-over-HTTPS 部署指南

## 项目简介

基于 Go 的 DNS-over-HTTPS 转发服务器，支持：
- RFC 8484 兼容的 DoH 端点
- 多上游 DNS（Google 8.8.8.8、Cloudflare 1.1.1.1）自动故障转移
- BoltDB 持久化缓存（TTL 感知）
- 广告/追踪域名过滤
- Web 管理面板（统计、日志、过滤管理、缓存清理）
- Nginx 反代 + Let's Encrypt TLS
- Apple .mobileconfig 描述文件

## 架构

```
客户端 (DoH) → Nginx (443, TLS) → Go Server (127.0.0.1:8053) → 上游 DNS
                                        ↓
                                   管理面板 (127.0.0.1:8054) → Nginx → /admin
```

## 前置条件

- Ubuntu 22.04+ 服务器（Oracle Cloud ARM/x86 均可）
- 域名已配置 A 记录指向服务器 IP
- Nginx 已安装并运行
- Certbot 已安装（用于 Let's Encrypt 证书）

## 快速部署

### 1. 安装 Go

```bash
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### 2. 克隆并构建

```bash
sudo mkdir -p /opt/doh-server
sudo chown $USER:$USER /opt/doh-server
cd /opt/doh-server
git clone https://github.com/lllllyccc/Dns-over-Https.git .
go build -o doh-server ./cmd/doh-server
```

### 3. 配置

编辑 `config.yaml`：

```yaml
listen: "127.0.0.1:8053"          # 仅本地监听
admin_listen: "127.0.0.1:8054"    # 管理面板仅本地
domain: "doh.你的域名.com"

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

logging:
  level: "info"
  query_log: true
  max_log_entries: 10000
```

### 4. 创建 systemd 服务

```bash
sudo tee /etc/systemd/system/doh-server.service << 'EOF'
[Unit]
Description=DNS-over-HTTPS Server
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/opt/doh-server
ExecStart=/opt/doh-server/doh-server /opt/doh-server/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable doh-server
sudo systemctl start doh-server
```

### 5. 配置 SSL 证书

```bash
# DNS 添加 A 记录: doh.你的域名.com → 服务器 IP
# 等待 DNS 生效后：
sudo certbot certonly --webroot -w /var/www/certbot -d doh.你的域名.com
```

### 6. 配置 Nginx 反代

```bash
sudo tee /etc/nginx/sites-available/doh << 'EOF'
server {
    listen 443 ssl http2;
    server_name doh.你的域名.com;

    ssl_certificate /etc/letsencrypt/live/doh.你的域名.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/doh.你的域名.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;

    location /dns-query {
        proxy_pass http://127.0.0.1:8053/dns-query;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_request_buffering off;
    }

    location /admin {
        proxy_pass http://127.0.0.1:8054;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /admin/api/ {
        proxy_pass http://127.0.0.1:8054/admin/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /health {
        proxy_pass http://127.0.0.1:8053/health;
    }

    location = /doh.mobileconfig {
        alias /var/www/doh/doh.mobileconfig;
        default_type application/x-apple-aspen-config;
        add_header Content-Disposition "attachment; filename=doh.mobileconfig";
    }

    location / {
        return 404;
    }
}

server {
    listen 80;
    server_name doh.你的域名.com;
    return 301 https://$host$request_uri;
}
EOF

sudo ln -sf /etc/nginx/sites-available/doh /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### 7. 部署 Apple 描述文件

```bash
sudo mkdir -p /var/www/doh
sudo cp doh.mobileconfig /var/www/doh/
```

## 使用方式

### DoH 客户端

```bash
# curl 测试
curl -H "accept: application/dns-message" \
  "https://doh.你的域名.com/dns-query?dns=qqoBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE"

# 健康检查
curl https://doh.你的域名.com/health
```

### 管理面板

访问 `https://doh.你的域名.com/admin/`

功能：
- 实时查询统计（总数、缓存命中率、屏蔽数）
- 上游 DNS 健康状态
- 最近 50 条查询日志
- 过滤规则管理（添加/删除/开关）
- 缓存管理（手动清除）

### Apple 设备

在 iPhone/iPad/Mac Safari 中打开：

```
https://doh.你的域名.com/doh.mobileconfig
```

安装后所有 DNS 查询自动通过 DoH 转发。

## 运维

### 查看日志

```bash
sudo journalctl -u doh-server -f
```

### 重启服务

```bash
sudo systemctl restart doh-server
```

### 更新代码

```bash
cd /opt/doh-server
git pull
go build -o doh-server ./cmd/doh-server
sudo systemctl restart doh-server
```

### 编辑过滤列表

```bash
vim /opt/doh-server/blocklist.txt
sudo systemctl restart doh-server
```

或通过管理面板在线增删。

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/dns-query` | GET/POST | DoH DNS 查询 |
| `/health` | GET | 健康检查 |
| `/admin/` | GET | 管理面板 |
| `/admin/api/stats` | GET | 服务器统计 |
| `/admin/api/health` | GET | 上游健康状态 |
| `/admin/api/logs` | GET | 查询日志 |
| `/admin/api/filter` | GET/POST/DELETE | 过滤规则管理 |
| `/admin/api/filter/toggle` | POST | 开关过滤 |
| `/admin/api/cache/purge` | POST | 清除缓存 |

## 故障排查

| 问题 | 解决方案 |
|------|---------|
| 502 Bad Gateway | 检查 doh-server 是否运行: `systemctl status doh-server` |
| 证书过期 | `sudo certbot renew` |
| 过滤不生效 | 检查 blocklist.txt 格式，每行 `0.0.0.0 domain.com` |
| 管理面板 404 | 确认 Nginx 配置包含 `/admin` location |
| 缓存未命中 | 检查 data/ 目录权限，确认 BoltDB 可写 |

## 安全建议

- 管理面板仅监听 `127.0.0.1`，不暴露公网
- 如需公网访问管理面板，建议添加 Basic Auth 或 IP 白名单
- 定期更新上游 DNS 列表
- 监控查询日志发现异常流量
