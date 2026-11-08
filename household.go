package sonos

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	avt "github.com/caglar10ur/sonos/services/avtransport"
	ren "github.com/caglar10ur/sonos/services/renderingcontrol"
)

// ErrRoomNotFound is returned by FindRoom for a room the household doesn't
// have.
var ErrRoomNotFound = errors.New("sonos: room not found")

// Household is a snapshot of a household's rooms and how they are grouped.
// It doesn't follow changes; call Refresh for a new snapshot, e.g. after
// grouping rooms.
//
// Room-level settings such as volume are on Room. Playback, the queue and
// group volume are on Group, which sends them to the group's coordinator.
type Household struct {
	// Rooms are in the order the topology lists them.
	Rooms []*Room
	// Groups are sets of rooms playing together; a room that isn't grouped
	// is a group of its own.
	Groups []*Group

	sonos   *Sonos
	from    *ZonePlayer // where the topology was read
	players *playerCache
}

// Room is a room: one player, or several acting as one (a stereo pair, a
// sub, surrounds).
type Room struct {
	Name string
	// UUID identifies the room's main player, e.g. "RINCON_000E58CDCA4001400".
	UUID string
	// Speakers counts the room's players, including bonded ones.
	Speakers int

	group    *Group
	location string
	players  *playerCache
}

// Group is a set of rooms playing the same thing. Its playback methods are
// sent to the coordinator.
type Group struct {
	ID string
	// Coordinator is the room that plays for the group.
	Coordinator *Room
	// Rooms are the group's rooms, the coordinator first.
	Rooms []*Room

	sonos *Sonos
}

// Household finds a player and returns a snapshot of its household.
func (s *Sonos) Household(ctx context.Context) (*Household, error) {
	first, err := s.firstPlayer(ctx)
	if err != nil {
		return nil, err
	}
	return s.householdFrom(ctx, first)
}

// householdFrom returns a snapshot of the household read from zp.
func (s *Sonos) householdFrom(ctx context.Context, zp *ZonePlayer) (*Household, error) {
	cache := &playerCache{client: zp.client, byUUID: map[string]*ZonePlayer{zp.UUID(): zp}}
	return newHousehold(ctx, s, zp, cache)
}

// FindRoom returns the room with the given name, grouped or not, from a
// fresh snapshot of the household. It returns an error wrapping
// ErrRoomNotFound if there is no such room.
func (s *Sonos) FindRoom(ctx context.Context, name string) (*Room, error) {
	h, err := s.Household(ctx)
	if err != nil {
		return nil, err
	}
	r := h.Room(name)
	if r == nil {
		return nil, fmt.Errorf("%w: %q", ErrRoomNotFound, name)
	}
	return r, nil
}

// refreshAttempt bounds how long Refresh waits for one player, so that a
// player that stopped answering doesn't use up the caller's deadline.
const refreshAttempt = 2 * time.Second

// Refresh returns a new snapshot of the household. Players already set up
// are reused, so it usually costs a single request. It asks the player the
// last snapshot came from, then the others, giving each a few seconds.
func (h *Household) Refresh(ctx context.Context) (*Household, error) {
	players := []*ZonePlayer{h.from}
	for _, zp := range h.players.all() {
		if zp != h.from {
			players = append(players, zp)
		}
	}
	var errs []error
	for _, zp := range players {
		attemptCtx, cancel := context.WithTimeout(ctx, refreshAttempt)
		fresh, err := newHousehold(attemptCtx, h.sonos, zp, h.players)
		cancel()
		if err == nil {
			return fresh, nil
		}
		errs = append(errs, err)
		if ctx.Err() != nil {
			break
		}
	}
	return nil, fmt.Errorf("sonos: no player answered: %w", errors.Join(errs...))
}

// Room returns the room with the given name, or nil.
func (h *Household) Room(name string) *Room {
	for _, r := range h.Rooms {
		if r.Name == name {
			return r
		}
	}
	return nil
}

func newHousehold(ctx context.Context, s *Sonos, from *ZonePlayer, cache *playerCache) (*Household, error) {
	state, err := from.ZoneGroupState(ctx)
	if err != nil {
		return nil, err
	}
	h := &Household{sonos: s, from: from, players: cache}
	for _, zg := range state.ZoneGroups {
		g := &Group{ID: zg.ID, sonos: s}
		for _, m := range zg.ZoneGroupMember {
			if m.Invisible == "1" {
				continue // part of another member's room
			}
			r := &Room{
				Name:     m.ZoneName,
				UUID:     m.UUID,
				Speakers: speakers(zg, m),
				group:    g,
				location: m.Location,
				players:  cache,
			}
			if m.UUID == zg.Coordinator {
				g.Coordinator = r
				g.Rooms = append([]*Room{r}, g.Rooms...)
			} else {
				g.Rooms = append(g.Rooms, r)
			}
			h.Rooms = append(h.Rooms, r)
		}
		if g.Coordinator != nil {
			h.Groups = append(h.Groups, g)
		}
	}
	return h, nil
}

// speakers counts the players of a room: the member itself, the hidden
// members bonded to it (listed in its channel map) and its satellites.
func speakers(g ZoneGroup, m ZoneGroupMember) int {
	n := 1 + len(m.Satellite)
	for _, other := range g.ZoneGroupMember {
		if other.UUID != m.UUID && other.Invisible == "1" && strings.Contains(m.ChannelMapSet, other.UUID+":") {
			n++
		}
	}
	return n
}

// playerCache holds the players of a household by UUID, shared between
// snapshots so each is set up once.
type playerCache struct {
	client *http.Client
	mu     sync.Mutex
	byUUID map[string]*ZonePlayer
}

func (c *playerCache) get(ctx context.Context, uuid, location string) (*ZonePlayer, error) {
	c.mu.Lock()
	zp := c.byUUID[uuid]
	c.mu.Unlock()
	if zp != nil && zp.location.String() == location {
		return zp, nil
	}
	zp, err := playerAt(ctx, location, c.client)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.byUUID[uuid] = zp
	c.mu.Unlock()
	return zp, nil
}

func (c *playerCache) all() []*ZonePlayer {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*ZonePlayer, 0, len(c.byUUID))
	for _, zp := range c.byUUID {
		out = append(out, zp)
	}
	return out
}

// Group returns the group the room was in when the snapshot was taken.
func (r *Room) Group() *Group {
	return r.group
}

// Player returns the room's main player, for the UPnP services it exposes.
func (r *Room) Player(ctx context.Context) (*ZonePlayer, error) {
	return r.players.get(ctx, r.UUID, r.location)
}

// Volume returns the room's volume, 0 to 100.
func (r *Room) Volume(ctx context.Context) (int, error) {
	zp, err := r.Player(ctx)
	if err != nil {
		return 0, err
	}
	res, err := zp.RenderingControl.GetVolume(ctx, &ren.GetVolumeArgs{Channel: ren.ChannelMaster})
	if err != nil {
		return 0, err
	}
	return int(res.CurrentVolume), nil
}

// SetVolume sets the room's volume, 0 to 100.
func (r *Room) SetVolume(ctx context.Context, volume int) error {
	if err := validVolume(volume); err != nil {
		return err
	}
	zp, err := r.Player(ctx)
	if err != nil {
		return err
	}
	_, err = zp.RenderingControl.SetVolume(ctx, &ren.SetVolumeArgs{
		Channel:       ren.ChannelMaster,
		DesiredVolume: uint16(volume),
	})
	return err
}

func validVolume(v int) error {
	if v < 0 || v > 100 {
		return fmt.Errorf("sonos: volume %d out of range 0-100", v)
	}
	return nil
}

// IsMuted reports whether the room is muted.
func (r *Room) IsMuted(ctx context.Context) (bool, error) {
	zp, err := r.Player(ctx)
	if err != nil {
		return false, err
	}
	res, err := zp.RenderingControl.GetMute(ctx, &ren.GetMuteArgs{Channel: ren.MuteChannelMaster})
	if err != nil {
		return false, err
	}
	return res.CurrentMute, nil
}

// Mute mutes the room.
func (r *Room) Mute(ctx context.Context) error {
	return r.SetMute(ctx, true)
}

// Unmute unmutes the room.
func (r *Room) Unmute(ctx context.Context) error {
	return r.SetMute(ctx, false)
}

// SetMute mutes or unmutes the room.
func (r *Room) SetMute(ctx context.Context, mute bool) error {
	zp, err := r.Player(ctx)
	if err != nil {
		return err
	}
	_, err = zp.RenderingControl.SetMute(ctx, &ren.SetMuteArgs{Channel: ren.MuteChannelMaster, DesiredMute: mute})
	return err
}

// Join makes the room play with group g, leaving its current group. It
// returns once the room's player reports the new grouping, so a
// Household.Refresh afterwards sees it; the snapshot itself isn't updated.
func (r *Room) Join(ctx context.Context, g *Group) error {
	if g.Coordinator.UUID == r.UUID {
		return nil // already its coordinator
	}
	zp, err := r.Player(ctx)
	if err != nil {
		return err
	}
	if _, err := zp.AVTransport.SetAVTransportURI(ctx, &avt.SetAVTransportURIArgs{CurrentURI: "x-rincon:" + g.Coordinator.UUID}); err != nil {
		return err
	}
	return r.awaitGroup(ctx, zp, "join "+g.Coordinator.Name, func(zg ZoneGroup) bool {
		return zg.Coordinator == g.Coordinator.UUID
	})
}

// Leave takes the room out of its group, into a group of its own. Leaving
// the coordinator hands the rest of the group to another room. It returns
// once the room's player reports the new grouping, so a Household.Refresh
// afterwards sees it; the snapshot itself isn't updated.
func (r *Room) Leave(ctx context.Context) error {
	zp, err := r.Player(ctx)
	if err != nil {
		return err
	}
	if _, err := zp.AVTransport.BecomeCoordinatorOfStandaloneGroup(ctx, nil); err != nil {
		return err
	}
	return r.awaitGroup(ctx, zp, "leave its group", func(zg ZoneGroup) bool {
		return zg.Coordinator == r.UUID && !slices.ContainsFunc(zg.ZoneGroupMember, func(m ZoneGroupMember) bool {
			return m.UUID != r.UUID && m.Invisible != "1"
		})
	})
}

// groupingDelay bounds how long Join and Leave wait for the topology to
// show the change, which takes a player about a second.
const groupingDelay = 5 * time.Second

// awaitGroup polls the topology zp reports until the group holding the
// room satisfies done.
func (r *Room) awaitGroup(ctx context.Context, zp *ZonePlayer, what string, done func(ZoneGroup) bool) error {
	ctx, cancel := context.WithTimeout(ctx, groupingDelay)
	defer cancel()
	for {
		state, err := zp.ZoneGroupState(ctx)
		if err == nil {
			for _, zg := range state.ZoneGroups {
				if slices.ContainsFunc(zg.ZoneGroupMember, func(m ZoneGroupMember) bool { return m.UUID == r.UUID }) && done(zg) {
					return nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("sonos: waiting for %s to %s: %w", r.Name, what, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
