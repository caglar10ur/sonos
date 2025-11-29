package handlers

import (
	"context"
	"fmt"

	"github.com/caglar10ur/sonos"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handlers) ListQueueHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		queueItems, err := zp.ListQueue()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
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

func (h *Handlers) SwitchToQueueHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.SwitchToQueue()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Switched to queue in %s", params.RoomName)},
			},
		}, nil, nil
	})
}
