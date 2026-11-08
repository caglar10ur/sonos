package handlers

import (
	"context"
	"fmt"

	"github.com/caglar10ur/sonos"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ZoneGroupMember struct {
	ZoneName string `json:"zone_name"`
	UUID     string `json:"uuid"`
}

type ZoneGroup struct {
	ID          string            `json:"id"`
	Coordinator string            `json:"coordinator"`
	Members     []ZoneGroupMember `json:"members"`
}

func (h *Handlers) AddGroupMemberHandler(ctx context.Context, req *mcp.CallToolRequest, params AddGroupMemberParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.CoordinatorRoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		return h.withRoom(ctx, params.MemberRoomName, func(member *sonos.Room) (*mcp.CallToolResult, any, error) {
			if err := member.Join(ctx, g); err != nil {
				return handleError(err, params.CoordinatorRoomName), nil, nil
			}
			return SuccessResponse(fmt.Sprintf("Member %s added to group %s", params.MemberRoomName, params.CoordinatorRoomName)), nil, nil
		})
	})
}

func (h *Handlers) RemoveGroupMemberHandler(ctx context.Context, req *mcp.CallToolRequest, params RemoveGroupMemberParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.CoordinatorRoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		var member *sonos.Room
		for _, r := range g.Rooms {
			if r.Name == params.MemberRoomName {
				member = r
			}
		}
		if member == nil {
			return handleError(fmt.Errorf("member room %s not found in group %s", params.MemberRoomName, params.CoordinatorRoomName), params.CoordinatorRoomName), nil, nil
		}
		if err := member.Leave(ctx); err != nil {
			return handleError(err, params.MemberRoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Member %s removed from group %s", params.MemberRoomName, params.CoordinatorRoomName)), nil, nil
	})
}

func (h *Handlers) ListGroupsHandler(ctx context.Context, req *mcp.CallToolRequest, params any) (*mcp.CallToolResult, any, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, h.searchTimeout)
	defer cancel()
	household, err := h.sonosController.Household(lookupCtx)
	if err != nil {
		return handleGenericError(err), nil, nil
	}

	groups := []ZoneGroup{}
	for _, g := range household.Groups {
		group := ZoneGroup{ID: g.ID, Coordinator: g.Coordinator.UUID}
		for _, r := range g.Rooms {
			group.Members = append(group.Members, ZoneGroupMember{ZoneName: r.Name, UUID: r.UUID})
		}
		groups = append(groups, group)
	}
	return nil, map[string]any{"groups": groups}, nil
}

func (h *Handlers) GetZoneGroupAttributesHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withPlayer(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		attrs, err := zp.ZoneGroupTopology.GetZoneGroupAttributes(ctx, nil)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return nil, map[string]string{
			"name":         attrs.CurrentZoneGroupName,
			"id":           attrs.CurrentZoneGroupID,
			"uuids":        attrs.CurrentZonePlayerUUIDsInGroup,
			"household_id": attrs.CurrentMuseHouseholdId,
		}, nil
	})
}

func (h *Handlers) GetGroupVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		volume, err := g.Volume(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return nil, map[string]any{"room": params.RoomName, "group_volume": volume}, nil
	})
}

func (h *Handlers) SetGroupVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params SetGroupVolumeParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		err := g.SetVolume(ctx, params.Volume)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return SuccessResponse(fmt.Sprintf("Group volume in %s set to %d", params.RoomName, params.Volume)), nil, nil
	})
}
