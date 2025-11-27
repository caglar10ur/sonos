package handlers

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caglar10ur/sonos"
	"github.com/caglar10ur/sonos/didl"
	"github.com/caglar10ur/sonos/mcp-server/sonoscontrol"
	"github.com/caglar10ur/sonos/mcp-server/spotify"
	avt "github.com/caglar10ur/sonos/services/AVTransport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	spot "github.com/zmb3/spotify/v2"
)

const (
	defaultTimeout = 10 * time.Second
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
	sonosController *sonoscontrol.SonosController
	spotifyClient   *spotify.SpotifyClient
}

func NewHandlers(sonosController *sonoscontrol.SonosController, spotifyClient *spotify.SpotifyClient) *Handlers {
	return &Handlers{
		sonosController: sonosController,
		spotifyClient:   spotifyClient,
	}
}

func (h *Handlers) withRoom(roomName string, fn func(*sonos.ZonePlayer) (*mcp.CallToolResult, any, error)) (*mcp.CallToolResult, any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zp, err := h.sonosController.CachedRoom(ctx, roomName)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("room not found: %s", err)},
			},
		}, nil, nil
	}

	return fn(zp)
}

func (h *Handlers) GetMediaInfoHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		mediaInfo, err := zp.AVTransport.GetMediaInfo(&avt.GetMediaInfoArgs{})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get media info for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("mediaInfo: %#v", mediaInfo)},
			},
		}, nil, nil
	})
}

func (h *Handlers) AddGroupMemberHandler(ctx context.Context, req *mcp.CallToolRequest, params AddGroupMemberParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.CoordinatorRoomName, func(coordinatorZp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		return h.withRoom(params.MemberRoomName, func(memberZp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
			_, err := memberZp.AVTransport.SetAVTransportURI(&avt.SetAVTransportURIArgs{
				InstanceID: 0,
				CurrentURI: fmt.Sprintf("x-rincon:%s", coordinatorZp.UUID()),
			})
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("failed to add member %s to group %s: %s", params.MemberRoomName, params.CoordinatorRoomName, err)},
					},
				}, nil, nil
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Member %s added to group %s", params.MemberRoomName, params.CoordinatorRoomName)},
				},
			}, nil, nil
		})
	})
}

func (h *Handlers) RemoveGroupMemberHandler(ctx context.Context, req *mcp.CallToolRequest, params RemoveGroupMemberParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.CoordinatorRoomName, func(coordinatorZp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		zoneGroupState, err := coordinatorZp.GetZoneGroupState()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error getting zone group state for %s: %v", params.CoordinatorRoomName, err)},
				},
			}, nil, nil
		}

		var location string
		for _, group := range zoneGroupState.ZoneGroups {
			if coordinatorZp.UUID() == group.Coordinator {
				for _, member := range group.ZoneGroupMember {
					if member.ZoneName == params.MemberRoomName {
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
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Member room %s not found in group %s", params.MemberRoomName, params.CoordinatorRoomName)},
				},
			}, nil, nil
		}

		u, err := sonos.FromLocation(location)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Member room location %s parsing failed", location)},
				},
			}, nil, nil
		}

		zp, err := sonos.NewZonePlayer(sonos.WithLocation(u))
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("member room not found: %s", err)},
				},
			}, nil, nil
		}

		_, err = zp.AVTransport.BecomeCoordinatorOfStandaloneGroup(&avt.BecomeCoordinatorOfStandaloneGroupArgs{
			InstanceID: 0,
		})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to remove member %s from group %s: %s", params.MemberRoomName, params.CoordinatorRoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Member %s removed from group %s", params.MemberRoomName, params.CoordinatorRoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) ListGroupsHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		var groupInfo strings.Builder

		zoneGroupState, err := zp.GetZoneGroupState()
		if err != nil {
			groupInfo.WriteString(fmt.Sprintf("Error getting zone group state for %s: %v\n", zp.RoomName(), err))
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: groupInfo.String()},
				},
			}, nil, nil
		}

		for _, group := range zoneGroupState.ZoneGroups {
			if zp.UUID() == group.Coordinator {
				groupInfo.WriteString(fmt.Sprintf("  Group ID: %s (Coordinator: %s)\n", group.ID, group.Coordinator))
				for _, member := range group.ZoneGroupMember {
					groupInfo.WriteString(fmt.Sprintf("    - Member: %s (UUID: %s)\n", member.ZoneName, member.UUID))
				}
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: groupInfo.String()},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetZoneGroupAttributesHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		attrs, err := zp.GetZoneGroupAttributes()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get zone group attributes for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Zone Group Name: %s, Zone Group ID: %s, Zone Player UUIDs in Group: %s, Muse Household ID: %s", attrs.CurrentZoneGroupName, attrs.CurrentZoneGroupID, attrs.CurrentZonePlayerUUIDsInGroup, attrs.CurrentMuseHouseholdId)},
			},
		}, nil, nil
	})
}

func (h *Handlers) ListSonosDevicesHandler(ctx context.Context, req *mcp.CallToolRequest, params any) (*mcp.CallToolResult, any, error) {
	devices, err := h.sonosController.ListSonosDevices(ctx)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to list sonos devices: %s", err)},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("%v", devices)},
		},
	}, nil, nil
}

func (h *Handlers) GetNowPlayingHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		info, err := zp.AVTransport.GetPositionInfo(&avt.GetPositionInfoArgs{
			InstanceID: 0,
		})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get position info: %s", err)},
				},
			}, nil, nil
		}

		var lite didl.Lite
		if err := xml.Unmarshal([]byte(info.TrackMetaData), &lite); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to unmarshal track metadata: %s", err)},
				},
			}, nil, nil
		}

		if len(lite.Item) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Nothing is playing"},
				},
			}, nil, nil
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

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Title: %s, Artist: %s, Album: %s", title, artist, album)},
			},
		}, nil, nil
	})
}

func (h *Handlers) PlayHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Play()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to play in room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playback started in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) StopHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Stop()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to stop in room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playback stopped in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) PauseHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Pause()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to pause in room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playback paused in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) NextHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Next()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to play next track in room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playing next track in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) PreviousHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Previous()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to play previous track in room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playing previous track in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		volume, err := zp.GetVolume()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get volume for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Current volume in %s is %d", params.RoomName, volume)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SetVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params SetVolumeParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SetVolume(params.Volume)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to set volume for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Volume in %s set to %d", params.RoomName, params.Volume)},
			},
		}, nil, nil
	})
}

func (h *Handlers) ListQueueHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		queueItems, err := zp.ListQueue()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to list queue for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		if len(queueItems) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Queue in %s is empty", params.RoomName)},
				},
			}, nil, nil
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

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Queue in %s:\n%s", params.RoomName, queueList)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetPositionInfoHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		info, err := zp.GetPositionInfo()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get position info for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Track: %d, Duration: %s, Elapsed: %s", info.Track, info.TrackDuration, info.RelTime)},
			},
		}, nil, nil
	})
}

func (h *Handlers) MuteHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Mute()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Room %s already muted or failed to mute: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Room %s muted", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) UnmuteHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Unmute()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Room %s already unmuted or failed to unmute: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Room %s unmuted", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetMuteStatusHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		isMuted, err := zp.IsMuted()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get mute status for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Room %s mute status: %t", params.RoomName, isMuted)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetAudioInputAttributesHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		attrs, err := zp.GetAudioInputAttributes()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get audio input attributes for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Audio Input Name: %s, Icon: %s", attrs.CurrentName, attrs.CurrentIcon)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetLineInLevelHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		levels, err := zp.GetLineInLevel()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get line-in levels for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Line-in levels for room %s: Left: %d, Right: %d", params.RoomName, levels.CurrentLeftLineInLevel, levels.CurrentRightLineInLevel)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SetLineInLevelHandler(ctx context.Context, req *mcp.CallToolRequest, params SetLineInLevelParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SetLineInLevel(int32(params.DesiredLeftLineInLevel), int32(params.DesiredRightLineInLevel))
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to set line-in levels for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Line-in levels for room %s set to Left: %d, Right: %d", params.RoomName, params.DesiredLeftLineInLevel, params.DesiredRightLineInLevel)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SelectAudioHandler(ctx context.Context, req *mcp.CallToolRequest, params SelectAudioParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SelectAudio(params.ObjectID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to select audio for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Audio input for room %s selected to %s", params.RoomName, params.ObjectID)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetZoneInfoHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		info, err := zp.GetZoneInfo()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get zone info for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Serial Number: %s, Software Version: %s, IP Address: %s, MAC Address: %s", info.SerialNumber, info.SoftwareVersion, info.IPAddress, info.MACAddress)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetUUIDHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		uuid := zp.UUID()
		if uuid == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get UUID for room %s", params.RoomName)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("UUID for room %s: %s", params.RoomName, uuid)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SwitchToLineInHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SwitchToLineIn()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to switch to line-in for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Switched to line-in in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SwitchToQueueHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SwitchToQueue()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to switch to queue for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Switched to queue in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetGroupVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		volume, err := zp.GetGroupVolume()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to get group volume for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Current group volume in %s is %d", params.RoomName, volume)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SetGroupVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params SetGroupVolumeParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SetGroupVolume(params.Volume)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to set group volume for room %s: %s", params.RoomName, err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Group volume in %s set to %d", params.RoomName, params.Volume)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SearchSpotifyHandler(ctx context.Context, req *mcp.CallToolRequest, params SearchSpotifyParams) (*mcp.CallToolResult, any, error) {
	var searchRequestType spot.SearchType
	switch params.SearchType {
	case "track":
		searchRequestType = spot.SearchTypeTrack
	case "album":
		searchRequestType = spot.SearchTypeAlbum
	case "artist":
		searchRequestType = spot.SearchTypeArtist
	case "playlist":
		searchRequestType = spot.SearchTypePlaylist
	default:
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "invalid search type"},
			},
		}, nil, nil
	}

	results, err := h.spotifyClient.Search(ctx, params.Query, searchRequestType)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("error searching spotify: %s", err)},
			},
		}, nil, nil
	}

	var result string

	switch params.SearchType {
	case "track":
		if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
			track := results.Tracks.Tracks[0]
			var albumArtURL string
			if len(track.Album.Images) > 0 {
				albumArtURL = track.Album.Images[0].URL
			}
			trackInfo := spotify.SpotifyTrackInfo{
				URI:      string(track.URI),
				Title:    track.Name,
				Artist:   track.Artists[0].Name,
				Album:    track.Album.Name,
				AlbumArt: albumArtURL,
			}
			jsonBytes, err := json.Marshal(trackInfo)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("failed to marshal track info: %v", err)},
					},
				}, nil, nil
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
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "no results found"},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

func (h *Handlers) PlaySpotifyURIHandler(ctx context.Context, req *mcp.CallToolRequest, params PlaySpotifyURIParams) (*mcp.CallToolResult, any, error) {
	var trackInfo spotify.SpotifyTrackInfo
	err := json.Unmarshal([]byte(params.TrackInfoJSON), &trackInfo)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to unmarshal track info: %v", err)},
			},
		}, nil, nil
	}

	return h.withRoom(params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		// Parse Spotify URI
		parts := strings.Split(trackInfo.URI, ":")
		if len(parts) != 3 || parts[0] != "spotify" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "invalid Spotify URI format"},
				},
			}, nil, nil
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
			itemid = fmt.Sprintf("1004206cspotify%%3aalbum%%3a%s", id)
		case "artist":
			// TODO
			resURI = fmt.Sprintf("x-rincon-cpcontainer:1005206cspotify%%3aartist%%3a%s?sid=12&flags=0&sn=2", id)
		case "playlist":
			// TODO
			resURI = fmt.Sprintf("x-rincon-cpcontainer:1006206cspotify:playlist:id?sid=12&flags=0&sn=2", id)
			//magic = "1006206"
		default:
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("unsupported Spotify URI type: %s", typeStr)},
				},
			}, nil, nil
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
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to marshall xml Spotify URI type: %s", typeStr)},
				},
			}, nil, nil
		}

		if false {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("this is what I generated: %s", string(output))},
				},
			}, nil, nil
		}
		_, err = zp.AVTransport.AddURIToQueue(&avt.AddURIToQueueArgs{
			InstanceID:                      0,
			EnqueuedURI:                     resURI,
			EnqueuedURIMetaData:             string(output),
			DesiredFirstTrackNumberEnqueued: 1,
			EnqueueAsNext:                   true,
		})

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to add Spotify URI %s to queue for room %s: %s", resURI, params.RoomName, err)},
				},
			}, nil, nil
		}

		err = zp.Play()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to play Spotify URI %s in room %s: %s", resURI, params.RoomName, err)},
				},
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playing Spotify URI %s in %s", trackInfo.URI, params.RoomName)},
			},
		}, nil, nil
	})
}