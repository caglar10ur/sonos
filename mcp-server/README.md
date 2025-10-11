# Sonos MCP Server

The `mcp-server` is a Multi-Capability Protocol (MCP) server designed to provide an interface for controlling Sonos devices on your network. It exposes various Sonos functionalities as callable tools, allowing for programmatic interaction with your Sonos system.

## Features

This server provides the following capabilities for Sonos control:

*   **Device Listing:**
    *   `list_sonos_devices`: List all Sonos devices on the network.
*   **Playback Control:**
    *   `play`: Start playback on a Sonos device.
    *   `stop`: Stop playback on a Sonos device.
    *   `pause`: Pause playback on a Sonos device.
    *   `next`: Play the next track on a Sonos device.
    *   `previous`: Play the previous track on a Sonos device.
*   **Volume Control:**
    *   `get_volume`: Get the current volume of a Sonos device.
    *   `set_volume`: Set the volume of a Sonos device (0-100).
    *   `mute`: Mute a Sonos device.
    *   `unmute`: Unmute a Sonos device.
    *   `get_mute_status`: Get the mute status of a Sonos device.
*   **Queue Management:**
    *   `list_queue`: List the songs in a device's queue.
*   **Media Information:**
    *   `get_now_playing`: Get the currently playing track on a Sonos device.
    *   `get_position_info`: Get the position information of the currently playing song.
    *   `get_media_info`: Get the current media information on a Sonos device.
*   **Audio Input Control:**
    *   `get_audio_input_attributes`: Get the name and icon of the audio input.
    *   `get_line_in_level`: Get the current left and right line-in levels.
    *   `set_line_in_level`: Set the left and right line-in levels.
    *   `select_audio`: Select an audio input by its ObjectID.
    *   `switch_to_line_in`: Switch playback to the line-in input.
    *   `switch_to_queue`: Switch playback to the queue.
*   **Device Information:**
    *   `get_zone_info`: Get detailed information about a Sonos device.
    *   `get_uuid`: Get the UUID of a Sonos device.
*   **Group Management:**
    *   `list_sonos_groups`: List all Sonos zone groups on the network.
    *   `get_zone_group_attributes`: Get the zone group attributes for a given room.
    *   `add_group_member`: Add a member to a Sonos group.
    *   `remove_group_member`: Remove a member from a Sonos group.
    *   `get_group_volume`: Get the current volume of a Sonos group.
    *   `set_group_volume`: Set the volume of a Sonos group.

## Installation

To build the `mcp-server`, navigate to the `mcp-server` directory and run:

```bash
go build -o mcp-server
```

This will create an executable named `mcp-server` in the current directory.

## Usage

The server can be run with different transport mechanisms by setting the `MCP_TRANSPORT` environment variable.

### Stdio (Default)

If `MCP_TRANSPORT` is not set, the server will communicate over standard input/output.

```bash
./mcp-server
```

### HTTP Server

To run as an HTTP server:

```bash
MCP_TRANSPORT=http ./mcp-server
```

By default, it listens on port `8080`. You can change this by setting the `PORT` environment variable:

```bash
PORT=9000 MCP_TRANSPORT=http ./mcp-server
```

### SSE Server

To run as an SSE (Server-Sent Events) server:

```bash
MCP_TRANSPORT=sse ./mcp-server
```

Similar to HTTP, you can specify the port using the `PORT` environment variable.
