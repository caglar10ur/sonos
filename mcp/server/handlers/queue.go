package handlers

import (
	"context"
	"fmt"

	"github.com/caglar10ur/sonos"
	"github.com/caglar10ur/sonos/didl"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type QueueItem struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
}

func (h *Handlers) ListQueueHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		queueItems, err := g.Queue(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		items := []QueueItem{} // [] rather than null when the queue is empty
		for _, item := range queueItems {
			items = append(items, QueueItem{
				Title:  didl.First(item.Title),
				Artist: didl.First(item.Creator),
				Album:  didl.First(item.Album),
			})
		}

		return nil, map[string]any{"room": params.RoomName, "items": items}, nil
	})
}

func (h *Handlers) SwitchToQueueHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		err := g.PlayQueue(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return SuccessResponse(fmt.Sprintf("Switched to queue in %s", params.RoomName)), nil, nil
	})
}

func (h *Handlers) PlayQueueTrackHandler(ctx context.Context, req *mcp.CallToolRequest, params QueuePositionParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		if err := g.PlayQueue(ctx); err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		if err := g.SeekTrack(ctx, params.Position); err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		if err := g.Play(ctx); err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Playing queue track %d in %s", params.Position, params.RoomName)), nil, nil
	})
}

func (h *Handlers) AddToQueueHandler(ctx context.Context, req *mcp.CallToolRequest, params AddToQueueParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		pos, err := g.AddToQueue(ctx, params.URI, params.Metadata, params.Position)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return nil, map[string]any{"room": params.RoomName, "position": pos}, nil
	})
}

func (h *Handlers) RemoveFromQueueHandler(ctx context.Context, req *mcp.CallToolRequest, params QueuePositionParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		if err := g.RemoveFromQueue(ctx, params.Position); err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Removed queue track %d in %s", params.Position, params.RoomName)), nil, nil
	})
}

func (h *Handlers) ClearQueueHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		if err := g.ClearQueue(ctx); err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Cleared the queue in %s", params.RoomName)), nil, nil
	})
}
