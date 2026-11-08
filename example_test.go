package sonos_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/caglar10ur/sonos"
	avt "github.com/caglar10ur/sonos/services/avtransport"
	"github.com/caglar10ur/sonos/soap"
)

// Turn a room up and skip to the next track. Volume belongs to the room,
// playback to its group, even when the room is grouped with others.
func Example() {
	s, err := sonos.New()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kitchen, err := s.FindRoom(ctx, "Kitchen")
	if err != nil {
		log.Fatal(err)
	}
	vol, err := kitchen.Volume(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := kitchen.SetVolume(ctx, min(vol+10, 100)); err != nil {
		log.Fatal(err)
	}
	if err := kitchen.Group().Next(ctx); err != nil {
		log.Fatal(err)
	}
}

// List what every group is playing.
func ExampleSonos_Household() {
	s, err := sonos.New()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h, err := s.Household(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, g := range h.Groups {
		np, err := g.NowPlaying(ctx)
		if err != nil {
			log.Fatal(err)
		}
		if np.Track == nil {
			fmt.Printf("%s: %s\n", g.Name(), np.State)
			continue
		}
		fmt.Printf("%s: %s %s - %s (%v of %v)\n", g.Name(), np.State,
			np.Track.Artist, np.Track.Title, np.Position, np.Track.Duration)
	}
}

// Search reports each group coordinator that answers until ctx is done.
// Household and FindRoom are simpler when you want rooms.
func ExampleSonos_Search() {
	s, err := sonos.New()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.Search(ctx, func(zp *sonos.ZonePlayer) {
		fmt.Println(zp.RoomName(), zp.ModelName())
	})
	if err != nil {
		log.Fatal(err)
	}
	<-ctx.Done()
}

// Print what a room's group plays after every change, until Ctrl-C.
func ExampleGroup_Watch() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	s, err := sonos.New()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	room, err := s.FindRoom(ctx, "Kitchen")
	if err != nil {
		log.Fatal(err)
	}
	for st, err := range room.Group().Watch(ctx) {
		if errors.Is(err, sonos.ErrGroupChanged) {
			log.Print("Kitchen was grouped differently; find it again to follow it")
			return
		} else if err != nil {
			log.Fatal(err)
		}
		if st.Track != nil {
			fmt.Println(st.State, st.Track.Artist, "-", st.Track.Title, "at volume", st.Volume)
		}
	}
}

// Print track changes until Ctrl-C, and stop if the player goes away.
func ExampleSonos_Subscribe() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	s, err := sonos.New()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close() // also unsubscribes

	room, err := s.FindRoom(ctx, "Kitchen")
	if err != nil {
		log.Fatal(err)
	}
	// Playback events come from the coordinator of the room's group.
	zp, err := room.Group().Player(ctx)
	if err != nil {
		log.Fatal(err)
	}

	sub, err := s.Subscribe(ctx, zp.AVTransport, func(evt any) {
		if lc, ok := evt.(sonos.AVTransportLastChange); ok {
			fmt.Println(lc.InstanceID.TransportState.Value, lc.InstanceID.CurrentTrackURI.Value)
		}
	}, &sonos.SubscribeOptions{AutoRenew: true})
	if err != nil {
		log.Fatal(err)
	}

	select {
	case <-ctx.Done():
	case <-sub.Done():
		// With AutoRenew, a subscription only ends early when the player
		// stops answering for longer than the subscription's timeout.
		if errors.Is(sub.Err(), sonos.ErrSubscriptionLost) {
			log.Print("player went away: ", sub.Err())
		}
	}
}

// Every UPnP action is available through the generated service clients of
// a player. When a player rejects a call, the error is a *soap.UPnPError
// carrying the UPnP error code.
func Example_upnpError() {
	s, err := sonos.New()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	room, err := s.FindRoom(ctx, "Kitchen")
	if errors.Is(err, sonos.ErrRoomNotFound) {
		log.Fatal("no such room")
	} else if err != nil {
		log.Fatal(err)
	}
	zp, err := room.Group().Player(ctx)
	if err != nil {
		log.Fatal(err)
	}

	_, err = zp.AVTransport.Seek(ctx, &avt.SeekArgs{Unit: avt.SeekModeTrackNR, Target: "1"})
	switch {
	case soap.IsCode(err, soap.TransitionNotAvailable):
		fmt.Println("nothing to seek in")
	case err != nil:
		log.Fatal(err)
	}
}
