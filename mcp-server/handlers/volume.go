package handlers

import (
	"context"
	"fmt"

	"github.com/caglar10ur/sonos"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handlers) GetVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		volume, err := zp.GetVolume()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Current volume in %s is %d", params.RoomName, volume)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SetVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params SetVolumeParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SetVolume(params.Volume)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Volume in %s set to %d", params.RoomName, params.Volume)},
			},
		}, nil, nil
	})
}

func (h *Handlers) MuteHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Mute()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Room %s muted", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) UnmuteHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Unmute()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Room %s unmuted", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetMuteStatusHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		isMuted, err := zp.IsMuted()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Room %s mute status: %t", params.RoomName, isMuted)},
			},
		}, nil, nil
	})
}
