# ProxyGate Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-04-11

## Active Technologies

- Go 1.25.0 + Go standard library, `github.com/mattn/go-sqlite3`, `gopkg.in/yaml.v3`, embedded HTML/JavaScript in Go source (001-stats-consistency-audit)

## Project Structure

```text
config/
custom/
pool/
proxy/
storage/
validator/
webui/
test/
```

## Commands

- `go test ./...`
- `go build -o proxygate .`

## Code Style

Go 1.25.0: Follow standard conventions

## Recent Changes

- 001-stats-consistency-audit: Added Go 1.25.0 + Go standard library, `github.com/mattn/go-sqlite3`, `gopkg.in/yaml.v3`, embedded HTML/JavaScript in Go source

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
