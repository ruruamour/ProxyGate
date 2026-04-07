# ProxyGate

[English](README_EN.md) | [简体中文](README.md)

> **A self-hosted proxy gateway** that aggregates public proxies and subscription nodes, validates them into one pool, and exposes unified HTTP/SOCKS5 outputs with session stickiness.

[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev/)

ProxyGate is a self-hosted proxy gateway written in Go. It ingests public proxies and Clash/V2ray-style subscription nodes, validates them by exit IP, geo location, latency, and HTTPS CONNECT capability, then serves them through unified HTTP and SOCKS5 gateway ports. It also supports sticky sessions and region filtering via `sid`, `t`, `region`, and `st`.

> Inspired by [isboyjc/GoProxy](https://github.com/isboyjc/GoProxy)

## Highlights

- Unified gateway output for both public proxies and subscription nodes
- HTTP and SOCKS5 ports, each with random-rotation and lowest-latency modes
- Session stickiness and geo filtering through auth username extensions
- Clash/V2ray subscription import with built-in sing-box conversion
- Validate-before-activate workflow for subscription nodes
- Automatic refill, optimization, health checks, and failure recovery
- Guest read-only WebUI and admin control panel

## Ports

| Port | Protocol | Mode | Purpose |
|------|----------|------|---------|
| 7777 | HTTP | Random rotation | Crawlers, IP diversity |
| 7776 | HTTP | Lowest latency | Stable outbound traffic |
| 7779 | SOCKS5 | Random rotation | Browsers, SSH, apps |
| 7780 | SOCKS5 | Lowest latency | Long-lived connections |
| 7778 | HTTP | WebUI | Dashboard and management |

## Quick Start

### Docker

```bash
docker compose up -d
```

WebUI:

```text
http://localhost:7778
default password: proxygate
```

By default, `docker compose up -d` pulls the prebuilt image:

```text
ghcr.io/ruruamour/proxygate:latest
```

To customize settings:

```bash
cp .env.example .env
vim .env
docker compose up -d
```

To build from local source instead of pulling the image:

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

### Local Run

```bash
cp .env.example .env
go mod download
go run .
```

Or:

```bash
go build -o proxygate . && ./proxygate
```

Notes:

- Go 1.25 and CGO are required because of `go-sqlite3`
- The app auto-loads a local `.env` file from the repo root
- Exported system environment variables take precedence over `.env`
- `sing-box` is optional at startup, but required for encrypted subscription nodes such as `vmess`, `vless`, `trojan`, `ss`, `hysteria2`, and `anytls`

## Proxy Usage

### HTTP

```bash
curl -x http://localhost:7777 https://httpbin.org/ip
curl -x http://localhost:7776 https://httpbin.org/ip
```

### SOCKS5

```bash
curl --socks5-hostname localhost:7779 https://httpbin.org/ip
curl --socks5-hostname localhost:7780 https://httpbin.org/ip
```

Use `socks5h://` or `curl --socks5-hostname` when you want remote DNS resolution. `socks5://` commonly performs local DNS resolution first.

### With Authentication

```bash
curl -x http://proxy:pass@your-server:7777 https://httpbin.org/ip
curl --socks5-hostname proxy:pass@your-server:7779 https://httpbin.org/ip
```

## Sticky Sessions and Geo Filters

After enabling proxy authentication, you can append extra options to the base username:

```text
proxy-region-US-sid-order-sync-t-10
proxy-region-JP-st-TOKYO-sid-bot-01-t-30
```

Parameters:

- `region`: country code, matched against the `exit_location` prefix
- `st`: state or city keyword, matched against `exit_location`
- `sid`: sticky session ID
- `t`: sticky session TTL in minutes, default `10`, max `120`

## Subscription Import

The admin WebUI supports:

- subscription URLs
- uploaded config files
- Clash YAML
- V2ray-style links
- Base64-encoded payloads
- plain-text proxy lists

Supported protocols include:

- `vmess`
- `vless`
- `trojan`
- `shadowsocks`
- `hysteria2`
- `anytls`
- `http`
- `socks5`

Imported subscription nodes are stored first, validated next, and activated only after they pass checks. Failed nodes are kept in disabled state and can be probed again later.

## Documentation

- [Architecture Notes](POOL_DESIGN.md)
- [Geo Filter Guide](GEO_FILTER.md)
- [Data Directory Guide](DATA_DIRECTORY.md)
- [Test Scripts](test/README.md)
- [Changelog](CHANGELOG.md)

## Disclaimer

This project is for learning, experimentation, and self-hosted technical use.

- Public proxy sources are third-party resources from the internet
- Availability, safety, and stability are not guaranteed
- Users are responsible for legal, operational, and security risks
- Subscription import is intended for managing your own proxy resources

## License

[MIT](LICENSE)
