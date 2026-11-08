# Sonos MCP

An [MCP](https://modelcontextprotocol.io) server that lets AI assistants
control the Sonos players on your network, and a small client to try it.

| | |
|---|---|
| [`server`](server) | The MCP server: tools for playback, volume, the queue, grouping and Spotify. |
| [`client`](client) | A command-line MCP client for listing and calling the server's tools. |

```bash
cd mcp
go build -o mcp-server ./server
go build -o mcp-client ./client
./mcp-client -tool list_sonos_devices
```
