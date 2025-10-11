package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/caglar10ur/sonos"
	"github.com/caglar10ur/sonos/didl"
	avt "github.com/caglar10ur/sonos/services/AVTransport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

type SpotifyTrackInfo struct {
	URI      string `json:"uri"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	AlbumArt string `json:"album_art_url"`
}

const (
	defaultTimeout = 10 * time.Second
)

var (
	cache sync.Map
	son   *sonos.Sonos
)

func cachedRoom(ctx context.Context, roomName string) (*sonos.ZonePlayer, error) {
	if zp, ok := cache.Load(roomName); ok {
		return zp.(*sonos.ZonePlayer), nil
	}

	zp, err := son.FindRoom(ctx, roomName)
	if err != nil {
		return nil, err
	}

	cache.Store(roomName, zp)
	return zp, nil
}

func main() {
	var err error

	// Create a new MCP server
	s := server.NewMCPServer(
		"Sonos",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	son, err = sonos.NewSonos()
	if err != nil {
		log.Fatalf("Creating a sonos client failed with %w", err)
	}
	defer son.Close()

	listTool := mcp.NewTool(
		"list_sonos_devices",
		mcp.WithDescription("List all Sonos devices on the network"),
	)
	s.AddTool(listTool, listSonosDevicesHandler)

	nowPlayingTool := mcp.NewTool(
		"get_now_playing",
		mcp.WithDescription("Get the currently playing track on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get media info from")),
	)
	s.AddTool(nowPlayingTool, getNowPlayingHandler)

	playTool := mcp.NewTool(
		"play",
		mcp.WithDescription("Start playback on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to start playback in")),
	)
	s.AddTool(playTool, playHandler)

	stopTool := mcp.NewTool(
		"stop",
		mcp.WithDescription("Stop playback on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to stop playback in")),
	)
	s.AddTool(stopTool, stopHandler)

	pauseTool := mcp.NewTool(
		"pause",
		mcp.WithDescription("Pause playback on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to pause playback in")),
	)
	s.AddTool(pauseTool, pauseHandler)

	nextTool := mcp.NewTool(
		"next",
		mcp.WithDescription("Play the next track on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to play the next track in")),
	)
	s.AddTool(nextTool, nextHandler)

	previousTool := mcp.NewTool(
		"previous",
		mcp.WithDescription("Play the previous track on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to play the previous track in")),
	)
	s.AddTool(previousTool, previousHandler)

	getVolumeTool := mcp.NewTool(
		"get_volume",
		mcp.WithDescription("Get the current volume of a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get the volume from")),
	)
	s.AddTool(getVolumeTool, getVolumeHandler)

	setVolumeTool := mcp.NewTool(
		"set_volume",
		mcp.WithDescription("Set the volume of a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to set the volume for")),
		mcp.WithNumber("volume", mcp.Required(),
			mcp.Description("The volume level to set (0-100)")),
	)
	s.AddTool(setVolumeTool, setVolumeHandler)

	listQueueTool := mcp.NewTool(
		"list_queue",
		mcp.WithDescription("List the songs in the device's queue"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to list the queue for")),
	)
	s.AddTool(listQueueTool, listQueueHandler)

	getPositionInfoTool := mcp.NewTool(
		"get_position_info",
		mcp.WithDescription("Get the position information of the currently playing song"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get position info from")),
	)
	s.AddTool(getPositionInfoTool, getPositionInfoHandler)

	muteTool := mcp.NewTool(
		"mute",
		mcp.WithDescription("Mute a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to mute")),
	)
	s.AddTool(muteTool, muteHandler)

	unmuteTool := mcp.NewTool(
		"unmute",
		mcp.WithDescription("Unmute a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to unmute")),
	)
	s.AddTool(unmuteTool, unmuteHandler)

	getMuteStatusTool := mcp.NewTool(
		"get_mute_status",
		mcp.WithDescription("Get the mute status of a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get the mute status from")),
	)
	s.AddTool(getMuteStatusTool, getMuteStatusHandler)

	getAudioInputAttributesTool := mcp.NewTool(
		"get_audio_input_attributes",
		mcp.WithDescription("Get the name and icon of the audio input"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get audio input attributes from")),
	)
	s.AddTool(getAudioInputAttributesTool, getAudioInputAttributesHandler)

	getLineInLevelTool := mcp.NewTool(
		"get_line_in_level",
		mcp.WithDescription("Get the current left and right line-in levels"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get line-in levels from")),
	)
	s.AddTool(getLineInLevelTool, getLineInLevelHandler)

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
	s.AddTool(setLineInLevelTool, setLineInLevelHandler)

	selectAudioTool := mcp.NewTool(
		"select_audio",
		mcp.WithDescription("Select an audio input by its ObjectID"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to select audio for")),
		mcp.WithString("object_id", mcp.Required(),
			mcp.Description("The ObjectID of the audio input to select")),
	)
	s.AddTool(selectAudioTool, selectAudioHandler)

	getZoneInfoTool := mcp.NewTool(
		"get_zone_info",
		mcp.WithDescription("Get detailed information about a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get zone info from")),
	)
	s.AddTool(getZoneInfoTool, getZoneInfoHandler)

	getUUIDTool := mcp.NewTool(
		"get_uuid",
		mcp.WithDescription("Get the UUID of a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get the UUID from")),
	)
	s.AddTool(getUUIDTool, getUUIDHandler)

	switchToLineInTool := mcp.NewTool(
		"switch_to_line_in",
		mcp.WithDescription("Switch playback to the line-in input"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to switch to line-in in")),
	)
	s.AddTool(switchToLineInTool, switchToLineInHandler)

	switchToQueueTool := mcp.NewTool(
		"switch_to_queue",
		mcp.WithDescription("Switch playback to the queue"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to switch to the queue in")),
	)
	s.AddTool(switchToQueueTool, switchToQueueHandler)

	listGroupsTool := mcp.NewTool(
		"list_sonos_groups",
		mcp.WithDescription("List all Sonos zone groups on the network"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of a room in the group (coordinator or member)")),
	)
	s.AddTool(listGroupsTool, listGroupsHandler)

	getZoneGroupAttributesTool := mcp.NewTool(
		"get_zone_group_attributes",
		mcp.WithDescription("Get the zone group attributes for a given room"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get zone group attributes for")),
	)
	s.AddTool(getZoneGroupAttributesTool, getZoneGroupAttributesHandler)

	addGroupMemberTool := mcp.NewTool(
		"add_group_member",
		mcp.WithDescription("Add a member to a Sonos group"),
		mcp.WithString("coordinator_room_name", mcp.Required(),
			mcp.Description("The name of the coordinator room")),
		mcp.WithString("member_room_name", mcp.Required(),
			mcp.Description("The name of the member room to add")),
	)
	s.AddTool(addGroupMemberTool, addGroupMemberHandler)

	removeGroupMemberTool := mcp.NewTool(
		"remove_group_member",
		mcp.WithDescription("Remove a member from a Sonos group"),
		mcp.WithString("coordinator_room_name", mcp.Required(),
			mcp.Description("The name of the coordinator room")),
		mcp.WithString("member_room_name", mcp.Required(),
			mcp.Description("The name of the member room to remove")),
	)
	s.AddTool(removeGroupMemberTool, removeGroupMemberHandler)

	getGroupVolumeTool := mcp.NewTool(
		"get_group_volume",
		mcp.WithDescription("Get the current volume of a Sonos group"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of a room in the group (coordinator or member)")),
	)
	s.AddTool(getGroupVolumeTool, getGroupVolumeHandler)

	setGroupVolumeTool := mcp.NewTool(
		"set_group_volume",
		mcp.WithDescription("Set the volume of a Sonos group"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of a room in the group (coordinator or member)")),
	)
	s.AddTool(setGroupVolumeTool, setGroupVolumeHandler)

	getMediaInfoTool := mcp.NewTool(
		"get_media_info",
		mcp.WithDescription("Get the current media information on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to get media info from")),
	)
	s.AddTool(getMediaInfoTool, getMediaInfoHandler)

	searchSpotifyTool := mcp.NewTool(
		"search_spotify",
		mcp.WithDescription("Search for a track, album, or artist on Spotify and returns its URI"),
		mcp.WithString("query", mcp.Required(),
			mcp.Description("The search query")),
		mcp.WithString("search_type", mcp.Required(),
			mcp.Description("The type of search to perform (track, album, playlist, or artist)")),
	)
	s.AddTool(searchSpotifyTool, searchSpotifyHandler)

	playSpotifyURITool := mcp.NewTool(
		"play_spotify_uri",
		mcp.WithDescription("Play a Spotify URI on a Sonos device"),
		mcp.WithString("room_name", mcp.Required(),
			mcp.Description("The name of the room to play the Spotify URI in")),
		mcp.WithString("track_info_json", mcp.Required(),
			mcp.Description("JSON string containing Spotify track, album or playlist information (URI, title, artist, album, album art URL)")),
	)
	s.AddTool(playSpotifyURITool, playSpotifyURIHandler)

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

func getMediaInfoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	mediaInfo, err := zp.AVTransport.GetMediaInfo(&avt.GetMediaInfoArgs{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get media info for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("mediaInfo: %#v", mediaInfo)), nil
}

func addGroupMemberHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	coordinatorRoomName, err := request.RequireString("coordinator_room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	memberRoomName, err := request.RequireString("member_room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	coordinatorZp, err := cachedRoom(ctx, coordinatorRoomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("coordinator room not found: %s", err)), nil
	}

	memberZp, err := cachedRoom(ctx, memberRoomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("member room not found: %s", err)), nil
	}

	_, err = memberZp.AVTransport.SetAVTransportURI(&avt.SetAVTransportURIArgs{
		InstanceID: 0,
		CurrentURI: fmt.Sprintf("x-rincon:%s", coordinatorZp.UUID()),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add member %s to group %s: %s", memberRoomName, coordinatorRoomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Member %s added to group %s", memberRoomName, coordinatorRoomName)), nil
}

func removeGroupMemberHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	coordinatorRoomName, err := request.RequireString("coordinator_room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	memberRoomName, err := request.RequireString("member_room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	coordinatorZp, err := cachedRoom(ctx, coordinatorRoomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("coordinator room not found: %s", err)), nil
	}

	zoneGroupState, err := coordinatorZp.GetZoneGroupState()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting zone group state for %s: %v", coordinatorRoomName, err)), nil
	}

	var location string
	for _, group := range zoneGroupState.ZoneGroups {
		if coordinatorZp.UUID() == group.Coordinator {
			for _, member := range group.ZoneGroupMember {
				if member.ZoneName == memberRoomName {
					location = member.Location
					break
				}
			}
		}
		if location != "" {
			break
		}
	}

	if location == "" {
		return mcp.NewToolResultError(fmt.Sprintf("Member room %s not found in group %s", memberRoomName, coordinatorRoomName)), nil
	}

	u, err := sonos.FromLocation(location)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return mcp.NewToolResultError(fmt.Sprintf("Member room location %s parsing failed", location)), nil
	}

	zp, err := sonos.NewZonePlayer(sonos.WithLocation(u))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("member room not found: %s", err)), nil
	}

	_, err = zp.AVTransport.BecomeCoordinatorOfStandaloneGroup(&avt.BecomeCoordinatorOfStandaloneGroupArgs{
		InstanceID: 0,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove member %s from group %s: %s", memberRoomName, coordinatorRoomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Member %s removed from group %s", memberRoomName, coordinatorRoomName)), nil
}

func listGroupsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	var groupInfo strings.Builder

	zoneGroupState, err := zp.GetZoneGroupState()
	if err != nil {
		groupInfo.WriteString(fmt.Sprintf("Error getting zone group state for %s: %v\n", zp.RoomName(), err))
		return mcp.NewToolResultText(groupInfo.String()), nil
	}

	for _, group := range zoneGroupState.ZoneGroups {
		if zp.UUID() == group.Coordinator {
			groupInfo.WriteString(fmt.Sprintf("  Group ID: %s (Coordinator: %s)\n", group.ID, group.Coordinator))
			for _, member := range group.ZoneGroupMember {
				groupInfo.WriteString(fmt.Sprintf("    - Member: %s (UUID: %s)\n", member.ZoneName, member.UUID))
			}
		}
	}
	return mcp.NewToolResultText(groupInfo.String()), nil
}

func getZoneGroupAttributesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	attrs, err := zp.GetZoneGroupAttributes()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get zone group attributes for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Zone Group Name: %s, Zone Group ID: %s, Zone Player UUIDs in Group: %s, Muse Household ID: %s", attrs.CurrentZoneGroupName, attrs.CurrentZoneGroupID, attrs.CurrentZonePlayerUUIDsInGroup, attrs.CurrentMuseHouseholdId)), nil
}

// listSonosDevicesHandler is the function that executes when the "list_sonos_devices" tool is called.
func listSonosDevicesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var devices []string

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	son.Search(ctx, func(s *sonos.Sonos, zp *sonos.ZonePlayer) {
		devices = append(devices, zp.RoomName())
		cache.Store(zp.RoomName(), zp)
	})

	<-ctx.Done()

	return mcp.NewToolResultText(fmt.Sprintf("%v", devices)), nil
}

func getNowPlayingHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	info, err := zp.AVTransport.GetPositionInfo(&avt.GetPositionInfoArgs{
		InstanceID: 0,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get position info: %s", err)), nil
	}

	var lite didl.Lite
	if err := xml.Unmarshal([]byte(info.TrackMetaData), &lite); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to unmarshal track metadata: %s", err)), nil
	}

	if len(lite.Item) == 0 {
		return mcp.NewToolResultText("Nothing is playing"), nil
	}

	item := lite.Item[0]
	var title, artist, album string
	if len(item.Title) > 0 {
		title = item.Title[0].Value
	}
	if len(item.Creator) > 0 {
		artist = item.Creator[0].Value
	}
	if len(item.Album) > 0 {
		album = item.Album[0].Value
	}

	return mcp.NewToolResultText(fmt.Sprintf("Title: %s, Artist: %s, Album: %s", title, artist, album)), nil
}

func playHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.Play()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to play in room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Playback started in %s", roomName)), nil
}

func stopHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.Stop()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stop in room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Playback stopped in %s", roomName)), nil
}

func pauseHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.Pause()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to pause in room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Playback paused in %s", roomName)), nil
}

func nextHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.Next()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to play next track in room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Playing next track in %s", roomName)), nil
}

func previousHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.Previous()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to play previous track in room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Playing previous track in %s", roomName)), nil
}

func getVolumeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	volume, err := zp.GetVolume()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get volume for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Current volume in %s is %d", roomName, volume)), nil
}

func setVolumeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	volume, err := request.RequireInt("volume")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.SetVolume(volume)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to set volume for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Volume in %s set to %d", roomName, volume)), nil
}

func listQueueHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	queueItems, err := zp.ListQueue()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list queue for room %s: %s", roomName, err)), nil
	}

	if len(queueItems) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("Queue in %s is empty", roomName)), nil
	}

	var queueList string
	for i, item := range queueItems {
		var title, artist, album string
		if len(item.Title) > 0 {
			title = item.Title[0].Value
		}
		if len(item.Creator) > 0 {
			artist = item.Creator[0].Value
		}
		if len(item.Album) > 0 {
			album = item.Album[0].Value
		}

		queueList += fmt.Sprintf("%d. Title: %s, Artist: %s, Album: %s\n", i+1, title, artist, album)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Queue in %s:\n%s", roomName, queueList)), nil
}

func getPositionInfoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	info, err := zp.GetPositionInfo()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get position info for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Track: %d, Duration: %s, Elapsed: %s", info.Track, info.TrackDuration, info.RelTime)), nil
}

func muteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.Mute()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Room %s already muted or failed to mute: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Room %s muted", roomName)), nil
}

func unmuteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.Unmute()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Room %s already unmuted or failed to unmute: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Room %s unmuted", roomName)), nil
}

func getMuteStatusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	isMuted, err := zp.IsMuted()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get mute status for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Room %s mute status: %t", roomName, isMuted)), nil
}

func getAudioInputAttributesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	attrs, err := zp.GetAudioInputAttributes()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get audio input attributes for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Audio Input Name: %s, Icon: %s", attrs.CurrentName, attrs.CurrentIcon)), nil
}

func getLineInLevelHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	levels, err := zp.GetLineInLevel()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get line-in levels for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Line-in levels for room %s: Left: %d, Right: %d", roomName, levels.CurrentLeftLineInLevel, levels.CurrentRightLineInLevel)), nil
}

func setLineInLevelHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	desiredLeftLineInLevel, err := request.RequireInt("desired_left_line_in_level")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	desiredRightLineInLevel, err := request.RequireInt("desired_right_line_in_level")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.SetLineInLevel(int32(desiredLeftLineInLevel), int32(desiredRightLineInLevel))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to set line-in levels for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Line-in levels for room %s set to Left: %d, Right: %d", roomName, desiredLeftLineInLevel, desiredRightLineInLevel)), nil
}

func selectAudioHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	objectID, err := request.RequireString("object_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.SelectAudio(objectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to select audio for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Audio input for room %s selected to %s", roomName, objectID)), nil
}

func getZoneInfoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	info, err := zp.GetZoneInfo()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get zone info for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Serial Number: %s, Software Version: %s, IP Address: %s, MAC Address: %s", info.SerialNumber, info.SoftwareVersion, info.IPAddress, info.MACAddress)), nil
}

func getUUIDHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	uuid := zp.UUID()
	if uuid == "" {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get UUID for room %s", roomName)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("UUID for room %s: %s", roomName, uuid)), nil
}

func switchToLineInHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.SwitchToLineIn()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to switch to line-in for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Switched to line-in in %s", roomName)), nil
}

func switchToQueueHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.SwitchToQueue()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to switch to queue for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Switched to queue in %s", roomName)), nil
}

func getGroupVolumeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	volume, err := zp.GetGroupVolume()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get group volume for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Current group volume in %s is %d", roomName, volume)), nil
}

func setGroupVolumeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	volume, err := request.RequireInt("volume")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	err = zp.SetGroupVolume(volume)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to set group volume for room %s: %s", roomName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Group volume in %s set to %d", roomName, volume)), nil
}

func searchSpotifyHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	searchType, err := request.RequireString("search_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("couldn't get token: %v", err)), nil
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotify.New(httpClient)

	var searchRequestType spotify.SearchType
	switch searchType {
	case "track":
		searchRequestType = spotify.SearchTypeTrack
	case "album":
		searchRequestType = spotify.SearchTypeAlbum
	case "artist":
		searchRequestType = spotify.SearchTypeArtist
	case "playlist":
		searchRequestType = spotify.SearchTypePlaylist
	default:
		return mcp.NewToolResultError("invalid search type"), nil
	}

	results, err := client.Search(ctx, query, searchRequestType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("error searching spotify: %s", err)), nil
	}

	var result string

	switch searchType {
	case "track":
		if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
			track := results.Tracks.Tracks[0]
			var albumArtURL string
			if len(track.Album.Images) > 0 {
				albumArtURL = track.Album.Images[0].URL
			}
			trackInfo := SpotifyTrackInfo{
				URI:      string(track.URI),
				Title:    track.Name,
				Artist:   track.Artists[0].Name,
				Album:    track.Album.Name,
				AlbumArt: albumArtURL,
			}
			jsonBytes, err := json.Marshal(trackInfo)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to marshal track info: %v", err)), nil
			}
			result = string(jsonBytes)
		}
	case "album":
		if results.Albums != nil && len(results.Albums.Albums) > 0 {
			result = string(results.Albums.Albums[0].URI)
		}
	case "artist":
		if results.Artists != nil && len(results.Artists.Artists) > 0 {
			result = string(results.Artists.Artists[0].URI)
		}
	case "playlist":
		if results.Playlists != nil && len(results.Playlists.Playlists) > 0 {
			result = string(results.Playlists.Playlists[0].URI)
		}
	}

	if result == "" {
		return mcp.NewToolResultError("no results found"), nil
	}

	return mcp.NewToolResultText(result), nil
}

// DIDLLite represents the root DIDL-Lite element
type DIDLLite struct {
	XMLName xml.Name  `xml:"DIDL-Lite"`
	Dc      string    `xml:"xmlns:dc,attr"`
	Upnp    string    `xml:"xmlns:upnp,attr"`
	R       string    `xml:"xmlns:r,attr"`
	Xmlns   string    `xml:"xmlns,attr"`
	Item    didl.Item `xml:"item"`
}

func playSpotifyURIHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	roomName, err := request.RequireString("room_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	trackInfoJSON, err := request.RequireString("track_info_json")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var trackInfo SpotifyTrackInfo
	err = json.Unmarshal([]byte(trackInfoJSON), &trackInfo)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to unmarshal track info: %v", err)), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := cachedRoom(ctx, roomName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("room not found: %s", err)), nil
	}

	// Parse Spotify URI
	parts := strings.Split(trackInfo.URI, ":")
	if len(parts) != 3 || parts[0] != "spotify" {
		return mcp.NewToolResultError("invalid Spotify URI format"), nil
	}

	typeStr := parts[1]
	id := parts[2]

	var resURI string
	var itemid string

	switch typeStr {
	case "track":
		resURI = fmt.Sprintf("x-sonos-spotify:spotify%%3atrack%%3a%s?sid=12&flags=0&sn=2", id)
		itemid = fmt.Sprintf("00032020%s", url.QueryEscape(trackInfo.URI))
	case "album":
		resURI = fmt.Sprintf("x-rincon-cpcontainer:1004206cspotify%%3aalbum%%3a%s?sid=12&flags=0&sn=2", id)
		itemid = fmt.Sprintf("1004206cspotify%3aalbum%3a%s", id)
	case "artist":
		// TODO
		resURI = fmt.Sprintf("x-rincon-cpcontainer:1005206cspotify%%3aartist%%3a%s?sid=12&flags=0&sn=2", id)
	case "playlist":
		// TODO
		resURI = fmt.Sprintf("x-rincon-cpcontainer:1006206cspotify:playlist:id?sid=12&flags=0&sn=2", id)
		//magic = "1006206"
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unsupported Spotify URI type: %s", typeStr)), nil
	}

	// Populate the structs with data
	didl := DIDLLite{
		Dc:    "http://purl.org/dc/elements/1.1/",
		Upnp:  "urn:schemas-upnp-org:metadata-1-0/upnp/",
		R:     "urn:schemas-rinconnetworks-com:metadata-1-0/",
		Xmlns: "urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/",
		Item: didl.Item{
			ID:          itemid,
			Restricted:  true,
			Class:       []didl.Class{{Value: "object.item.audioItem.musicTrack"}},
			Title:       []didl.Title{{Value: trackInfo.Title}},
			Creator:     []didl.Creator{{Value: trackInfo.Artist}},
			Album:       []didl.Album{{Value: trackInfo.Album}},
			AlbumArtURI: []didl.AlbumArtURI{{Value: trackInfo.AlbumArt}},
			AlbumArtist: []didl.AlbumArtist{{Value: trackInfo.Artist}},
			Desc: []didl.Desc{{
				ID:        "cdudn",
				NameSpace: "urn:schemas-rinconnetworks-com:metadata-1-0/",
				Value:     "SA_RINCON3079_X_#Svc3079-0-Token",
			}},
		},
	}

	// Marshal the struct to XML
	output, err := xml.Marshal(didl)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshall xml Spotify URI type: %s", typeStr)), nil
	}

	if false {
		return mcp.NewToolResultError(fmt.Sprintf("this is what I generated: %s", string(output))), nil
	}
	_, err = zp.AVTransport.AddURIToQueue(&avt.AddURIToQueueArgs{
		InstanceID:                      0,
		EnqueuedURI:                     resURI,
		EnqueuedURIMetaData:             string(output),
		DesiredFirstTrackNumberEnqueued: 1,
		EnqueueAsNext:                   true,
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add Spotify URI %s to queue for room %s: %s", resURI, roomName, err)), nil
	}

	err = zp.Play()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to play Spotify URI %s in room %s: %s", resURI, roomName, err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Playing Spotify URI %s in %s", trackInfo.URI, roomName)), nil
}
