package handlers

import (
	"context"
	"fmt"

	"github.com/caglar10ur/sonos"
	ain "github.com/caglar10ur/sonos/services/audioin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handlers) GetAudioInputAttributesHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withPlayer(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		attrs, err := zp.AudioIn.GetAudioInputAttributes(ctx, nil)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return nil, map[string]string{"name": attrs.CurrentName, "icon": attrs.CurrentIcon}, nil
	})
}

func (h *Handlers) GetLineInLevelHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withPlayer(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		levels, err := zp.AudioIn.GetLineInLevel(ctx, nil)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return nil, map[string]int{"left": int(levels.CurrentLeftLineInLevel), "right": int(levels.CurrentRightLineInLevel)}, nil
	})
}

func (h *Handlers) SetLineInLevelHandler(ctx context.Context, req *mcp.CallToolRequest, params SetLineInLevelParams) (*mcp.CallToolResult, any, error) {
	return h.withPlayer(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		_, err := zp.AudioIn.SetLineInLevel(ctx, &ain.SetLineInLevelArgs{
			DesiredLeftLineInLevel:  int32(params.DesiredLeftLineInLevel),
			DesiredRightLineInLevel: int32(params.DesiredRightLineInLevel),
		})
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return SuccessResponse(fmt.Sprintf("Line-in levels for room %s set to Left: %d, Right: %d", params.RoomName, params.DesiredLeftLineInLevel, params.DesiredRightLineInLevel)), nil, nil
	})
}

func (h *Handlers) SelectAudioHandler(ctx context.Context, req *mcp.CallToolRequest, params SelectAudioParams) (*mcp.CallToolResult, any, error) {
	return h.withPlayer(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		_, err := zp.AudioIn.SelectAudio(ctx, &ain.SelectAudioArgs{ObjectID: params.ObjectID})
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return SuccessResponse(fmt.Sprintf("Audio input for room %s selected to %s", params.RoomName, params.ObjectID)), nil, nil
	})
}

func (h *Handlers) SwitchToLineInHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	r, err := h.room(ctx, params.RoomName)
	if err != nil {
		return handleError(err, params.RoomName), nil, nil
	}
	// The room's group plays the room's own line-in.
	if err := r.Group().PlayLineIn(ctx, r); err != nil {
		return handleError(err, params.RoomName), nil, nil
	}
	return SuccessResponse(fmt.Sprintf("Switched to line-in in %s", params.RoomName)), nil, nil
}
