package handlers

import (
	"context"
	"fmt"

	"github.com/caglar10ur/sonos"
	avt "github.com/caglar10ur/sonos/services/AVTransport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handlers) GetMediaInfoHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		mediaInfo, err := zp.AVTransport.GetMediaInfo(&avt.GetMediaInfoArgs{})
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("mediaInfo: %#v", mediaInfo)},
			},
		}, nil, nil
	})
}

func (h *Handlers) ListSonosDevicesHandler(ctx context.Context, req *mcp.CallToolRequest, params any) (*mcp.CallToolResult, any, error) {
	ctx, cancel := context.WithTimeout(ctx, h.searchTimeout)
	defer cancel()

	devices, err := h.sonosController.ListSonosDevices(ctx)
	if err != nil {
		return handleGenericError(err), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("%v", devices)},
		},
	}, nil, nil
}

func (h *Handlers) GetZoneInfoHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		info, err := zp.GetZoneInfo()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Serial Number: %s, Software Version: %s, IP Address: %s, MAC Address: %s", info.SerialNumber, info.SoftwareVersion, info.IPAddress, info.MACAddress)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetUUIDHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		uuid := zp.UUID()
		if uuid == "" {
			return handleError(fmt.Errorf("failed to get UUID"), params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("UUID for room %s: %s", params.RoomName, uuid)},
			},
		}, nil, nil
	})
}
