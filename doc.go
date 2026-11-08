// Package sonos discovers, controls and subscribes to events of Sonos
// players on the local network over their UPnP API.
//
// A household is made of rooms, and rooms play in groups: a room that isn't
// grouped with others is a group of its own. Room-level settings such as
// volume are on Room; playback, the queue and group volume are on Group,
// which sends them to the group's coordinator.
//
//	s, err := sonos.New()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer s.Close()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	kitchen, err := s.FindRoom(ctx, "Kitchen")
//	if err != nil {
//		log.Fatal(err)
//	}
//	err = kitchen.SetVolume(ctx, 30)
//	np, err := kitchen.Group().NowPlaying(ctx) // np.State, np.Track.Title, np.Position
//	err = kitchen.Group().Next(ctx)
//
// Groups also seek, edit their queue, shuffle and repeat, and save and
// restore what they play, e.g. around an announcement:
//
//	g := kitchen.Group()
//	snap, err := g.Snapshot(ctx)
//	err = g.SetURI(ctx, "https://example.com/doorbell.mp3", "")
//	err = g.Play(ctx)
//	// ... wait for it to end
//	err = snap.Restore(ctx)
//
// Rooms join and leave groups with Room.Join and Room.Leave.
//
// Household returns every room and group at once. It is a snapshot: call
// Refresh after rooms are grouped differently.
//
// Watch reports a group's state, merged from its players' events, now and
// after every change:
//
//	for st, err := range kitchen.Group().Watch(ctx) {
//		if err != nil {
//			break // e.g. ErrGroupChanged: refresh and watch the new group
//		}
//		fmt.Println(st.State, st.Track, st.Volume)
//	}
//
// For the raw events of a service, subscribe to it until Close:
//
//	zp, err := kitchen.Group().Player(ctx)
//	sub, err := s.Subscribe(ctx, zp.AVTransport, func(evt any) {
//		if lc, ok := evt.(sonos.AVTransportLastChange); ok {
//			log.Println(lc.InstanceID.TransportState.Value)
//		}
//	}, &sonos.SubscribeOptions{AutoRenew: true})
//
// Done reports when a subscription ends, so there is no need to poll the
// device to know events are still coming:
//
//	<-sub.Done()
//	if errors.Is(sub.Err(), sonos.ErrSubscriptionLost) {
//		// The device stopped answering: find it again and resubscribe.
//	}
//
// Every UPnP action is available through the generated clients of a
// ZonePlayer, e.g. zp.AVTransport.Seek. They return *soap.UPnPError when
// the device rejects a call; soap.IsCode checks its code.
//
// The sonosctl command (cmd/sonosctl) uses most of the package, and its
// README has notes on what to check when nothing is discovered or no events
// arrive.
package sonos

//go:generate bash cmd/makeservices/makeallservices.sh
