package handlers

import (
	"context"
	"fmt"

	"github.com/caglar10ur/sonos"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type NowPlaying struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
}

type PositionInfo struct {
	// Track is the 1-based position in the queue, 0 when not playing from it.
	Track    int    `json:"track"`
	Duration string `json:"duration"`
	Position string `json:"position"`
}

func (h *Handlers) GetNowPlayingHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		np, err := g.NowPlaying(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		if np.Track == nil {
			return SuccessResponse("Nothing is playing"), nil, nil
		}
		return nil, NowPlaying{Title: np.Track.Title, Artist: np.Track.Artist, Album: np.Track.Album}, nil
	})
}

func (h *Handlers) PlayHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		err := g.Play(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Playback started in %s", params.RoomName)), nil, nil
	})
}

func (h *Handlers) StopHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		err := g.Stop(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Playback stopped in %s", params.RoomName)), nil, nil
	})
}

func (h *Handlers) PauseHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		err := g.Pause(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Playback paused in %s", params.RoomName)), nil, nil
	})
}

func (h *Handlers) NextHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		err := g.Next(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Playing next track in %s", params.RoomName)), nil, nil
	})
}

func (h *Handlers) PreviousHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		err := g.Previous(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Playing previous track in %s", params.RoomName)), nil, nil
	})
}

func (h *Handlers) GetPositionInfoHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		np, err := g.NowPlaying(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		info := PositionInfo{Track: np.QueuePosition, Position: np.Position.String()}
		if np.Track != nil {
			info.Duration = np.Track.Duration.String()
		}
		return nil, info, nil
	})
}

func (h *Handlers) SeekHandler(ctx context.Context, req *mcp.CallToolRequest, params SeekParams) (*mcp.CallToolResult, any, error) {
	pos, err := sonos.ParsePosition(params.Position)
	if err != nil {
		return handleError(err, params.RoomName), nil, nil
	}
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		if err := g.Seek(ctx, pos); err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Moved to %v in %s", pos, params.RoomName)), nil, nil
	})
}

type PlayMode struct {
	Shuffle bool   `json:"shuffle"`
	Repeat  string `json:"repeat"`
}

var repeatNames = map[sonos.Repeat]string{sonos.RepeatOff: "off", sonos.RepeatAll: "all", sonos.RepeatOne: "one"}

func (h *Handlers) GetPlayModeHandler(ctx context.Context, req *mcp.CallToolRequest, params RoomNameParams) (*mcp.CallToolResult, any, error) {
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		mode, err := g.PlayMode(ctx)
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return nil, PlayMode{Shuffle: mode.Shuffle, Repeat: repeatNames[mode.Repeat]}, nil
	})
}

func (h *Handlers) SetPlayModeHandler(ctx context.Context, req *mcp.CallToolRequest, params SetPlayModeParams) (*mcp.CallToolResult, any, error) {
	mode := sonos.PlayMode{Shuffle: params.Shuffle, Repeat: -1}
	for r, name := range repeatNames {
		if name == params.Repeat || (params.Repeat == "" && r == sonos.RepeatOff) {
			mode.Repeat = r
		}
	}
	if mode.Repeat < 0 {
		return handleError(fmt.Errorf("repeat must be off, all or one, not %q", params.Repeat), params.RoomName), nil, nil
	}
	return h.withGroup(ctx, params.RoomName, func(g *sonos.Group) (*mcp.CallToolResult, any, error) {
		if err := g.SetPlayMode(ctx, mode); err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return SuccessResponse(fmt.Sprintf("Play mode in %s set to shuffle=%v repeat=%s", params.RoomName, mode.Shuffle, repeatNames[mode.Repeat])), nil, nil
	})
}
