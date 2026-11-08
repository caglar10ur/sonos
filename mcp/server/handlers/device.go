package handlers

import (
	"context"

	"github.com/caglar10ur/sonos"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ZoneInfo struct {
	SerialNumber    string `json:"serial_number"`
	SoftwareVersion string `json:"software_version"`
	IPAddress       string `json:"ip_address"`
	MACAddress      string `json:"mac_address"`
}

type MediaInfo struct {
	NumberOfTracks uint32 `json:"number_of_tracks"`
	Duration       string `json:"duration"`
	CurrentURI     string `json:"current_uri"`
	NextURI        string `json:"next_uri"`
	PlayMedium     string `json:"play_medium"`
	WriteStatus    string `json:"write_status"`
}

func (h *Handlers) GetMediaInfoHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		zp, err := g.Player(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		mediaInfo, err := zp.AVTransport.GetMediaInfo(ctx, nil)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return nil, MediaInfo{
			NumberOfTracks: mediaInfo.NrTracks,
			Duration:       mediaInfo.MediaDuration,
			CurrentURI:     mediaInfo.CurrentURI,
			NextURI:        mediaInfo.NextURI,
			PlayMedium:     string(mediaInfo.PlayMedium),
			WriteStatus:    mediaInfo.WriteStatus,
		}, nil
	})
}

func (h *Handlers) ListSonosDevicesHandler(ctx context.Context, req *mcp.CallToolRequest, params any) (*mcp.CallToolResult, any, error) {
	ctx, cancel := context.WithTimeout(ctx, h.searchTimeout)
	defer cancel()

	household, err := h.sonosController.Household(ctx)
	if err != nil {
		return handleGenericError(err), nil, nil
	}
	rooms := make([]string, 0, len(household.Rooms))
	for _, r := range household.Rooms {
		rooms = append(rooms, r.Name)
	}
	return nil, map[string]any{"rooms": rooms}, nil
}

func (h *Handlers) GetZoneInfoHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withPlayer(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		info, err := zp.DeviceProperties.GetZoneInfo(ctx, nil)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return nil, ZoneInfo{
			SerialNumber:    info.SerialNumber,
			SoftwareVersion: info.SoftwareVersion,
			IPAddress:       info.IPAddress,
			MACAddress:      info.MACAddress,
		}, nil
	})
}

func (h *Handlers) GetUUIDHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(r *sonos.Room) (*mcp.CallToolResult, any, error) {
		return nil, map[string]string{"uuid": r.UUID, "room": params.RoomName}, nil
	})
}
