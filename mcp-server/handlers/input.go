package handlers

import (
	"context"
	"fmt"

	"github.com/caglar10ur/sonos"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handlers) GetAudioInputAttributesHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		attrs, err := zp.GetAudioInputAttributes()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Audio Input Name: %s, Icon: %s", attrs.CurrentName, attrs.CurrentIcon)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetLineInLevelHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		levels, err := zp.GetLineInLevel()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Line-in levels for room %s: Left: %d, Right: %d", params.RoomName, levels.CurrentLeftLineInLevel, levels.CurrentRightLineInLevel)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SetLineInLevelHandler(ctx context.Context, req *mcp.CallToolRequest, params SetLineInLevelParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SetLineInLevel(int32(params.DesiredLeftLineInLevel), int32(params.DesiredRightLineInLevel))
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Line-in levels for room %s set to Left: %d, Right: %d", params.RoomName, params.DesiredLeftLineInLevel, params.DesiredRightLineInLevel)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SelectAudioHandler(ctx context.Context, req *mcp.CallToolRequest, params SelectAudioParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SelectAudio(params.ObjectID)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Audio input for room %s selected to %s", params.RoomName, params.ObjectID)},
			},
		}, nil, nil
	})
}

func (h *Handlers) SwitchToLineInHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SwitchToLineIn()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Switched to line-in in %s", params.RoomName)},
			},
		}, nil, nil
	})
}
