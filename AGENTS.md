# AGENTS.md

## Build/Run Commands
```bash
go build -o json2md .
./json2md
# Server runs on http://localhost:8080
```

## Project Structure
```
json2md/
├── main.go           # Gin web server with /api/convert endpoint
├── converter.go      # JSON to Markdown table converter logic
├── templates/
│   └── index.html    # Web UI (3-panel JSON input / Markdown preview / Markdown code)
├── go.mod
└── go.sum
```

## Commands
- **Build**: `go build -o json2md .`
- **Run**: `go run .` or `./json2md`
- **Test**: No tests exist yet
- **Lint**: `go vet ./...` (standard Go vet)

## Architecture
- **main.go**: Gin server on :8080, serves `templates/index.html` at `/`, handles `POST /api/convert` with JSON body `{json: string}`
- **converter.go**: Parses JSON with comments (`//` comments parsed as field descriptions), extracts field names, types, required/optional flags, and descriptions. Outputs Markdown table with columns: 字段名, 类型, 必填, 描述
- **templates/index.html**: 3-panel web UI (JSON input, Markdown preview table, Markdown code)

## JSON Comment Format
Comments in JSON are parsed as field descriptions:
```json
{
  "field": "value", // 必填，描述
  "optional": "val" // 非必填，描述
}
```
Supports `必填`/`必填，`/`必填,` (required), `非必填`/`选填`/`可选` (optional).

## Commands for CI/Lint
```bash
go vet ./...
go build ./...
```