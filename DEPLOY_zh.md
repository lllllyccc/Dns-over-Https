# 部署指南

**注意：本文为DEPLOY.md的简体中文版本，由mimo-v2.5-pro翻译，仅供参考。**

## 前置条件

- Ubuntu 22.04+ 服务器（Oracle Cloud ARM/x86 均可）
- 域名已配置 A 记录指向服务器 IP
- Nginx 已安装
- Certbot 已安装（用于 Let's Encrypt 证书）

## 1. 安装 Go

```bash
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

## 2. 克隆并构建

```bash
sudo mkdir -p /opt/doh-server
sudo chown $USER:$USER /opt/doh-server
cd /opt/doh-server
git clone https://github.com/lllllyccc/Dns-over-Https.git .
go build -o doh-server ./cmd/doh-server
```

## 3. 配置

编辑 `config.yaml`：

```yaml
listen: "127.0.0.1:8053"
admin_listen: "127.0.0.1:8054"
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

admin:
  username: "admin"
  password: "你的安全密码"

logging:
  level: "info"
  query_log: true
  max_log_entries: 10000
```

## 4. Systemd 服务

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

## 5. SSL 证书

```bash
# 确保 DNS A 记录已设置：doh.你的域名.com → 服务器 IP
sudo mkdir -p /var/www/certbot
sudo certbot certonly --webroot -w /var/www/certbot -d doh.你的域名.com
```

## 6. Nginx 反向代理

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

## 7. Apple 描述文件

```bash
sudo mkdir -p /var/www/doh
sudo cp doh.mobileconfig /var/www/doh/
```

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

## 故障排查

| 问题 | 解决方案 |
|------|---------|
| 502 Bad Gateway | 检查服务状态：`systemctl status doh-server` |
| 证书过期 | `sudo certbot renew` |
| 过滤不生效 | 检查 blocklist.txt 格式：`0.0.0.0 domain.com` |
| 管理面板 404 | 确认 Nginx 配置包含 `/admin` location |
| 缓存未持久化 | 检查 `data/` 目录权限 |

## 安全建议

- 管理面板默认仅监听 `127.0.0.1`
- 使用强密码作为管理员凭据
- 建议为管理面板配置 IP 白名单
- 监控查询日志发现异常流量
