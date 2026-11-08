# MCP Client (Go)

A simple CLI tool to connect to an MCP (Model Context Protocol) server.

## Features
- Connect via **stdio** (launches the server as a subprocess).
- Connect via **Streamable HTTP** (connects to a running MCP server).
- List available tools and their input schemas.
- Call specific tools with JSON arguments.

## Installation

Ensure you have Go installed. Then build the client and the
[server](../server) it launches, in the `mcp` directory:

```bash
go build -o mcp-client ./client
go build -o mcp-server ./server
```

## Usage

### List tools (stdio)
By default, it tries to launch `./mcp-server` in the current directory.

```bash
./mcp-client
```

### List tools (HTTP)
```bash
./mcp-client -transport http -url http://localhost:8888/mcp
```

### Call a tool
```bash
./mcp-client -tool get_volume -tool-args '{"room_name": "Living Room"}'
```

### Configuration
| Flag | Default | Description |
|------|---------|-------------|
| `-transport` | `stdio` | Transport type (`stdio` or `http`) |
| `-url` | `http://localhost:8888/mcp` | Server URL for HTTP transport |
| `-cmd` | `./mcp-server` | Server command for stdio transport |
| `-args` | (empty) | Server arguments (comma separated) |
| `-tool` | (empty) | Tool name to call |
| `-tool-args` | `{}` | Tool arguments in JSON format |
| `-timeout` | `30s` | Operation timeout |
