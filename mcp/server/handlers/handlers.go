package handlers

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/caglar10ur/sonos"
	"github.com/caglar10ur/sonos/didl"
	"github.com/caglar10ur/sonos/mcp/server/sonoscontrol"
	"github.com/caglar10ur/sonos/mcp/server/spotify"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DIDLLite represents the root DIDL-Lite element
type DIDLLite struct {
	XMLName xml.Name  `xml:"DIDL-Lite"`
	Dc      string    `xml:"xmlns:dc,attr"`
	Upnp    string    `xml:"xmlns:upnp,attr"`
	R       string    `xml:"xmlns:r,attr"`
	Xmlns   string    `xml:"xmlns,attr"`
	Item    didl.Item `xml:"item"`
}

type RoomNameParams struct {
	RoomName string `json:"room_name"`
}

type SetVolumeParams struct {
	RoomName string `json:"room_name"`
	Volume   int    `json:"volume"`
}

type SetLineInLevelParams struct {
	RoomName                string `json:"room_name"`
	DesiredLeftLineInLevel  int    `json:"desired_left_line_in_level"`
	DesiredRightLineInLevel int    `json:"desired_right_line_in_level"`
}

type SelectAudioParams struct {
	RoomName string `json:"room_name"`
	ObjectID string `json:"object_id"`
}

type AddGroupMemberParams struct {
	CoordinatorRoomName string `json:"coordinator_room_name"`
	MemberRoomName      string `json:"member_room_name"`
}

type RemoveGroupMemberParams struct {
	CoordinatorRoomName string `json:"coordinator_room_name"`
	MemberRoomName      string `json:"member_room_name"`
}

type SeekParams struct {
	RoomName string `json:"room_name"`
	Position string `json:"position" jsonschema:"position in the current track, e.g. 1m30s or 0:01:30"`
}

type QueuePositionParams struct {
	RoomName string `json:"room_name"`
	Position int    `json:"position" jsonschema:"queue position, counting from 1"`
}

type AddToQueueParams struct {
	RoomName string `json:"room_name"`
	URI      string `json:"uri" jsonschema:"URI of the track, album or playlist"`
	Metadata string `json:"metadata,omitempty" jsonschema:"optional DIDL-Lite metadata"`
	Position int    `json:"position,omitempty" jsonschema:"queue position to insert at, counting from 1; 0 or omitted appends"`
}

type SetPlayModeParams struct {
	RoomName string `json:"room_name"`
	Shuffle  bool   `json:"shuffle"`
	Repeat   string `json:"repeat" jsonschema:"off, all or one"`
}

type SetGroupVolumeParams struct {
	RoomName string `json:"room_name"`
	Volume   int    `json:"volume"`
}

type SearchSpotifyParams struct {
	Query      string `json:"query"`
	SearchType string `json:"search_type"`
}

type PlaySpotifyURIParams struct {
	RoomName      string `json:"room_name"`
	TrackInfoJSON string `json:"track_info_json"`
}

type Handlers struct {
	sonosController sonoscontrol.SonosControllerInterface
	spotifyClient   spotify.SpotifyClientInterface
	searchTimeout   time.Duration
}

func NewHandlers(sonosController sonoscontrol.SonosControllerInterface, spotifyClient spotify.SpotifyClientInterface, searchTimeout time.Duration) *Handlers {
	return &Handlers{
		sonosController: sonosController,
		spotifyClient:   spotifyClient,
		searchTimeout:   searchTimeout,
	}
}

func (h *Handlers) RegisterTools(s *mcp.Server) {
	// 1. Generic Tools
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_sonos_devices",
		Description: "List all Sonos devices on the network",
	}, h.ListSonosDevicesHandler)

	// 2. Room Tools (RoomNameParams)
	roomTools := []struct {
		Name    string
		Desc    string
		Handler func(context.Context, *mcp.CallToolRequest, RoomNameParams) (*mcp.CallToolResult, any, error)
	}{
		{"get_now_playing", "Get the currently playing track on a Sonos device", h.GetNowPlayingHandler},
		{"play", "Start playback on a Sonos device", h.PlayHandler},
		{"stop", "Stop playback on a Sonos device", h.StopHandler},
		{"pause", "Pause playback on a Sonos device", h.PauseHandler},
		{"next", "Play the next track on a Sonos device", h.NextHandler},
		{"previous", "Play the previous track on a Sonos device", h.PreviousHandler},
		{"get_volume", "Get the current volume of a Sonos device", h.GetVolumeHandler},
		{"list_queue", "List the songs in the device's queue", h.ListQueueHandler},
		{"get_position_info", "Get the position information of the currently playing song", h.GetPositionInfoHandler},
		{"mute", "Mute a Sonos device", h.MuteHandler},
		{"unmute", "Unmute a Sonos device", h.UnmuteHandler},
		{"get_mute_status", "Get the mute status of a Sonos device", h.GetMuteStatusHandler},
		{"get_audio_input_attributes", "Get the name and icon of the audio input", h.GetAudioInputAttributesHandler},
		{"get_line_in_level", "Get the current left and right line-in levels", h.GetLineInLevelHandler},
		{"get_zone_info", "Get detailed information about a Sonos device", h.GetZoneInfoHandler},
		{"get_uuid", "Get the UUID of a Sonos device", h.GetUUIDHandler},
		{"switch_to_line_in", "Switch playback to the line-in input", h.SwitchToLineInHandler},
		{"switch_to_queue", "Switch playback to the queue", h.SwitchToQueueHandler},
		{"get_zone_group_attributes", "Get the zone group attributes for a given room", h.GetZoneGroupAttributesHandler},
		{"get_group_volume", "Get the current volume of a Sonos group", h.GetGroupVolumeHandler},
		{"get_media_info", "Get the current media information on a Sonos device", h.GetMediaInfoHandler},
		{"clear_queue", "Remove every track from the queue of a room's group", h.ClearQueueHandler},
		{"get_play_mode", "Get whether a room's group shuffles and what it repeats", h.GetPlayModeHandler},
	}

	for _, t := range roomTools {
		mcp.AddTool(s, &mcp.Tool{Name: t.Name, Description: t.Desc}, t.Handler)
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_sonos_groups",
		Description: "List all Sonos zone groups on the network",
	}, h.ListGroupsHandler)

	// 3. Tools with specific params
	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_volume",
		Description: "Set the volume of a Sonos device",
	}, h.SetVolumeHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_line_in_level",
		Description: "Set the left and right line-in levels",
	}, h.SetLineInLevelHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "select_audio",
		Description: "Select an audio input by its ObjectID",
	}, h.SelectAudioHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_group_member",
		Description: "Add a member to a Sonos group",
	}, h.AddGroupMemberHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "remove_group_member",
		Description: "Remove a member from a Sonos group",
	}, h.RemoveGroupMemberHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "seek",
		Description: "Move to a position in the current track of a room's group",
	}, h.SeekHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "play_queue_track",
		Description: "Play the track at a queue position in a room's group",
	}, h.PlayQueueTrackHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_to_queue",
		Description: "Add a track, album or playlist URI to the queue of a room's group",
	}, h.AddToQueueHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "remove_from_queue",
		Description: "Remove the track at a queue position from a room's group",
	}, h.RemoveFromQueueHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_play_mode",
		Description: "Set whether a room's group shuffles and what it repeats",
	}, h.SetPlayModeHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_group_volume",
		Description: "Set the volume of a Sonos group",
	}, h.SetGroupVolumeHandler)

	if h.spotifyClient != nil {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "search_spotify",
			Description: "Search for a track, album, or artist on Spotify and returns its URI",
		}, h.SearchSpotifyHandler)

		mcp.AddTool(s, &mcp.Tool{
			Name:        "play_spotify_uri",
			Description: "Play a Spotify URI on a Sonos device",
		}, h.PlaySpotifyURIHandler)
	}
}

// room looks a room up, bounding only the lookup: discovering a household
// that doesn't answer would otherwise run until the client gives up.
func (h *Handlers) room(ctx context.Context, roomName string) (*sonos.Room, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, h.searchTimeout)
	defer cancel()
	return h.sonosController.Room(lookupCtx, roomName)
}

// withRoom runs fn with a room, for room-level actions such as volume.
func (h *Handlers) withRoom(ctx context.Context, roomName string, fn func(*sonos.Room) (*mcp.CallToolResult, any, error)) (*mcp.CallToolResult, any, error) {
	r, err := h.room(ctx, roomName)
	if err != nil {
		return handleError(err, roomName), nil, nil
	}
	return fn(r)
}

// withPlayer runs fn with the player of a room, for UPnP calls the library
// doesn't wrap.
func (h *Handlers) withPlayer(ctx context.Context, roomName string, fn func(*sonos.ZonePlayer) (*mcp.CallToolResult, any, error)) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, roomName, func(r *sonos.Room) (*mcp.CallToolResult, any, error) {
		zp, err := r.Player(ctx)
		if err != nil {
			return handleError(err, roomName), nil, nil
		}
		return fn(zp)
	})
}

// withGroup runs fn with the group a room plays in, for group-level actions
// such as playback, the queue and group volume.
func (h *Handlers) withGroup(ctx context.Context, roomName string, fn func(*sonos.Group) (*mcp.CallToolResult, any, error)) (*mcp.CallToolResult, any, error) {
	r, err := h.room(ctx, roomName)
	if err != nil {
		return handleError(err, roomName), nil, nil
	}
	return fn(r.Group())
}

func handleError(err error, roomName string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("failed to perform action in room %s: %s", roomName, err)},
		},
		IsError: true,
	}
}

func handleGenericError(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("failed to perform action: %s", err)},
		},
		IsError: true,
	}
}

func SuccessResponse(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}
