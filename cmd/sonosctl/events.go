package main

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/caglar10ur/sonos"
)

// cmdWatch prints a line whenever what a room plays, or how loud, changes.
// It follows the room when it is grouped differently.
func cmdWatch(ctx context.Context, c *cli, args []string) error {
	for {
		r, err := c.room(ctx, args[0])
		if err != nil {
			return err
		}
		var last string
		for st, err := range r.Group().Watch(ctx) {
			if err != nil {
				if errors.Is(err, sonos.ErrGroupChanged) {
					break // find the room's new group
				}
				return err
			}
			line := fmt.Sprintf("%-12s  %s", st.Group.Name(), describe(&st.NowPlaying))
			if st.Track != nil && st.Track.Duration > 0 {
				line += fmt.Sprintf("  %s/%s", clock(st.Position), clock(st.Track.Duration))
			}
			line += fmt.Sprintf("  volume %d", st.Volume)
			if line != last {
				c.printf("%s  %s\n", time.Now().Format(time.TimeOnly), line)
				last = line
			}
		}
		if ctx.Err() != nil {
			return nil
		}
	}
}

// cmdEvents prints every event of every service of a room's player, until
// Ctrl-C or until a subscription is lost.
func cmdEvents(ctx context.Context, c *cli, args []string) error {
	r, err := c.room(ctx, args[0])
	if err != nil {
		return err
	}
	zp, err := r.Player(ctx)
	if err != nil {
		return err
	}
	services := []sonos.EventService{
		zp.AlarmClock, zp.AudioIn, zp.AVTransport, zp.ConnectionManager,
		zp.ContentDirectory, zp.DeviceProperties, zp.GroupManagement,
		zp.GroupRenderingControl, zp.MusicServices, zp.Queue, zp.RenderingControl,
		zp.SystemProperties, zp.VirtualLineIn, zp.ZoneGroupTopology,
	}
	lost := make(chan error, len(services))
	subscribed := 0
	for _, svc := range services {
		sub, err := c.sonos.Subscribe(ctx, svc, func(evt any) { c.printEvent(evt) }, &sonos.SubscribeOptions{AutoRenew: true})
		if err != nil {
			// Not every player has every service, e.g. AudioIn without a
			// line-in.
			fmt.Fprintf(c.errOut, "sonosctl: skipping %T: %v\n", svc, err)
			continue
		}
		subscribed++
		go func() {
			<-sub.Done()
			if err := sub.Err(); errors.Is(err, sonos.ErrSubscriptionLost) {
				lost <- fmt.Errorf("%T: %w", svc, err)
			}
		}()
	}
	if subscribed == 0 {
		return fmt.Errorf("%s's player accepted no subscription", r.Name)
	}
	select {
	case <-ctx.Done():
		return nil
	case err := <-lost:
		return err
	}
}

// printEvent prints an event. LastChange and ZoneGroupState payloads arrive
// decoded; every other state variable arrives as its generated type, e.g.
// deviceproperties.ZoneName.
func (c *cli) printEvent(evt any) {
	switch e := evt.(type) {
	case fmt.Stringer:
		c.printf("### %T\n%s\n", e, e)
	default:
		c.printf("### %T: %v\n", e, e)
	}
}

// announceStart is how long announce waits for the URI to start playing.
const announceStart = 15 * time.Second

// cmdAnnounce plays a URI in a room's group, optionally at another volume,
// waits for it to end and puts back what was playing.
func cmdAnnounce(ctx context.Context, c *cli, args []string) (err error) {
	r, err := c.room(ctx, args[0])
	if err != nil {
		return err
	}
	volume := -1
	if len(args) == 3 {
		if volume, err = strconv.Atoi(args[2]); err != nil || volume < 0 || volume > 100 {
			return fmt.Errorf("invalid volume %q: want 0 to 100", args[2])
		}
	}

	g := r.Group()
	snap, err := g.Snapshot(ctx)
	if err != nil {
		return err
	}
	// Put things back even after Ctrl-C.
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if rerr := snap.Restore(ctx); rerr != nil {
			err = errors.Join(err, fmt.Errorf("restoring: %w", rerr))
		}
	}()

	// Watch from before the announcement, so that a short one can't be over
	// before the first state arrives.
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	next, stop := iter.Pull2(g.Watch(watchCtx))
	defer stop()
	st, err, ok := next()
	if !ok {
		return nil // interrupted
	} else if err != nil {
		return err
	}
	a := announcement{before: trackURI(st)}

	if volume >= 0 {
		for _, room := range g.Rooms {
			if err := room.SetVolume(ctx, volume); err != nil {
				return err
			}
		}
	}
	if err := g.SetURI(ctx, args[1], ""); err != nil {
		return err
	}
	if err := g.Play(ctx); err != nil {
		return err
	}

	var timedOut atomic.Bool
	timer := time.AfterFunc(announceStart, func() { timedOut.Store(true); cancel() })
	defer timer.Stop()
	for {
		st, err, ok := next()
		switch {
		case !ok && timedOut.Load() && !a.started:
			return fmt.Errorf("%s didn't start playing within %v", args[1], announceStart)
		case !ok:
			return nil // interrupted
		case err != nil:
			return err
		case a.done(st):
			return nil
		case a.started:
			timer.Stop()
		}
	}
}

// announcement follows the states of a group playing an announcement.
type announcement struct {
	before  string // the track URI before it
	started bool
	playing string // its track URI, as the player reports it
}

// done reports whether the announcement is over, given the group's next
// state.
func (a *announcement) done(st *sonos.GroupState) bool {
	uri := trackURI(st)
	switch {
	case a.started:
		return uri != a.playing || st.State != sonos.Playing && st.State != sonos.Transitioning
	case st.State == sonos.Playing && uri != a.before:
		a.started, a.playing = true, uri
	case st.State == sonos.Stopped && uri != a.before:
		return true // over, or failed to play, before it was seen playing
	}
	return false
}

func trackURI(st *sonos.GroupState) string {
	if st.Track == nil {
		return ""
	}
	return st.Track.URI
}
