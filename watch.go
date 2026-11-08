package sonos

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	avt "github.com/caglar10ur/sonos/services/avtransport"
	rcg "github.com/caglar10ur/sonos/services/grouprenderingcontrol"
)

// ErrGroupChanged ends a Watch when a room joins or leaves the group, or
// another room becomes its coordinator. Refresh the household and watch the
// new group.
var ErrGroupChanged = errors.New("sonos: group changed")

// GroupState is the state of a group, as Watch reports it.
type GroupState struct {
	Group *Group
	NowPlaying
	PlayMode PlayMode
	// Volume and Muted are the group's; Rooms has each room's.
	Volume int
	Muted  bool
	// Rooms are in the order of Group.Rooms.
	Rooms []RoomState
}

// RoomState is the volume of one room of a group.
type RoomState struct {
	Room   *Room
	Volume int
	Muted  bool
}

const (
	// watchTimeout is the lifetime Watch's subscriptions ask for. It bounds
	// how long a player that rebooted goes unnoticed.
	watchTimeout = 5 * time.Minute
	// watchSettle is how long Watch waits after a change for the ones that
	// usually follow it, e.g. new metadata after a track change, so that a
	// burst of events is reported once.
	watchSettle = 100 * time.Millisecond
)

// watchFirstEvents is how long Watch waits for each player to send its
// initial state. Tests shorten it.
var watchFirstEvents = 10 * time.Second

// Watch reports the state of the group now and after every change, until
// ctx is done or the loop stops:
//
//	for st, err := range g.Watch(ctx) {
//		if err != nil {
//			log.Print(err) // e.g. ErrGroupChanged or ErrSubscriptionLost
//			break
//		}
//		fmt.Println(st.State, st.Track.Title, st.PositionAt(time.Now()))
//	}
//
// Each value is the whole state, so a slow loop misses nothing: changes
// made meanwhile are merged into the next value. The sequence ends after
// an error, e.g. ErrGroupChanged when rooms are grouped differently.
//
// Players don't send events as playback moves forward, so Position is
// read when the track or playback state changes; use PositionAt for the
// position at a later time.
//
// The group must come from Sonos.Household or Sonos.FindRoom, whose event
// listener Watch subscribes through, to the coordinator's playback and
// group volume, each room's volume, and the topology.
func (g *Group) Watch(ctx context.Context) iter.Seq2[*GroupState, error] {
	return func(yield func(*GroupState, error) bool) {
		w, err := g.startWatch(ctx)
		if err != nil {
			yield(nil, err)
			return
		}
		defer w.stop(ctx)
		w.run(ctx, yield)
	}
}

// watcher merges the events of a group's subscriptions into its state.
type watcher struct {
	group   *Group
	coord   *ZonePlayer
	changed chan struct{} // signalled after each event, holds at most one
	failed  chan error    // a subscription ended

	subs  []*Subscription
	names []string // of subs, for errors

	mu           sync.Mutex
	st           GroupState
	heard        []bool // whether each subscription sent its first event
	err          error  // set when the group changed
	needPosition bool   // the track or playback state changed

	// AVTransport variables, as last reported
	transportURI, track, trackURI, trackMeta, trackDuration string
}

func (g *Group) startWatch(ctx context.Context) (*watcher, error) {
	if g.sonos == nil {
		return nil, errors.New("sonos: Watch needs a group from Sonos.Household or Sonos.FindRoom")
	}
	coord, err := g.Player(ctx)
	if err != nil {
		return nil, err
	}
	w := &watcher{
		group:   g,
		coord:   coord,
		changed: make(chan struct{}, 1),
		st:      GroupState{Group: g},
	}

	type target struct {
		name    string
		svc     EventService
		handler EventHandler
		raw     bool
	}
	targets := []target{
		{"AVTransport of " + g.Coordinator.Name, coord.AVTransport, w.onTransport, true},
		{"GroupRenderingControl of " + g.Coordinator.Name, coord.GroupRenderingControl, w.onGroupVolume, false},
		{"ZoneGroupTopology of " + g.Coordinator.Name, coord.ZoneGroupTopology, w.onTopology, false},
	}
	for i, r := range g.Rooms {
		zp, err := r.Player(ctx)
		if err != nil {
			return nil, err
		}
		w.st.Rooms = append(w.st.Rooms, RoomState{Room: r})
		targets = append(targets, target{"RenderingControl of " + r.Name, zp.RenderingControl, w.onRoomVolume(i), false})
	}

	w.heard = make([]bool, len(targets))
	w.failed = make(chan error, len(targets))
	opts := &SubscribeOptions{Timeout: watchTimeout, AutoRenew: true}
	for i, t := range targets {
		sub, err := g.sonos.subscribe(ctx, t.svc, w.heardFrom(i, t.handler), opts, t.raw)
		if err != nil {
			w.stop(ctx)
			return nil, fmt.Errorf("sonos: subscribing to %s: %w", t.name, err)
		}
		w.subs = append(w.subs, sub)
		w.names = append(w.names, t.name)
		go func() {
			<-sub.Done()
			if err := sub.Err(); !errors.Is(err, ErrUnsubscribed) {
				w.failed <- fmt.Errorf("sonos: %s: %w", t.name, err)
			}
		}()
	}
	return w, nil
}

// stop ends every subscription.
func (w *watcher) stop(ctx context.Context) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for _, sub := range w.subs {
		wg.Go(func() { sub.Unsubscribe(ctx) })
	}
	wg.Wait()
}

func (w *watcher) run(ctx context.Context, yield func(*GroupState, error) bool) {
	first := time.NewTimer(watchFirstEvents)
	defer first.Stop()
	waiting := first.C // until every subscription sent its first event

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-w.failed:
			yield(nil, err)
			return
		case <-waiting:
			yield(nil, fmt.Errorf("sonos: no events from %s within %v; check that players can reach the event listener",
				strings.Join(w.silent(), ", "), watchFirstEvents))
			return
		case <-w.changed:
		}

		// Let the rest of a burst arrive.
		select {
		case <-ctx.Done():
			return
		case <-time.After(watchSettle):
		}
		select {
		case <-w.changed:
		default:
		}

		w.mu.Lock()
		err, ready, needPosition := w.err, !slices.Contains(w.heard, false), w.needPosition
		if ready {
			w.needPosition = false
		}
		w.mu.Unlock()
		if err != nil {
			yield(nil, err)
			return
		}
		if !ready {
			continue
		}
		waiting = nil

		if needPosition {
			pos, err := w.coord.AVTransport.GetPositionInfo(ctx, nil)
			if err != nil {
				if ctx.Err() == nil {
					yield(nil, err)
				}
				return
			}
			w.mu.Lock()
			w.st.Position, w.st.At = parseDuration(pos.RelTime), time.Now()
			w.mu.Unlock()
		}
		if !yield(w.snapshot(), nil) {
			return
		}
	}
}

// silent returns the subscriptions that haven't sent an event yet.
func (w *watcher) silent() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var names []string
	for i, heard := range w.heard {
		if !heard {
			names = append(names, w.names[i])
		}
	}
	return names
}

func (w *watcher) snapshot() *GroupState {
	w.mu.Lock()
	defer w.mu.Unlock()
	st := w.st
	st.Rooms = slices.Clone(w.st.Rooms)
	return &st
}

// heardFrom wraps the handler of subscription i to note that it has sent
// an event and that the state changed.
func (w *watcher) heardFrom(i int, handler EventHandler) EventHandler {
	return func(evt any) {
		w.mu.Lock()
		handler(evt)
		w.heard[i] = true
		w.mu.Unlock()
		select {
		case w.changed <- struct{}{}:
		default:
		}
	}
}

// avtChange holds the AVTransport variables Watch follows; a nil field
// wasn't in the event.
type avtChange struct {
	InstanceID struct {
		TransportState       *Val
		CurrentPlayMode      *Val
		CurrentTrack         *Val
		CurrentTrackURI      *Val
		CurrentTrackDuration *Val
		CurrentTrackMetaData *Val
		AVTransportURI       *Val
	}
}

// The handlers below run with w.mu held.

func (w *watcher) onTransport(evt any) {
	lc, ok := evt.(avt.LastChange)
	if !ok {
		return
	}
	var c avtChange
	if err := xml.Unmarshal([]byte(lc), &c); err != nil {
		w.group.sonos.logger.Warn("sonos: dropping malformed AVTransport event", "err", err)
		return
	}
	v := c.InstanceID
	set := func(dst *string, val *Val) {
		if val != nil {
			*dst = val.Value
		}
	}
	set(&w.transportURI, v.AVTransportURI)
	set(&w.track, v.CurrentTrack)
	if strings.HasPrefix(w.transportURI, "x-rincon:") {
		// The coordinator joined another group; the topology event saying
		// so may come later.
		w.err = ErrGroupChanged
		return
	}
	set(&w.trackURI, v.CurrentTrackURI)
	set(&w.trackMeta, v.CurrentTrackMetaData)
	set(&w.trackDuration, v.CurrentTrackDuration)
	if v.TransportState != nil {
		w.st.State = PlaybackState(v.TransportState.Value)
	}
	if v.CurrentPlayMode != nil {
		for m, name := range playModes {
			if string(name) == v.CurrentPlayMode.Value {
				w.st.PlayMode = m
			}
		}
	}
	w.st.Track = trackFrom(w.trackURI, w.trackMeta, w.trackDuration, w.coord.location)
	w.st.QueuePosition = 0
	if strings.HasPrefix(w.transportURI, "x-rincon-queue:") {
		w.st.QueuePosition, _ = strconv.Atoi(w.track)
	}
	w.needPosition = true
}

func (w *watcher) onGroupVolume(evt any) {
	switch e := evt.(type) {
	case rcg.GroupVolume:
		w.st.Volume = int(e)
	case rcg.GroupMute:
		w.st.Muted = bool(e)
	}
}

func (w *watcher) onRoomVolume(i int) EventHandler {
	return func(evt any) {
		lc, ok := evt.(RenderingControlLastChange)
		if !ok {
			return
		}
		r := &w.st.Rooms[i]
		for _, v := range lc.InstanceID.Volume {
			if v.Channel == "Master" {
				r.Volume, _ = strconv.Atoi(v.Value)
			}
		}
		for _, v := range lc.InstanceID.Mute {
			if v.Channel == "Master" {
				r.Muted = v.Value == "1" || v.Value == "true"
			}
		}
	}
}

func (w *watcher) onTopology(evt any) {
	state, ok := evt.(ZoneGroupTopologyZoneGroupState)
	if !ok || len(state.ZoneGroups) == 0 {
		return
	}
	if !sameGroup(w.group, ZoneGroupState(state)) {
		w.err = ErrGroupChanged
	}
}

// sameGroup reports whether the topology still has g: the same coordinator
// with the same rooms.
func sameGroup(g *Group, state ZoneGroupState) bool {
	for _, zg := range state.ZoneGroups {
		if zg.Coordinator != g.Coordinator.UUID {
			continue
		}
		var rooms []string
		for _, m := range zg.ZoneGroupMember {
			if m.Invisible != "1" {
				rooms = append(rooms, m.UUID)
			}
		}
		return len(rooms) == len(g.Rooms) && !slices.ContainsFunc(g.Rooms, func(r *Room) bool {
			return !slices.Contains(rooms, r.UUID)
		})
	}
	return false
}
