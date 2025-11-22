package main

import (
	"context"
	"log"
	"os"

	"github.com/caglar10ur/sonos/mcp-server/handlers"
	"github.com/caglar10ur/sonos/mcp-server/sonoscontrol"
	"github.com/caglar10ur/sonos/mcp-server/spotify"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Sonos",
		"1.1.0",
		server.WithToolCapabilities(false),
	)

	sonosController, err := sonoscontrol.NewSonosController()
	if err != nil {
		log.Fatalf("Creating a sonos controller failed: %v", err)
	}

	spotifyClient, err := spotify.NewSpotifyClient(context.Background())
	if err != nil {
		log.Fatalf("Creating a spotify client failed: %v", err)
	}

	h := handlers.NewHandlers(sonosController, spotifyClient)

	listTool := mcp.NewTool(
		"list_sonos_devices",
		mcp.WithDescription("List all Sonos devices on the network"),
	)
	s.AddTool(listTool, h.ListSonosDevicesHandler)

	nowPlayingTool := mcp.NewTool(
		"get_now_playing",
		mcp.WithDescription("Get the currently playing track on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get media info from")),
	)
	s.AddTool(nowPlayingTool, h.GetNowPlayingHandler)

	playTool := mcp.NewTool(
		"play",
		mcp.WithDescription("Start playback on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to start playback in")),
	)
	s.AddTool(playTool, h.PlayHandler)

	stopTool := mcp.NewTool(
		"stop",
		mcp.WithDescription("Stop playback on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to stop playback in")),
	)
	s.AddTool(stopTool, h.StopHandler)

	pauseTool := mcp.NewTool(
		"pause",
		mcp.WithDescription("Pause playback on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to pause playback in")),
	)
	s.AddTool(pauseTool, h.PauseHandler)

	nextTool := mcp.NewTool(
		"next",
		mcp.WithDescription("Play the next track on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to play the next track in")),
	)
	s.AddTool(nextTool, h.NextHandler)

	previousTool := mcp.NewTool(
		"previous",
		mcp.WithDescription("Play the previous track on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to play the previous track in")),
	)
	s.AddTool(previousTool, h.PreviousHandler)

	getVolumeTool := mcp.NewTool(
		"get_volume",
		mcp.WithDescription("Get the current volume of a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get the volume from")),
	)
	s.AddTool(getVolumeTool, h.GetVolumeHandler)

	setVolumeTool := mcp.NewTool(
		"set_volume",
		mcp.WithDescription("Set the volume of a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to set the volume for")),
		mcp.WithNumber("volume", mcp.Required(),
			mcp.Description("The volume level to set (0-100)")),
	)
	s.AddTool(setVolumeTool, h.SetVolumeHandler)

	listQueueTool := mcp.NewTool(
		"list_queue",
		mcp.WithDescription("List the songs in the device's queue"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to list the queue for")),
	)
	s.AddTool(listQueueTool, h.ListQueueHandler)

	getPositionInfoTool := mcp.NewTool(
		"get_position_info",
		mcp.WithDescription("Get the position information of the currently playing song"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get position info from")),
	)
	s.AddTool(getPositionInfoTool, h.GetPositionInfoHandler)

	muteTool := mcp.NewTool(
		"mute",
		mcp.WithDescription("Mute a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to mute")),
	)
	s.AddTool(muteTool, h.MuteHandler)

	unmuteTool := mcp.NewTool(
		"unmute",
		mcp.WithDescription("Unmute a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to unmute")),
	)
	s.AddTool(unmuteTool, h.UnmuteHandler)

	getMuteStatusTool := mcp.NewTool(
		"get_mute_status",
		mcp.WithDescription("Get the mute status of a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get the mute status from")),
	)
	s.AddTool(getMuteStatusTool, h.GetMuteStatusHandler)

	getAudioInputAttributesTool := mcp.NewTool(
		"get_audio_input_attributes",
		mcp.WithDescription("Get the name and icon of the audio input"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get audio input attributes from")),
	)
	s.AddTool(getAudioInputAttributesTool, h.GetAudioInputAttributesHandler)

	getLineInLevelTool := mcp.NewTool(
		"get_line_in_level",
		mcp.WithDescription("Get the current left and right line-in levels"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get line-in levels from")),
	)
	s.AddTool(getLineInLevelTool, h.GetLineInLevelHandler)

	setLineInLevelTool := mcp.NewTool(
		"set_line_in_level",
		mcp.WithDescription("Set the left and right line-in levels"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to set line-in levels for")),
		mcp.WithNumber("desired_left_line_in_level", mcp.Required(),
			mcp.Description("The desired left line-in level")),
		mcp.WithNumber("desired_right_line_in_level", mcp.Required(),
			mcp.Description("The desired right line-in level")),
	)
	s.AddTool(setLineInLevelTool, h.SetLineInLevelHandler)

	selectAudioTool := mcp.NewTool(
		"select_audio",
		mcp.WithDescription("Select an audio input by its ObjectID"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to select audio for")),
		mcp.WithString("object_id", mcp.Required(),
			mcp.Description("The ObjectID of the audio input to select")),
	)
	s.AddTool(selectAudioTool, h.SelectAudioHandler)

	getZoneInfoTool := mcp.NewTool(
		"get_zone_info",
		mcp.WithDescription("Get detailed information about a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get zone info from")),
	)
	s.AddTool(getZoneInfoTool, h.GetZoneInfoHandler)

	getUUIDTool := mcp.NewTool(
		"get_uuid",
		mcp.WithDescription("Get the UUID of a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get the UUID from")),
	)
	s.AddTool(getUUIDTool, h.GetUUIDHandler)

	switchToLineInTool := mcp.NewTool(
		"switch_to_line_in",
		mcp.WithDescription("Switch playback to the line-in input"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to switch to line-in in")),
	)
	s.AddTool(switchToLineInTool, h.SwitchToLineInHandler)

	switchToQueueTool := mcp.NewTool(
		"switch_to_queue",
		mcp.WithDescription("Switch playback to the queue"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to switch to the queue in")),
	)
	s.AddTool(switchToQueueTool, h.SwitchToQueueHandler)

	listGroupsTool := mcp.NewTool(
		"list_sonos_groups",
		mcp.WithDescription("List all Sonos zone groups on the network"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of a room in the group (coordinator or member)")),
	)
	s.AddTool(listGroupsTool, h.ListGroupsHandler)

	getZoneGroupAttributesTool := mcp.NewTool(
		"get_zone_group_attributes",
		mcp.WithDescription("Get the zone group attributes for a given room"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get zone group attributes for")),
	)
	s.AddTool(getZoneGroupAttributesTool, h.GetZoneGroupAttributesHandler)

	addGroupMemberTool := mcp.NewTool(
		"add_group_member",
		mcp.WithDescription("Add a member to a Sonos group"),
		mcp.WithString("coordinator_room_name", mcp.Required(),
			mcp.Description("The name of the coordinator room")),
		mcp.WithString("member_room_name", mcp.Required(),
			mcp.Description("The name of the member room to add")),
	)
	s.AddTool(addGroupMemberTool, h.AddGroupMemberHandler)

	removeGroupMemberTool := mcp.NewTool(
		"remove_group_member",
		mcp.WithDescription("Remove a member from a Sonos group"),
		mcp.WithString("coordinator_room_name", mcp.Required(),
			mcp.Description("The name of the coordinator room")),
		mcp.WithString("member_room_name", mcp.Required(),
			mcp.Description("The name of the member room to remove")),
	)
	s.AddTool(removeGroupMemberTool, h.RemoveGroupMemberHandler)

	getGroupVolumeTool := mcp.NewTool(
		"get_group_volume",
		mcp.WithDescription("Get the current volume of a Sonos group"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of a room in the group (coordinator or member)")),
	)
	s.AddTool(getGroupVolumeTool, h.GetGroupVolumeHandler)

	setGroupVolumeTool := mcp.NewTool(
		"set_group_volume",
		mcp.WithDescription("Set the volume of a Sonos group"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of a room in the group (coordinator or member)")),
	)
	s.AddTool(setGroupVolumeTool, h.SetGroupVolumeHandler)

	getMediaInfoTool := mcp.NewTool(
		"get_media_info",
		mcp.WithDescription("Get the current media information on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get media info from")),
	)
	s.AddTool(getMediaInfoTool, h.GetMediaInfoHandler)

	searchSpotifyTool := mcp.NewTool(
		"search_spotify",
		mcp.WithDescription("Search for a track, album, or artist on Spotify and returns its URI"),
		mcp.WithString("query", mcp.Required(),
			mcp.Description("The search query")),
		mcp.WithString("search_type", mcp.Required(),
			mcp.Description("The type of search to perform (track, album, playlist, or artist)")),
	)
	s.AddTool(searchSpotifyTool, h.SearchSpotifyHandler)

	playSpotifyURITool := mcp.NewTool(
		"play_spotify_uri",
		mcp.WithDescription("Play a Spotify URI on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to play the Spotify URI in")),
		mcp.WithString("track_info_json", mcp.Required(),
			mcp.Description("JSON string containing Spotify track, album or playlist information (URI, title, artist, album, album art URL)")),
	)
	s.AddTool(playSpotifyURITool, h.PlaySpotifyURIHandler)

	// Choose transport based on environment
	transport := os.Getenv("MCP_TRANSPORT")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	switch transport {
	case "http":
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(":" + port); err != nil {
			log.Fatalf("HTTP Server error: %v", err)
		}
	case "sse":
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(":" + port); err != nil {
			log.Fatalf("SSE Server error: %v", err)
		}
	default:
		// Start the stdio server. This allows communication over standard input/output.
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Stdio Server error: %v", err)
		}
	}
}
