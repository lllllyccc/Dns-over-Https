# DNS-over-HTTPS Server

基于 Go 的轻量级高性能 DNS-over-HTTPS (DoH) 转发服务器。通过 HTTPS 加密 DNS 查询，提升隐私安全，绕过 DNS 层面的限制。

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 为什么需要？

传统 DNS 查询以明文传输，容易被窃听和篡改。本 DoH 服务器：

- **加密** DNS 查询，通过 HTTPS 传输（RFC 8484）
- **转发** 到可信的上游解析器（Google、Cloudflare）
- **缓存** 本地响应，降低延迟
- **屏蔽** DNS 层面的广告和追踪器

## 功能特性

| 功能 | 说明 |
|------|------|
| RFC 8484 兼容 | 完整的 DoH 支持，支持 GET 和 POST 方法 |
| 多上游 | Google DNS、Cloudflare DNS，自动故障转移 |
| 持久化缓存 | 基于 BoltDB，TTL 感知的缓存淘汰 |
| 广告/追踪屏蔽 | 基于 hosts 文件的过滤，支持运行时管理 |
| 管理面板 | Web UI，查看统计、日志、管理过滤规则和缓存 |
| Apple 描述文件 | `.mobileconfig`，iOS/macOS 一键配置 |
| Systemd 服务 | 开机自启、自动重启、零停机更新 |

## 快速开始

```bash
git clone https://github.com/lllllyccc/Dns-over-Https.git
cd Dns-over-Https

cp config.example.yaml config.yaml
# 编辑 config.yaml 填入你的配置

go build -o doh-server ./cmd/doh-server
./doh-server config.yaml
```

## 架构

```
客户端 (DoH) ──► Nginx (443/TLS) ──► Go 服务 (8053) ──► Google DNS
                                          │               Cloudflare DNS
                                          ▼
                                     缓存 (BoltDB)
```

## 配置

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

详见 [DEPLOY_zh.md](DEPLOY_zh.md) 部署指南。

## 管理面板

- **实时统计**：查询总数、缓存命中率、屏蔽数量
- **服务器状态**：CPU、内存、磁盘使用率、运行时间、负载
- **上游健康**：DNS 解析器状态实时监控
- **查询日志**：最近 50 条查询记录（来源、类型、延迟）
- **过滤管理**：添加/删除域名、开关过滤功能
- **缓存管理**：查看条目数、清除缓存

## Apple 设备

1. 在 Safari 中打开 `https://doh.yourdomain.com/doh.mobileconfig`
2. 允许下载描述文件
3. 设置 → 通用 → VPN与设备管理 → 安装

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/dns-query` | GET/POST | DoH DNS 解析 |
| `/health` | GET | 健康检查 |
| `/admin/` | GET | 管理面板 |
| `/admin/api/stats` | GET | 服务器统计 |
| `/admin/api/system` | GET | 系统信息（CPU、内存、磁盘） |
| `/admin/api/health` | GET | 上游 DNS 健康状态 |
| `/admin/api/logs` | GET | 查询日志 |
| `/admin/api/filter` | GET/POST/DELETE | 过滤规则管理 |
| `/admin/api/cache/purge` | POST | 清除 DNS 缓存 |

## 测试

```bash
curl https://doh.yourdomain.com/health

curl -H "accept: application/dns-message" \
  "https://doh.yourdomain.com/dns-query?dns=qqoBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE"
```

## 许可证

[MIT](LICENSE)
