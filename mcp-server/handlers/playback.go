package handlers

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/caglar10ur/sonos"
	"github.com/caglar10ur/sonos/didl"
	avt "github.com/caglar10ur/sonos/services/AVTransport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handlers) GetNowPlayingHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		info, err := zp.AVTransport.GetPositionInfo(&avt.GetPositionInfoArgs{
			InstanceID: 0,
		})
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		var lite didl.Lite
		if err := xml.Unmarshal([]byte(info.TrackMetaData), &lite); err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		if len(lite.Item) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Nothing is playing"},
				},
			}, nil, nil
		}

		item := lite.Item[0]
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

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Title: %s, Artist: %s, Album: %s", title, artist, album)},
			},
		}, nil, nil
	})
}

func (h *Handlers) PlayHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Play()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playback started in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) StopHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Stop()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playback stopped in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) PauseHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Pause()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playback paused in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) NextHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Next()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playing next track in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) PreviousHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		err := zp.Previous()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playing previous track in %s", params.RoomName)},
			},
		}, nil, nil
	})
}

func (h *Handlers) GetPositionInfoHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		info, err := zp.GetPositionInfo()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Track: %d, Duration: %s, Elapsed: %s", info.Track, info.TrackDuration, info.RelTime)},
			},
		}, nil, nil
	})
}
