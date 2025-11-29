# Sonos MCP Server

The `mcp-server` is a Multi-Capability Protocol (MCP) server designed to provide an interface for controlling Sonos devices on your network. It exposes various Sonos functionalities as callable tools, allowing for programmatic interaction with your Sonos system.

## Features

This server provides the following capabilities for Sonos control:

*   **Device Listing:**
    *   `list_sonos_devices`: List all Sonos devices on the network. Discovery results are cached for performance.
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
*   **Spotify Integration:**
    *   `search_spotify`: Search for a track, album, or artist on Spotify and returns its URI.
    *   `play_spotify_uri`: Play a Spotify URI on a Sonos device.

## Installation

To build the `mcp-server`, navigate to the `mcp-server` directory and run:

```bash
go build -o mcp-server
```

This will create an executable named `mcp-server` in the current directory.

## Usage

The server supports the following command-line flags:

*   `-transport`: Transport type for MCP server (`http` or `stdio`). Default is `stdio`.
*   `-port`: Port for the HTTP server. Default is `8888`.
*   `-search-timeout`: Timeout for Sonos device search (e.g., `2s`, `500ms`). Default is `2s`.
*   `-spotify-client-id`: Spotify client ID (can also be set via `SPOTIFY_CLIENT_ID` env var).
*   `-spotify-client-secret`: Spotify client secret (can also be set via `SPOTIFY_CLIENT_SECRET` env var).

### Stdio (Default)

If the `-transport` flag is not provided or set to `stdio`, the server will communicate over standard input/output. This is the standard mode for use with MCP clients (like Claude Desktop).

```bash
./mcp-server
```

To increase the device discovery timeout if your network is slow:

```bash
./mcp-server -search-timeout 5s
```

### HTTP Server

To run as an HTTP server:

```bash
./mcp-server -transport http
```

By default, it listens on port `8888`. You can change this by using the `-port` flag:

```bash
./mcp-server -transport http -port 9000
```

## Environment Variables

The following environment variables are required for Spotify integration:

*   `SPOTIFY_CLIENT_ID`: Your Spotify application client ID.
*   `SPOTIFY_CLIENT_SECRET`: Your Spotify application client secret.

### Obtaining Spotify Credentials

To obtain your `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET`:

1.  Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard).
2.  Log in with your Spotify account.
3.  Click on "Create an app".
4.  Fill in the App Name and App Description. You can leave the Redirect URI empty for this server.
5.  After creating the app, your Client ID will be visible. Click "Show client secret" to reveal your Client Secret.
6.  Use these credentials as environment variables when running the `mcp-server`.