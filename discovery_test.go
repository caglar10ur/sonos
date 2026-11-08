package sonos

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newFakeDevice serves a device description and a GetZoneGroupState answer
// that makes the device its own coordinator. It counts description fetches.
func newFakeDevice(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var fetches atomic.Int32
	zgs := `&lt;ZoneGroupState&gt;&lt;ZoneGroups&gt;&lt;ZoneGroup Coordinator=&quot;RINCON_000E58CDCA4001400&quot; ID=&quot;RINCON_000E58CDCA4001400:1&quot;&gt;&lt;/ZoneGroup&gt;&lt;/ZoneGroups&gt;&lt;/ZoneGroupState&gt;`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fetches.Add(1)
			io.WriteString(w, mockDeviceDescription)
			return
		}
		io.WriteString(w, soapEnvelope("ZoneGroupTopology", "GetZoneGroupState", "<ZoneGroupState>"+zgs+"</ZoneGroupState>"))
	}))
	t.Cleanup(srv.Close)
	return srv, &fetches
}

// runSearch feeds the same SSDP answer to readSearchResponses `repeats`
// times and returns how many players were reported.
func runSearch(t *testing.T, s *Sonos, location string, repeats int) int {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	var found atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.readSearchResponses(context.Background(), conn, true, func(*ZonePlayer) { found.Add(1) })
	}()

	sender, err := net.DialUDP("udp4", nil, conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()
	answer := "HTTP/1.1 200 OK\r\nCACHE-CONTROL: max-age = 1800\r\nLOCATION: " + location +
		"\r\nST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n"
	for range repeats {
		if _, err := sender.Write([]byte(answer)); err != nil {
			t.Fatal(err)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for found.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond) // let any duplicates through
	conn.Close()
	<-done
	return int(found.Load())
}

// Repeated searches on one Sonos must each report the player again, and
// duplicate answers within one search must be fetched only once.
func TestSearchIsRepeatableAndDeduplicates(t *testing.T) {
	dev, fetches := newFakeDevice(t)
	s := newTestSonos(t)
	location := dev.URL + "/xml/device_description.xml"

	for i := range 2 {
		if n := runSearch(t, s, location, 3); n != 1 {
			t.Fatalf("search %d reported %d players, want 1", i+1, n)
		}
	}
	if n := fetches.Load(); n != 2 {
		t.Errorf("device description fetched %d times, want 2 (once per search)", n)
	}
}

func TestZoneGroupStateAttributes(t *testing.T) {
	const raw = `<ZoneGroupState><ZoneGroups>
<ZoneGroup Coordinator="RINCON_A" ID="RINCON_A:1">
  <ZoneGroupMember UUID="RINCON_A" Location="http://10.0.0.2:1400/xml/device_description.xml" ZoneName="Kitchen"
    Icon="x-rincon-roomicon:kitchen" Configuration="1" SoftwareVersion="80.1" SWGen="2" BootSeq="42" ChannelMapSet="RINCON_A:LF,RF">
    <Satellite UUID="RINCON_S" Location="http://10.0.0.3:1400/xml/device_description.xml" ZoneName="Kitchen" SWGen="2" Invisible="1"/>
  </ZoneGroupMember>
</ZoneGroup></ZoneGroups>
<VanishedDevices><VanishedDevice UUID="RINCON_V" ZoneName="Garage" Reason="powered off"/></VanishedDevices>
</ZoneGroupState>`

	var st ZoneGroupState
	if err := xml.Unmarshal([]byte(raw), &st); err != nil {
		t.Fatal(err)
	}
	m := st.ZoneGroups[0].ZoneGroupMember[0]
	if m.Icon != "x-rincon-roomicon:kitchen" || m.Configuration != "1" || m.SWGen != "2" || m.BootSeq != "42" {
		t.Errorf("member attributes not parsed: %+v", m)
	}
	if m.ChannelMapSet != "RINCON_A:LF,RF" {
		t.Errorf("ChannelMapSet = %q", m.ChannelMapSet)
	}
	if len(m.Satellite) != 1 || m.Satellite[0].SWGen != "2" {
		t.Errorf("satellite not parsed: %+v", m.Satellite)
	}
	if len(st.VanishedDevices) != 1 || st.VanishedDevices[0].ZoneName != "Garage" {
		t.Errorf("vanished devices not parsed: %+v", st.VanishedDevices)
	}
}

var startingIndexRe = regexp.MustCompile(`<StartingIndex>(\d+)</StartingIndex>`)

func TestListQueuePages(t *testing.T) {
	const total = 250
	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:Queue:1#Browse": func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			start, _ := strconv.Atoi(startingIndexRe.FindStringSubmatch(string(body))[1])
			end := min(start+queuePageSize, total)

			var didl strings.Builder
			didl.WriteString(`<DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">`)
			for i := start; i < end; i++ {
				fmt.Fprintf(&didl, `<item id="Q:0/%d"></item>`, i+1)
			}
			didl.WriteString(`</DIDL-Lite>`)
			var escaped bytes.Buffer
			xml.EscapeText(&escaped, []byte(didl.String()))

			res := fmt.Sprintf("<Result>%s</Result><NumberReturned>%d</NumberReturned><TotalMatches>%d</TotalMatches>",
				escaped.String(), end-start, total)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(soapEnvelope("Queue", "Browse", res))),
				Header:     make(http.Header),
			}, nil
		},
	}
	zp := NewMockZonePlayer(t, handlers)

	items, err := groupOf(zp).Queue(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != total {
		t.Fatalf("got %d items, want %d", len(items), total)
	}
	if items[total-1].ID != "Q:0/250" {
		t.Errorf("last item ID = %q", items[total-1].ID)
	}
}

func TestSetVolumeRejectsOutOfRange(t *testing.T) {
	zp := NewMockZonePlayer(t, nil) // any request would fail with 404
	for _, v := range []int{-1, 101, 70000} {
		if err := roomOf(zp).SetVolume(t.Context(), v); err == nil || !strings.Contains(err.Error(), "out of range") {
			t.Errorf("SetVolume(%d) = %v, want out of range error", v, err)
		}
		if err := groupOf(zp).SetVolume(t.Context(), v); err == nil || !strings.Contains(err.Error(), "out of range") {
			t.Errorf("SetGroupVolume(%d) = %v, want out of range error", v, err)
		}
	}
}

func TestUUIDWithoutPrefix(t *testing.T) {
	zp := NewMockZonePlayer(t, nil)
	if got := zp.UUID(); got != "RINCON_000E58CDCA4001400" {
		t.Errorf("UUID() = %q", got)
	}
	zp.Description.Device.UDN = "RINCON_NOPREFIX"
	if got := zp.UUID(); got != "RINCON_NOPREFIX" {
		t.Errorf("UUID() without prefix = %q", got)
	}
}
