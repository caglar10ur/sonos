package sonos

import (
	"errors"
	"iter"
	"strings"
	"testing"
	"time"
)

const (
	avtEvents = "/MediaRenderer/AVTransport/Event"
	grcEvents = "/MediaRenderer/GroupRenderingControl/Event"
	rcEvents  = "/MediaRenderer/RenderingControl/Event"
	zgtEvents = "/ZoneGroupTopology/Event"
)

func escape(t *testing.T, s string) string {
	t.Helper()
	var b strings.Builder
	if err := xmlEscape(&b, s); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// propertySet is a NOTIFY body with one property per name/value pair; the
// values are escaped.
func propertySet(t *testing.T, kv ...string) string {
	var b strings.Builder
	b.WriteString(`<e:propertyset xmlns:e="urn:schemas-upnp-org:event-1-0">`)
	for i := 0; i < len(kv); i += 2 {
		b.WriteString("<e:property><" + kv[i] + ">" + escape(t, kv[i+1]) + "</" + kv[i] + "></e:property>")
	}
	b.WriteString("</e:propertyset>")
	return b.String()
}

// lastChange is a LastChange property whose InstanceID holds the given
// variables, e.g. `<Volume channel="Master" val="20"/>`.
func lastChange(t *testing.T, vars string) string {
	return propertySet(t, "LastChange", `<Event><InstanceID val="0">`+vars+`</InstanceID></Event>`)
}

func trackMeta(title string) string {
	return `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">` +
		`<item id="-1" parentID="-1"><dc:title>` + title + `</dc:title><dc:creator>Artist</dc:creator><upnp:albumArtURI>/getaa?u=1</upnp:albumArtURI></item></DIDL-Lite>`
}

// send waits for a subscription to a room's events and sends it body.
func (h *fakeHousehold) send(t *testing.T, room, path, body string) {
	t.Helper()
	var cb string
	waitFor(t, "subscription to "+room+path, func() bool { cb = h.callback(room, path); return cb != "" })
	if code := notify(t, cb, body); code != 200 {
		t.Fatalf("NOTIFY %s%s: status %d", room, path, code)
	}
}

func next(t *testing.T, pull func() (*GroupState, error, bool)) (*GroupState, error) {
	t.Helper()
	type result struct {
		st  *GroupState
		err error
		ok  bool
	}
	ch := make(chan result, 1)
	go func() {
		st, err, ok := pull()
		ch <- result{st, err, ok}
	}()
	select {
	case r := <-ch:
		if !r.ok {
			t.Fatal("Watch ended")
		}
		return r.st, r.err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Watch")
		return nil, nil
	}
}

func TestWatch(t *testing.T) {
	h := newFakeHousehold(t, []string{"Living Room", "Kitchen"})
	h.relTime = "0:01:10"
	s := newTestSonos(t)
	hh, err := s.householdFrom(t.Context(), h.player(t, "Kitchen"))
	if err != nil {
		t.Fatal(err)
	}
	g := hh.Room("Kitchen").Group()

	pull, stop := iter.Pull2(g.Watch(t.Context()))
	defer stop()

	// Every player sends its whole state when subscribed to.
	go func() {
		h.send(t, "Living Room", avtEvents, lastChange(t,
			`<TransportState val="PLAYING"/><CurrentPlayMode val="SHUFFLE"/><CurrentTrack val="3"/>`+
				`<CurrentTrackURI val="x-file-cifs://nas/a.mp3"/><CurrentTrackDuration val="0:03:00"/>`+
				`<CurrentTrackMetaData val="`+escape(t, trackMeta("First"))+`"/>`+
				`<AVTransportURI val="x-rincon-queue:`+h.uuid["Living Room"]+`#0"/>`))
		h.send(t, "Living Room", grcEvents, propertySet(t, "GroupVolume", "40", "GroupMute", "0"))
		h.send(t, "Living Room", zgtEvents, propertySet(t, "ZoneGroupState", h.topology([]string{"Living Room", "Kitchen"})))
		h.send(t, "Living Room", rcEvents, lastChange(t, `<Volume channel="Master" val="20"/><Mute channel="Master" val="0"/>`))
		h.send(t, "Kitchen", rcEvents, lastChange(t, `<Volume channel="Master" val="30"/><Mute channel="Master" val="1"/>`))
	}()

	st, err := next(t, pull)
	if err != nil {
		t.Fatal(err)
	}
	if st.State != Playing || st.Track == nil || st.Track.Title != "First" || st.Track.Duration != 3*time.Minute || st.QueuePosition != 3 {
		t.Errorf("now playing = %+v, track %+v", st.NowPlaying, st.Track)
	}
	if !strings.HasPrefix(st.Track.AlbumArtURI, h.players["Living Room"].URL) {
		t.Errorf("album art %q not resolved against the coordinator", st.Track.AlbumArtURI)
	}
	if st.Position != 70*time.Second || st.At.IsZero() {
		t.Errorf("position = %v at %v, want 1m10s", st.Position, st.At)
	}
	if st.PlayMode != (PlayMode{Shuffle: true, Repeat: RepeatAll}) || st.Volume != 40 || st.Muted {
		t.Errorf("mode %+v, volume %d, muted %v", st.PlayMode, st.Volume, st.Muted)
	}
	if len(st.Rooms) != 2 || st.Rooms[0].Room.Name != "Living Room" || st.Rooms[0].Volume != 20 || st.Rooms[0].Muted ||
		st.Rooms[1].Volume != 30 || !st.Rooms[1].Muted {
		t.Errorf("rooms = %+v", st.Rooms)
	}
	if got := st.PositionAt(st.At.Add(time.Hour)); got != 3*time.Minute {
		t.Errorf("PositionAt past the end = %v, want the track's duration", got)
	}

	// A change carries only what changed; the rest is kept.
	h.send(t, "Kitchen", rcEvents, lastChange(t, `<Volume channel="Master" val="35"/>`))
	st, err = next(t, pull)
	if err != nil {
		t.Fatal(err)
	}
	if st.Rooms[1].Volume != 35 || !st.Rooms[1].Muted || st.Rooms[0].Volume != 20 || st.Track.Title != "First" {
		t.Errorf("after a volume change: rooms %+v, track %+v", st.Rooms, st.Track)
	}

	// A track change: the position is read again.
	h.mu.Lock()
	h.relTime = "0:00:02"
	h.mu.Unlock()
	h.send(t, "Living Room", avtEvents, lastChange(t,
		`<CurrentTrack val="4"/><CurrentTrackMetaData val="`+escape(t, trackMeta("Second"))+`"/>`))
	st, err = next(t, pull)
	if err != nil {
		t.Fatal(err)
	}
	if st.Track.Title != "Second" || st.QueuePosition != 4 || st.Position != 2*time.Second || st.State != Playing {
		t.Errorf("after a track change: %+v, track %+v", st.NowPlaying, st.Track)
	}

	// Kitchen leaves the group.
	h.send(t, "Living Room", zgtEvents, propertySet(t, "ZoneGroupState", h.topology([]string{"Living Room"}, []string{"Kitchen"})))
	if _, err := next(t, pull); !errors.Is(err, ErrGroupChanged) {
		t.Fatalf("err = %v, want ErrGroupChanged", err)
	}
	if _, _, ok := pull(); ok {
		t.Error("Watch went on after an error")
	}
	waitFor(t, "unsubscribing", func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.unsubscribes == 5
	})
}

// Watch must say so when events don't arrive, e.g. blocked by a firewall,
// instead of waiting forever.
// When the coordinator joins another group, its AVTransport says so before
// the topology does; Watch must not report it as the group's playback.
func TestWatchCoordinatorJoinsAnotherGroup(t *testing.T) {
	h := newFakeHousehold(t, []string{"Kitchen"}, []string{"Garage"})
	hh, err := newTestSonos(t).householdFrom(t.Context(), h.player(t, "Kitchen"))
	if err != nil {
		t.Fatal(err)
	}
	pull, stop := iter.Pull2(hh.Room("Kitchen").Group().Watch(t.Context()))
	defer stop()
	go func() {
		h.send(t, "Kitchen", avtEvents, lastChange(t, `<TransportState val="STOPPED"/>`))
		h.send(t, "Kitchen", grcEvents, propertySet(t, "GroupVolume", "10"))
		h.send(t, "Kitchen", zgtEvents, propertySet(t, "ZoneGroupState", h.topology([]string{"Kitchen"}, []string{"Garage"})))
		h.send(t, "Kitchen", rcEvents, lastChange(t, `<Volume channel="Master" val="10"/>`))
	}()
	if _, err := next(t, pull); err != nil {
		t.Fatal(err)
	}
	h.send(t, "Kitchen", avtEvents, lastChange(t,
		`<TransportState val="PLAYING"/><AVTransportURI val="x-rincon:`+h.uuid["Garage"]+`"/>`))
	if st, err := next(t, pull); !errors.Is(err, ErrGroupChanged) {
		t.Fatalf("got %+v, %v; want ErrGroupChanged", st, err)
	}
}

func TestWatchNoEvents(t *testing.T) {
	defer func(d time.Duration) { watchFirstEvents = d }(watchFirstEvents)
	watchFirstEvents = 200 * time.Millisecond

	h := newFakeHousehold(t, []string{"Kitchen"})
	hh, err := newTestSonos(t).householdFrom(t.Context(), h.player(t, "Kitchen"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range hh.Room("Kitchen").Group().Watch(t.Context()) {
		if err == nil || !strings.Contains(err.Error(), "no events from AVTransport of Kitchen") {
			t.Fatalf("err = %v", err)
		}
		return
	}
	t.Fatal("Watch ended without an error")
}

func TestWatchNeedsSonos(t *testing.T) {
	g := groupOf(NewMockZonePlayer(t, nil))
	for _, err := range g.Watch(t.Context()) {
		if err == nil {
			t.Fatal("Watch without a Sonos should fail")
		}
	}
}
