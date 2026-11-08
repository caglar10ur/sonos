package handlers

import (
	"context"
	"fmt"

	"github.com/caglar10ur/sonos"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handlers) GetVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(r *sonos.Room) (*mcp.CallToolResult, any, error) {
		volume, err := r.Volume(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return nil, map[string]any{"room": params.RoomName, "volume": volume}, nil
	})
}

func (h *Handlers) SetVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, params SetVolumeParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(r *sonos.Room) (*mcp.CallToolResult, any, error) {
		err := r.SetVolume(ctx, params.Volume)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return SuccessResponse(fmt.Sprintf("Volume in %s set to %d", params.RoomName, params.Volume)), nil, nil
	})
}

func (h *Handlers) MuteHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(r *sonos.Room) (*mcp.CallToolResult, any, error) {
		err := r.Mute(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return SuccessResponse(fmt.Sprintf("Room %s muted", params.RoomName)), nil, nil
	})
}

func (h *Handlers) UnmuteHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(r *sonos.Room) (*mcp.CallToolResult, any, error) {
		err := r.Unmute(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return SuccessResponse(fmt.Sprintf("Room %s unmuted", params.RoomName)), nil, nil
	})
}

func (h *Handlers) GetMuteStatusHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(r *sonos.Room) (*mcp.CallToolResult, any, error) {
		isMuted, err := r.IsMuted(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return nil, map[string]any{"room": params.RoomName, "is_muted": isMuted}, nil
	})
}
