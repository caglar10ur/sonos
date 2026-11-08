package sonos

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeHousehold serves one player per room. Every player answers
// GetZoneGroupState with the same topology, built from groups: each group
// is a list of room names, the first being its coordinator.
type fakeHousehold struct {
	players map[string]*httptest.Server // room -> player
	uuid    map[string]string           // room -> UUID

	mu           sync.Mutex
	callbacks    map[string]string // "room event path" -> CALLBACK URL
	unsubscribes int
	relTime      string // GetPositionInfo's RelTime
	stalled      string // a room whose player doesn't answer SOAP calls
	topologies   int    // GetZoneGroupState requests answered
}

func (h *fakeHousehold) isStalled(room string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stalled == room
}

// callback returns the callback URL subscribed to a room's event path, or
// "" if there is none yet.
func (h *fakeHousehold) callback(room, path string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.callbacks[room+" "+path]
}

func newFakeHousehold(t *testing.T, groups ...[]string) *fakeHousehold {
	t.Helper()
	h := &fakeHousehold{players: map[string]*httptest.Server{}, uuid: map[string]string{}, callbacks: map[string]string{}, relTime: "0:00:00"}
	var topology atomic.Value // set once every player has an address

	for i, room := range flatten(groups) {
		uuid := fmt.Sprintf("RINCON_%012d01400", i+1)
		desc := strings.NewReplacer("RINCON_000E58CDCA4001400", uuid, "<roomName>Kitchen</roomName>", "<roomName>"+room+"</roomName>").
			Replace(mockDeviceDescription)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet:
				io.WriteString(w, desc)
				return
			case r.Method == "SUBSCRIBE":
				h.mu.Lock()
				h.callbacks[room+" "+r.URL.Path] = strings.Trim(r.Header.Get("CALLBACK"), "<>")
				h.mu.Unlock()
				w.Header().Set("SID", "uuid:"+room+r.URL.Path)
				w.Header().Set("TIMEOUT", "Second-300")
				return
			case r.Method == "UNSUBSCRIBE":
				h.mu.Lock()
				h.unsubscribes++
				h.mu.Unlock()
				return
			case h.isStalled(room):
				// Read the request first: only then does the server notice
				// the client giving up.
				io.Copy(io.Discard, r.Body)
				<-r.Context().Done()
				return
			case strings.Contains(r.Header.Get("SOAPAction"), "#GetPositionInfo"):
				h.mu.Lock()
				rel := h.relTime
				h.mu.Unlock()
				io.WriteString(w, soapEnvelope("AVTransport", "GetPositionInfo", "<RelTime>"+rel+"</RelTime>"))
				return
			}
			h.mu.Lock()
			h.topologies++
			h.mu.Unlock()
			var escaped strings.Builder
			if err := xmlEscape(&escaped, topology.Load().(string)); err != nil {
				t.Error(err)
			}
			io.WriteString(w, soapEnvelope("ZoneGroupTopology", "GetZoneGroupState", "<ZoneGroupState>"+escaped.String()+"</ZoneGroupState>"))
		}))
		t.Cleanup(srv.Close)
		h.players[room], h.uuid[room] = srv, uuid
	}

	topology.Store(h.topology(groups...))
	return h
}

// topology returns a ZoneGroupState of the household's rooms grouped as
// given.
func (h *fakeHousehold) topology(groups ...[]string) string {
	var b strings.Builder
	b.WriteString("<ZoneGroupState><ZoneGroups>")
	for _, g := range groups {
		fmt.Fprintf(&b, `<ZoneGroup Coordinator="%s" ID="%s:1">`, h.uuid[g[0]], h.uuid[g[0]])
		for _, room := range g {
			fmt.Fprintf(&b, `<ZoneGroupMember UUID="%s" Location="%s/xml/device_description.xml" ZoneName="%s"/>`,
				h.uuid[room], h.players[room].URL, room)
		}
		b.WriteString("</ZoneGroup>")
	}
	b.WriteString("</ZoneGroups></ZoneGroupState>")
	return b.String()
}

func flatten(groups [][]string) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

func xmlEscape(b *strings.Builder, s string) error {
	_, err := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").WriteString(b, s)
	return err
}

func (h *fakeHousehold) player(t *testing.T, room string) *ZonePlayer {
	t.Helper()
	zp, err := playerAt(t.Context(), h.players[room].URL+"/xml/device_description.xml", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	return zp
}

// A household snapshot must list every room, grouped or not, with its
// group and coordinator, whichever player it was read from.
func TestHousehold(t *testing.T) {
	h := newFakeHousehold(t, []string{"Living Room", "Kitchen"}, []string{"Office"})
	hh, err := newTestSonos(t).householdFrom(t.Context(), h.player(t, "Kitchen"))
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, r := range hh.Rooms {
		names = append(names, r.Name)
	}
	if fmt.Sprint(names) != "[Living Room Kitchen Office]" {
		t.Errorf("rooms = %v", names)
	}
	if len(hh.Groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(hh.Groups))
	}

	kitchen := hh.Room("Kitchen")
	if kitchen == nil || kitchen.UUID != h.uuid["Kitchen"] {
		t.Fatalf("Room(Kitchen) = %+v", kitchen)
	}
	g := kitchen.Group()
	if g.Coordinator.Name != "Living Room" || g.Name() != "Living Room + 1" || g.Rooms[0] != g.Coordinator {
		t.Errorf("Kitchen's group: coordinator %s, name %q", g.Coordinator.Name, g.Name())
	}
	if office := hh.Room("Office"); office.Group().Coordinator != office || office.Group().Name() != "Office" {
		t.Errorf("Office should be its own group's coordinator")
	}
	if hh.Room("Garage") != nil {
		t.Error("Room(Garage) should be nil")
	}

	// The coordinator's player is the one group actions go to.
	zp, err := g.Player(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if zp.RoomName() != "Living Room" {
		t.Errorf("group player is %s, want Living Room", zp.RoomName())
	}

	// Refresh reuses players already set up.
	fresh, err := hh.Refresh(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	again, err := fresh.Room("Kitchen").Group().Player(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if again != zp {
		t.Error("Refresh set the coordinator's player up again")
	}
}

func TestSearchAfterClose(t *testing.T) {
	s := newTestSonos(t)
	s.Close()
	if err := s.Search(t.Context(), func(*ZonePlayer) {}); !errors.Is(err, ErrClosed) {
		t.Errorf("Search after Close = %v, want ErrClosed", err)
	}
	if _, err := s.Household(t.Context()); !errors.Is(err, ErrClosed) {
		t.Errorf("Household after Close = %v, want ErrClosed", err)
	}
}

// A player that stops answering must not use up Refresh's deadline: the
// other players are asked in time.
func TestRefreshSkipsStalledPlayer(t *testing.T) {
	h := newFakeHousehold(t, []string{"Kitchen"}, []string{"Office"})
	hh, err := newTestSonos(t).householdFrom(t.Context(), h.player(t, "Kitchen"))
	if err != nil {
		t.Fatal(err)
	}
	// Set Office's player up, so Refresh knows another player to ask.
	if _, err := hh.Room("Office").Player(t.Context()); err != nil {
		t.Fatal(err)
	}
	h.mu.Lock()
	h.stalled = "Kitchen"
	h.mu.Unlock()

	ctx, cancel := context.WithTimeout(t.Context(), refreshAttempt+2*time.Second)
	defer cancel()
	start := time.Now()
	fresh, err := hh.Refresh(ctx)
	if err != nil {
		t.Fatalf("Refresh = %v, want it to ask Office", err)
	}
	if len(fresh.Rooms) != 2 || time.Since(start) > refreshAttempt+time.Second {
		t.Errorf("got %d rooms after %v", len(fresh.Rooms), time.Since(start))
	}
}

// Search sets players up in parallel but must call fn one at a time.
func TestSearchReportsEveryPlayerSerially(t *testing.T) {
	h := newFakeHousehold(t, []string{"A"}, []string{"B"}, []string{"C"}, []string{"D"})
	s := newTestSonos(t)

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	var (
		mu       sync.Mutex
		rooms    []string
		inFn     atomic.Int32
		parallel atomic.Bool
	)
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.readSearchResponses(t.Context(), conn, true, func(zp *ZonePlayer) {
			if inFn.Add(1) > 1 {
				parallel.Store(true)
			}
			time.Sleep(20 * time.Millisecond) // widen the window for overlap
			mu.Lock()
			rooms = append(rooms, zp.RoomName())
			mu.Unlock()
			inFn.Add(-1)
		})
	}()

	sender, err := net.DialUDP("udp4", nil, conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()
	for _, room := range []string{"A", "B", "C", "D"} {
		answer := "HTTP/1.1 200 OK\r\nLOCATION: " + h.players[room].URL + "/xml/device_description.xml\r\n\r\n"
		if _, err := sender.Write([]byte(answer)); err != nil {
			t.Fatal(err)
		}
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(rooms)
		mu.Unlock()
		if n == 4 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	conn.Close()
	<-done

	slices.Sort(rooms)
	if fmt.Sprint(rooms) != "[A B C D]" {
		t.Errorf("reported %v, want [A B C D]", rooms)
	}
	if parallel.Load() {
		t.Error("fn was called concurrently")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topologies != 1 {
		t.Errorf("fetched the topology %d times, want once per search", h.topologies)
	}
}

func TestTimeoutHeaderRoundsUp(t *testing.T) {
	for d, want := range map[time.Duration]string{
		time.Hour:               "Second-3600",
		90 * time.Second:        "Second-90",
		1500 * time.Millisecond: "Second-2",
		500 * time.Millisecond:  "Second-1",
		0:                       "Second-1",
	} {
		if got := timeoutHeader(d); got != want {
			t.Errorf("timeoutHeader(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestDeviceIconURL(t *testing.T) {
	var d DeviceDescription
	err := xml.Unmarshal([]byte(`<root><device><iconList><icon><id>0</id><mimetype>image/png</mimetype>
<url>/img/icon-S41.png</url></icon></iconList></device></root>`), &d)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Device.Icons) != 1 || d.Device.Icons[0].URL != "/img/icon-S41.png" {
		t.Errorf("icons = %+v", d.Device.Icons)
	}
}
