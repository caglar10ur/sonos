package handlers

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/caglar10ur/sonos"
	"github.com/caglar10ur/sonos/didl"
	"github.com/caglar10ur/sonos/mcp-server/sonoscontrol"
	"github.com/caglar10ur/sonos/mcp-server/spotify"
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

func NewHandlers(sonosController sonoscontrol.SonosControllerInterface, spotifyClient spotify.SpotifyClientInterface, searchTimeout time.Duration) *Handlers {
	return &Handlers{
		sonosController: sonosController,
		spotifyClient:   spotifyClient,
		searchTimeout:   searchTimeout,
	}
}

func (h *Handlers) withRoom(ctx context.Context, roomName string, fn func(*sonos.ZonePlayer) (*mcp.CallToolResult, any, error)) (*mcp.CallToolResult, any, error) {
	zp, err := h.sonosController.CachedRoom(ctx, roomName)
	if err != nil {
		return handleError(err, roomName), nil, nil
	}

	return fn(zp)
}
