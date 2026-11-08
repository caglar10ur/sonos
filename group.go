package sonos

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/caglar10ur/sonos/didl"
	avt "github.com/caglar10ur/sonos/services/avtransport"
	rcg "github.com/caglar10ur/sonos/services/grouprenderingcontrol"
	que "github.com/caglar10ur/sonos/services/queue"
)

// Name returns the group's name: its coordinator's room, plus how many
// other rooms play with it, e.g. "Kitchen + 2".
func (g *Group) Name() string {
	if len(g.Rooms) <= 1 {
		return g.Coordinator.Name
	}
	return fmt.Sprintf("%s + %d", g.Coordinator.Name, len(g.Rooms)-1)
}

// Player returns the coordinator's player, for the UPnP services it
// exposes. Group-level actions, e.g. AVTransport's, must go to it.
func (g *Group) Player(ctx context.Context) (*ZonePlayer, error) {
	return g.Coordinator.Player(ctx)
}

// transport runs fn against the coordinator's AVTransport.
func (g *Group) transport(ctx context.Context, fn func(*avt.Service) error) error {
	zp, err := g.Player(ctx)
	if err != nil {
		return err
	}
	return fn(zp.AVTransport)
}

// Play starts or resumes playback.
func (g *Group) Play(ctx context.Context) error {
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.Play(ctx, &avt.PlayArgs{Speed: avt.TransportPlaySpeed1})
		return err
	})
}

// Pause pauses playback.
func (g *Group) Pause(ctx context.Context) error {
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.Pause(ctx, nil)
		return err
	})
}

// Stop stops playback.
func (g *Group) Stop(ctx context.Context) error {
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.Stop(ctx, nil)
		return err
	})
}

// Next skips to the next track.
func (g *Group) Next(ctx context.Context) error {
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.Next(ctx, nil)
		return err
	})
}

// Previous goes back to the previous track.
func (g *Group) Previous(ctx context.Context) error {
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.Previous(ctx, nil)
		return err
	})
}

// SetURI sets what the group plays, e.g. a stream URL, with optional
// DIDL-Lite metadata. Call Play to start it.
func (g *Group) SetURI(ctx context.Context, uri, metadata string) error {
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.SetAVTransportURI(ctx, &avt.SetAVTransportURIArgs{CurrentURI: uri, CurrentURIMetaData: metadata})
		return err
	})
}

// PlayQueue makes the group play from its queue. Call Play to start it.
func (g *Group) PlayQueue(ctx context.Context) error {
	return g.SetURI(ctx, "x-rincon-queue:"+g.Coordinator.UUID+"#0", "")
}

// PlayLineIn makes the group play the line-in input of a room, which may be
// in another group. Call Play to start it.
func (g *Group) PlayLineIn(ctx context.Context, from *Room) error {
	return g.SetURI(ctx, "x-rincon-stream:"+from.UUID, "")
}

// NowPlaying returns what the group is playing and how far into it.
func (g *Group) NowPlaying(ctx context.Context) (*NowPlaying, error) {
	np, _, err := g.nowPlaying(ctx)
	return np, err
}

// nowPlaying is NowPlaying, also returning the media info it reads.
func (g *Group) nowPlaying(ctx context.Context) (*NowPlaying, *avt.GetMediaInfoResponse, error) {
	zp, err := g.Player(ctx)
	if err != nil {
		return nil, nil, err
	}
	transport, err := zp.AVTransport.GetTransportInfo(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	pos, err := zp.AVTransport.GetPositionInfo(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	np := &NowPlaying{
		State:    PlaybackState(transport.CurrentTransportState),
		Track:    trackFrom(pos.TrackURI, pos.TrackMetaData, pos.TrackDuration, zp.location),
		Position: parseDuration(pos.RelTime),
		At:       time.Now(),
	}
	media, err := zp.AVTransport.GetMediaInfo(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	if strings.HasPrefix(media.CurrentURI, "x-rincon-queue:") {
		np.QueuePosition = int(pos.Track)
	}
	return np, media, nil
}

// queuePageSize is how many items Queue requests per Browse call.
const queuePageSize = 100

// Queue returns every item in the group's queue, fetching it in pages.
func (g *Group) Queue(ctx context.Context) ([]didl.Item, error) {
	zp, err := g.Player(ctx)
	if err != nil {
		return nil, err
	}
	var items []didl.Item
	for {
		res, err := zp.Queue.Browse(ctx, &que.BrowseArgs{
			StartingIndex:  uint32(len(items)),
			RequestedCount: queuePageSize,
		})
		if err != nil {
			return nil, err
		}
		page, err := didl.Parse(res.Result)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Item...)

		if res.NumberReturned == 0 || len(page.Item) == 0 || uint32(len(items)) >= res.TotalMatches {
			return items, nil
		}
	}
}

// Volume returns the group's volume, 0 to 100.
func (g *Group) Volume(ctx context.Context) (int, error) {
	zp, err := g.Player(ctx)
	if err != nil {
		return 0, err
	}
	res, err := zp.GroupRenderingControl.GetGroupVolume(ctx, nil)
	if err != nil {
		return 0, err
	}
	return int(res.CurrentVolume), nil
}

// SetVolume sets the group's volume, 0 to 100, keeping the rooms' volumes
// in proportion.
func (g *Group) SetVolume(ctx context.Context, volume int) error {
	if err := validVolume(volume); err != nil {
		return err
	}
	zp, err := g.Player(ctx)
	if err != nil {
		return err
	}
	// The player scales each room from a snapshot of their volumes; take a
	// fresh one so changes made in a single room since are respected.
	if _, err := zp.GroupRenderingControl.SnapshotGroupVolume(ctx, nil); err != nil {
		return err
	}
	_, err = zp.GroupRenderingControl.SetGroupVolume(ctx, &rcg.SetGroupVolumeArgs{DesiredVolume: uint16(volume)})
	return err
}

// Seek moves to a position in the current track.
func (g *Group) Seek(ctx context.Context, position time.Duration) error {
	if position < 0 {
		return fmt.Errorf("sonos: position %v out of range", position)
	}
	return g.seek(ctx, avt.SeekModeRelTime, formatDuration(position))
}

// SeekTrack moves to a track of the queue, counting from 1 as
// NowPlaying.QueuePosition does. The group must be playing from its queue;
// see PlayQueue.
func (g *Group) SeekTrack(ctx context.Context, position int) error {
	if position < 1 {
		return fmt.Errorf("sonos: queue position %d out of range", position)
	}
	return g.seek(ctx, avt.SeekModeTrackNR, strconv.Itoa(position))
}

func (g *Group) seek(ctx context.Context, unit avt.SeekModeEnum, target string) error {
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.Seek(ctx, &avt.SeekArgs{Unit: unit, Target: target})
		return err
	})
}

// AddToQueue adds a track, album or playlist to the group's queue, with
// optional DIDL-Lite metadata, and returns the queue position of its first
// track. at is the queue position to insert at, counting from 1; 0 appends.
func (g *Group) AddToQueue(ctx context.Context, uri, metadata string, at int) (int, error) {
	if at < 0 {
		return 0, fmt.Errorf("sonos: queue position %d out of range", at)
	}
	var first int
	err := g.transport(ctx, func(t *avt.Service) error {
		res, err := t.AddURIToQueue(ctx, &avt.AddURIToQueueArgs{
			EnqueuedURI:                     uri,
			EnqueuedURIMetaData:             metadata,
			DesiredFirstTrackNumberEnqueued: uint32(at),
		})
		if err == nil {
			first = int(res.FirstTrackNumberEnqueued)
		}
		return err
	})
	return first, err
}

// RemoveFromQueue removes the track at a queue position, counting from 1.
func (g *Group) RemoveFromQueue(ctx context.Context, position int) error {
	if position < 1 {
		return fmt.Errorf("sonos: queue position %d out of range", position)
	}
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.RemoveTrackFromQueue(ctx, &avt.RemoveTrackFromQueueArgs{ObjectID: "Q:0/" + strconv.Itoa(position)})
		return err
	})
}

// ClearQueue removes every track from the group's queue.
func (g *Group) ClearQueue(ctx context.Context) error {
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.RemoveAllTracksFromQueue(ctx, nil)
		return err
	})
}

// Repeat is what a group repeats.
type Repeat int

const (
	RepeatOff Repeat = iota
	RepeatAll        // repeat the queue
	RepeatOne        // repeat the current track
)

// PlayMode is how a group plays its queue.
type PlayMode struct {
	Shuffle bool
	Repeat  Repeat
}

var playModes = map[PlayMode]avt.CurrentPlayModeEnum{
	{false, RepeatOff}: avt.CurrentPlayModeNormal,
	{false, RepeatAll}: avt.CurrentPlayModeRepeatAll,
	{false, RepeatOne}: avt.CurrentPlayModeRepeatOne,
	{true, RepeatOff}:  avt.CurrentPlayModeShuffleNorepeat,
	{true, RepeatAll}:  avt.CurrentPlayModeShuffle,
	{true, RepeatOne}:  avt.CurrentPlayModeShuffleRepeatOne,
}

// PlayMode returns whether the group shuffles and what it repeats.
func (g *Group) PlayMode(ctx context.Context) (PlayMode, error) {
	var mode PlayMode
	err := g.transport(ctx, func(t *avt.Service) error {
		res, err := t.GetTransportSettings(ctx, nil)
		if err != nil {
			return err
		}
		for m, v := range playModes {
			if v == res.PlayMode {
				mode = m
				return nil
			}
		}
		return fmt.Errorf("sonos: unknown play mode %q", res.PlayMode)
	})
	return mode, err
}

// SetPlayMode sets whether the group shuffles and what it repeats.
func (g *Group) SetPlayMode(ctx context.Context, mode PlayMode) error {
	v, ok := playModes[mode]
	if !ok {
		return fmt.Errorf("sonos: invalid play mode %+v", mode)
	}
	return g.transport(ctx, func(t *avt.Service) error {
		_, err := t.SetPlayMode(ctx, &avt.SetPlayModeArgs{NewPlayMode: v})
		return err
	})
}

// Snapshot is what a group was playing, where, and how loud each of its
// rooms was, so that it can be put back, e.g. after an announcement:
//
//	snap, err := g.Snapshot(ctx)
//	// play the announcement with SetURI and Play, wait for it to end
//	err = snap.Restore(ctx)
//
// It doesn't save the queue's contents or the grouping.
type Snapshot struct {
	group         *Group
	uri, metadata string
	queuePosition int
	position      time.Duration
	playing       bool
	mode          PlayMode
	rooms         []roomSettings
}

type roomSettings struct {
	room   *Room
	volume int
	muted  bool
}

// Snapshot saves what the group is playing and the volume of its rooms.
func (g *Group) Snapshot(ctx context.Context) (*Snapshot, error) {
	np, media, err := g.nowPlaying(ctx)
	if err != nil {
		return nil, err
	}
	snap := &Snapshot{
		group:         g,
		uri:           media.CurrentURI,
		metadata:      media.CurrentURIMetaData,
		queuePosition: np.QueuePosition,
		position:      np.Position,
		playing:       np.State == Playing || np.State == Transitioning,
	}
	if snap.queuePosition > 0 {
		if snap.mode, err = g.PlayMode(ctx); err != nil {
			return nil, err
		}
	}
	for _, r := range g.Rooms {
		s := roomSettings{room: r}
		if s.volume, err = r.Volume(ctx); err != nil {
			return nil, err
		}
		if s.muted, err = r.IsMuted(ctx); err != nil {
			return nil, err
		}
		snap.rooms = append(snap.rooms, s)
	}
	return snap, nil
}

// Restore puts back what the snapshot saved, and resumes playback if the
// group was playing.
func (s *Snapshot) Restore(ctx context.Context) error {
	g := s.group
	if s.uri != "" {
		if err := g.SetURI(ctx, s.uri, s.metadata); err != nil {
			return err
		}
	}
	if s.queuePosition > 0 {
		if err := g.SetPlayMode(ctx, s.mode); err != nil {
			return err
		}
		if err := g.SeekTrack(ctx, s.queuePosition); err != nil {
			return err
		}
		if s.position > 0 {
			if err := g.Seek(ctx, s.position); err != nil {
				return err
			}
		}
	}
	for _, rs := range s.rooms {
		if err := rs.room.SetVolume(ctx, rs.volume); err != nil {
			return err
		}
		if err := rs.room.SetMute(ctx, rs.muted); err != nil {
			return err
		}
	}
	if s.playing {
		return g.Play(ctx)
	}
	return nil
}
