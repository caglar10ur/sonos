package sonos

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// call is one SOAP action a recordingPlayer received.
type call struct {
	action string            // e.g. "Seek"
	args   map[string]string // argument name -> value
}

// recordingPlayer returns a player that records every SOAP action it
// receives and answers it with responses[action], or an empty response.
func recordingPlayer(t *testing.T, responses map[string]string) (*ZonePlayer, func() []call) {
	t.Helper()
	var (
		mu    sync.Mutex
		calls []call
	)
	argRE := regexp.MustCompile(`<(\w+)>([^<]*)</\w+>`)
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		soapAction := strings.Trim(req.Header.Get("SOAPAction"), `"`)
		if soapAction == "" {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(mockDeviceDescription)), Header: http.Header{}}, nil
		}
		urn, action, _ := strings.Cut(soapAction, "#")
		body, _ := io.ReadAll(req.Body)
		c := call{action: action, args: map[string]string{}}
		for _, m := range argRE.FindAllSubmatch(body, -1) {
			c.args[string(m[1])] = string(m[2])
		}
		mu.Lock()
		calls = append(calls, c)
		mu.Unlock()
		service := strings.TrimSuffix(strings.TrimPrefix(urn, "urn:schemas-upnp-org:service:"), ":1")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(soapEnvelope(service, action, responses[action]))),
			Header:     http.Header{},
		}, nil
	})
	zp := NewMockZonePlayerWithTransport(t, rt)
	return zp, func() []call {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(calls)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// summary renders calls as "Action(Arg=Value ...)" for the given args.
func summary(calls []call, args ...string) []string {
	var out []string
	for _, c := range calls {
		var parts []string
		for _, a := range args {
			if v, ok := c.args[a]; ok {
				parts = append(parts, a+"="+v)
			}
		}
		out = append(out, c.action+"("+strings.Join(parts, " ")+")")
	}
	return out
}

func checkCalls(t *testing.T, got []call, want []string, args ...string) {
	t.Helper()
	if s := summary(got, args...); strings.Join(s, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls:\n  %s\nwant:\n  %s", strings.Join(s, "\n  "), strings.Join(want, "\n  "))
	}
}

// topologyResponse is a GetZoneGroupState response holding the given
// ZoneGroup elements.
func topologyResponse(t *testing.T, groups string) string {
	return "<ZoneGroupState>" + escape(t, "<ZoneGroupState><ZoneGroups>"+groups+"</ZoneGroups></ZoneGroupState>") + "</ZoneGroupState>"
}

const (
	kitchenMember = `<ZoneGroupMember UUID="RINCON_000E58CDCA4001400" ZoneName="Kitchen"/>`
	garageMember  = `<ZoneGroupMember UUID="RINCON_OTHER" ZoneName="Garage"/>`
)

// Join and Leave must return once the player reports the new grouping, so
// that a Refresh right after sees it.
func TestJoinLeave(t *testing.T) {
	zp, calls := recordingPlayer(t, map[string]string{
		"GetZoneGroupState": topologyResponse(t, `<ZoneGroup Coordinator="RINCON_OTHER" ID="RINCON_OTHER:1">`+garageMember+kitchenMember+`</ZoneGroup>`),
	})
	room := roomOf(zp)
	garage := &Group{Coordinator: &Room{Name: "Garage", UUID: "RINCON_OTHER"}}
	if err := room.Join(t.Context(), garage); err != nil {
		t.Fatal(err)
	}
	if err := room.Join(t.Context(), room.Group()); err != nil { // its own: nothing to do
		t.Fatal(err)
	}
	checkCalls(t, calls(), []string{
		"SetAVTransportURI(CurrentURI=x-rincon:RINCON_OTHER)",
		"GetZoneGroupState()",
	}, "CurrentURI")

	zp, calls = recordingPlayer(t, map[string]string{
		"GetZoneGroupState": topologyResponse(t,
			`<ZoneGroup Coordinator="RINCON_000E58CDCA4001400" ID="A:1">`+kitchenMember+`</ZoneGroup>`+
				`<ZoneGroup Coordinator="RINCON_OTHER" ID="B:1">`+garageMember+`</ZoneGroup>`),
	})
	if err := roomOf(zp).Leave(t.Context()); err != nil {
		t.Fatal(err)
	}
	checkCalls(t, calls(), []string{"BecomeCoordinatorOfStandaloneGroup()", "GetZoneGroupState()"})

	// The player still reports the room as grouped: Leave keeps waiting.
	zp, _ = recordingPlayer(t, map[string]string{
		"GetZoneGroupState": topologyResponse(t, `<ZoneGroup Coordinator="RINCON_000E58CDCA4001400" ID="A:1">`+kitchenMember+garageMember+`</ZoneGroup>`),
	})
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	if err := roomOf(zp).Leave(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Leave = %v; want it to wait for the topology", err)
	}
}

func TestSeek(t *testing.T) {
	zp, calls := recordingPlayer(t, nil)
	g := groupOf(zp)
	if err := g.Seek(t.Context(), time.Hour+2*time.Minute+3500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := g.SeekTrack(t.Context(), 7); err != nil {
		t.Fatal(err)
	}
	if g.Seek(t.Context(), -time.Second) == nil || g.SeekTrack(t.Context(), 0) == nil {
		t.Error("out of range seeks should fail")
	}
	checkCalls(t, calls(), []string{
		"Seek(Unit=REL_TIME Target=1:02:03)",
		"Seek(Unit=TRACK_NR Target=7)",
	}, "Unit", "Target")
}

func TestQueueEditing(t *testing.T) {
	zp, calls := recordingPlayer(t, map[string]string{
		"AddURIToQueue": "<FirstTrackNumberEnqueued>4</FirstTrackNumberEnqueued><NumTracksAdded>1</NumTracksAdded><NewQueueLength>4</NewQueueLength>",
	})
	g := groupOf(zp)
	pos, err := g.AddToQueue(t.Context(), "x-file-cifs://nas/a.mp3", "", 0)
	if err != nil || pos != 4 {
		t.Fatalf("AddToQueue = %d, %v; want 4", pos, err)
	}
	if _, err := g.AddToQueue(t.Context(), "x-file-cifs://nas/b.mp3", "", 2); err != nil {
		t.Fatal(err)
	}
	if err := g.RemoveFromQueue(t.Context(), 3); err != nil {
		t.Fatal(err)
	}
	if err := g.ClearQueue(t.Context()); err != nil {
		t.Fatal(err)
	}
	checkCalls(t, calls(), []string{
		"AddURIToQueue(EnqueuedURI=x-file-cifs://nas/a.mp3 DesiredFirstTrackNumberEnqueued=0)",
		"AddURIToQueue(EnqueuedURI=x-file-cifs://nas/b.mp3 DesiredFirstTrackNumberEnqueued=2)",
		"RemoveTrackFromQueue(ObjectID=Q:0/3)",
		"RemoveAllTracksFromQueue()",
	}, "EnqueuedURI", "DesiredFirstTrackNumberEnqueued", "ObjectID")
}

func TestPlayMode(t *testing.T) {
	zp, calls := recordingPlayer(t, map[string]string{
		"GetTransportSettings": "<PlayMode>SHUFFLE_REPEAT_ONE</PlayMode><RecQualityMode>NOT_IMPLEMENTED</RecQualityMode>",
	})
	g := groupOf(zp)
	mode, err := g.PlayMode(t.Context())
	if err != nil || mode != (PlayMode{Shuffle: true, Repeat: RepeatOne}) {
		t.Fatalf("PlayMode = %+v, %v", mode, err)
	}
	if err := g.SetPlayMode(t.Context(), PlayMode{Repeat: RepeatAll}); err != nil {
		t.Fatal(err)
	}
	if g.SetPlayMode(t.Context(), PlayMode{Repeat: 7}) == nil {
		t.Error("SetPlayMode accepted an invalid mode")
	}
	checkCalls(t, calls(), []string{"GetTransportSettings()", "SetPlayMode(NewPlayMode=REPEAT_ALL)"}, "NewPlayMode")
}

// Restore must put back the queue, track, position, play mode, volume and
// mute, and resume playback.
func TestSnapshotRestore(t *testing.T) {
	zp, calls := recordingPlayer(t, map[string]string{
		"GetMediaInfo":         "<CurrentURI>x-rincon-queue:RINCON_000E58CDCA4001400#0</CurrentURI><CurrentURIMetaData></CurrentURIMetaData>",
		"GetTransportInfo":     "<CurrentTransportState>PLAYING</CurrentTransportState>",
		"GetPositionInfo":      "<Track>5</Track><TrackDuration>0:03:00</TrackDuration><RelTime>0:01:30</RelTime>",
		"GetTransportSettings": "<PlayMode>SHUFFLE</PlayMode>",
		"GetVolume":            "<CurrentVolume>30</CurrentVolume>",
		"GetMute":              "<CurrentMute>1</CurrentMute>",
	})
	g := groupOf(zp)
	snap, err := g.Snapshot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	before := len(calls())
	if err := snap.Restore(t.Context()); err != nil {
		t.Fatal(err)
	}
	checkCalls(t, calls()[before:], []string{
		"SetAVTransportURI(CurrentURI=x-rincon-queue:RINCON_000E58CDCA4001400#0)",
		"SetPlayMode(NewPlayMode=SHUFFLE)",
		"Seek(Unit=TRACK_NR Target=5)",
		"Seek(Unit=REL_TIME Target=0:01:30)",
		"SetVolume(DesiredVolume=30)",
		"SetMute(DesiredMute=true)",
		"Play()",
	}, "CurrentURI", "NewPlayMode", "Unit", "Target", "DesiredVolume", "DesiredMute")
}

// A stream isn't in the queue: Restore sets it back without seeking, and
// doesn't play if the group was stopped.
func TestSnapshotRestoreStream(t *testing.T) {
	zp, calls := recordingPlayer(t, map[string]string{
		"GetMediaInfo":     "<CurrentURI>x-rincon-mp3radio://radio.example/stream</CurrentURI><CurrentURIMetaData>meta</CurrentURIMetaData>",
		"GetTransportInfo": "<CurrentTransportState>STOPPED</CurrentTransportState>",
		"GetVolume":        "<CurrentVolume>12</CurrentVolume>",
		"GetMute":          "<CurrentMute>0</CurrentMute>",
	})
	snap, err := groupOf(zp).Snapshot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	before := len(calls())
	if err := snap.Restore(t.Context()); err != nil {
		t.Fatal(err)
	}
	checkCalls(t, calls()[before:], []string{
		"SetAVTransportURI(CurrentURI=x-rincon-mp3radio://radio.example/stream CurrentURIMetaData=meta)",
		"SetVolume(DesiredVolume=12)",
		"SetMute(DesiredMute=false)",
	}, "CurrentURI", "CurrentURIMetaData", "DesiredVolume", "DesiredMute")
}
