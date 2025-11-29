package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/caglar10ur/sonos"
	avt "github.com/caglar10ur/sonos/services/AVTransport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handlers) AddGroupMemberHandler(ctx context.Context, req *mcp.CallToolRequest, params AddGroupMemberParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.CoordinatorRoomName, func(coordinatorZp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		return h.withRoom(ctx, params.MemberRoomName, func(memberZp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
			_, err := memberZp.AVTransport.SetAVTransportURI(&avt.SetAVTransportURIArgs{
				InstanceID: 0,
				CurrentURI: fmt.Sprintf("x-rincon:%s", coordinatorZp.UUID()),
			})
			if err != nil {
				return handleError(err, params.CoordinatorRoomName), nil, nil
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
	return h.withRoom(ctx, params.CoordinatorRoomName, func(coordinatorZp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		zoneGroupState, err := coordinatorZp.GetZoneGroupState()
		if err != nil {
			return handleError(err, params.CoordinatorRoomName), nil, nil
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
			return handleGenericError(fmt.Errorf("member room location %s parsing failed", location)), nil, nil
		}

		zp, err := sonos.NewZonePlayer(sonos.WithLocation(u))
		if err != nil {
			return handleError(err, params.MemberRoomName), nil, nil
		}

		_, err = zp.AVTransport.BecomeCoordinatorOfStandaloneGroup(&avt.BecomeCoordinatorOfStandaloneGroupArgs{
			InstanceID: 0,
		})
		if err != nil {
			return handleError(err, params.MemberRoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Member %s removed from group %s", params.MemberRoomName, params.CoordinatorRoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) ListGroupsHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		var groupInfo strings.Builder

		zoneGroupState, err := zp.GetZoneGroupState()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
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
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		attrs, err := zp.GetZoneGroupAttributes()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Zone Group Name: %s, Zone Group ID: %s, Zone Player UUIDs in Group: %s, Muse Household ID: %s", attrs.CurrentZoneGroupName, attrs.CurrentZoneGroupID, attrs.CurrentZonePlayerUUIDsInGroup, attrs.CurrentMuseHouseholdId)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetGroupVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		volume, err := zp.GetGroupVolume()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Current group volume in %s is %d", params.RoomName, volume)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SetGroupVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params SetGroupVolumeParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SetGroupVolume(params.Volume)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Group volume in %s set to %d", params.RoomName, params.Volume)},
			},
		}, nil, nil
	})
}
