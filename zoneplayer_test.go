package sonos

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	ain "github.com/caglar10ur/sonos/services/audioin"
)

// Helper to wrap body in SOAP envelope
func soapEnvelope(service, action, bodyContent string) string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:` + action + `Response xmlns:u="urn:schemas-upnp-org:service:` + service + `:1">` +
		bodyContent +
		`</u:` + action + `Response>
  </s:Body>
</s:Envelope>`
}

// Helper to create a simple success handler
func mockSuccessHandler(service, action string) func(*http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(soapEnvelope(service, action, ""))),
			Header:     make(http.Header),
		}, nil
	}
}

// Helper to create a handler with body content
func mockResponseHandler(service, action, bodyContent string) func(*http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(soapEnvelope(service, action, bodyContent))),
			Header:     make(http.Header),
		}, nil
	}
}

func TestGetVolume(t *testing.T) {
	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:RenderingControl:1#GetVolume": func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			if !bytes.Contains(body, []byte("<Channel>Master</Channel>")) {
				t.Errorf("Expected Channel Master in request, got: %s", string(body))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(soapEnvelope("RenderingControl", "GetVolume", "<CurrentVolume>42</CurrentVolume>"))),
				Header:     make(http.Header),
			}, nil
		},
	}

	zp := NewMockZonePlayer(t, handlers)

	vol, err := roomOf(zp).Volume(t.Context())
	if err != nil {
		t.Fatalf("GetVolume failed: %v", err)
	}

	if vol != 42 {
		t.Errorf("Expected volume 42, got %d", vol)
	}
}

func TestSetVolume(t *testing.T) {
	var receivedVolume string

	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:RenderingControl:1#SetVolume": func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			if bytes.Contains(body, []byte("<DesiredVolume>25</DesiredVolume>")) {
				receivedVolume = "25"
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(soapEnvelope("RenderingControl", "SetVolume", ""))),
				Header:     make(http.Header),
			}, nil
		},
	}

	zp := NewMockZonePlayer(t, handlers)

	err := roomOf(zp).SetVolume(t.Context(), 25)
	if err != nil {
		t.Fatalf("SetVolume failed: %v", err)
	}

	if receivedVolume != "25" {
		t.Errorf("SetVolume did not send the correct volume value in the payload")
	}
}

func TestPlaybackControls(t *testing.T) {
	tests := []struct {
		name   string
		action string
		call   func(*ZonePlayer) error
	}{
		{"Play", "Play", func(zp *ZonePlayer) error { return groupOf(zp).Play(t.Context()) }},
		{"Stop", "Stop", func(zp *ZonePlayer) error { return groupOf(zp).Stop(t.Context()) }},
		{"Pause", "Pause", func(zp *ZonePlayer) error { return groupOf(zp).Pause(t.Context()) }},
		{"Next", "Next", func(zp *ZonePlayer) error { return groupOf(zp).Next(t.Context()) }},
		{"Previous", "Previous", func(zp *ZonePlayer) error { return groupOf(zp).Previous(t.Context()) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(*http.Request) (*http.Response, error){
				"urn:schemas-upnp-org:service:AVTransport:1#" + tt.action: mockSuccessHandler("AVTransport", tt.action),
			}
			zp := NewMockZonePlayer(t, handlers)
			if err := tt.call(zp); err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
}

func TestListQueue(t *testing.T) {
	didlResult := `&lt;DIDL-Lite xmlns:dc=&quot;http://purl.org/dc/elements/1.1/&quot; xmlns:upnp=&quot;urn:schemas-upnp-org:metadata-1-0/upnp/&quot; xmlns:r=&quot;urn:schemas-rinconnetworks-com:metadata-1-0/&quot; xmlns=&quot;urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/&quot;&gt;&lt;item id=&quot;Q:0/1&quot; parentID=&quot;Q:0&quot; restricted=&quot;true&quot;&gt;&lt;dc:title&gt;Song Title&lt;/dc:title&gt;&lt;dc:creator&gt;Artist Name&lt;/dc:creator&gt;&lt;upnp:album&gt;Album Name&lt;/upnp:album&gt;&lt;upnp:albumArtURI&gt;http://192.168.1.100:1400/getaa?s=1&amp;amp;u=x-sonos-http%3atrack.mp3&lt;/upnp:albumArtURI&gt;&lt;res protocolInfo=&quot;http-get:*:audio/mpeg:*&quot; duration=&quot;0:03:30&quot;&gt;x-sonos-http:track.mp3&lt;/res&gt;&lt;upnp:class&gt;object.item.audioItem.musicTrack&lt;/upnp:class&gt;&lt;/item&gt;&lt;/DIDL-Lite&gt;`
	body := `<Result>` + didlResult + `</Result>
      <NumberReturned>1</NumberReturned>
      <TotalMatches>1</TotalMatches>
      <UpdateID>100</UpdateID>`

	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:Queue:1#Browse": mockResponseHandler("Queue", "Browse", body),
	}

	zp := NewMockZonePlayer(t, handlers)

	items, err := groupOf(zp).Queue(t.Context())
	if err != nil {
		t.Fatalf("ListQueue failed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item in queue, got %d", len(items))
	}

	if items[0].Title[0].Value != "Song Title" {
		t.Errorf("Expected title 'Song Title', got '%s'", items[0].Title[0].Value)
	}
}

func TestNowPlaying(t *testing.T) {
	meta := `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">` +
		`<item id="-1" parentID="-1"><dc:title>Karma Police</dc:title><dc:creator>Radiohead</dc:creator>` +
		`<upnp:album>OK Computer</upnp:album><upnp:albumArtURI>/getaa?s=1&amp;u=x</upnp:albumArtURI></item></DIDL-Lite>`
	var escaped strings.Builder
	xml.EscapeText(&escaped, []byte(meta))

	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:AVTransport:1#GetTransportInfo": mockResponseHandler("AVTransport", "GetTransportInfo",
			`<CurrentTransportState>PLAYING</CurrentTransportState><CurrentTransportStatus>OK</CurrentTransportStatus><CurrentSpeed>1</CurrentSpeed>`),
		"urn:schemas-upnp-org:service:AVTransport:1#GetPositionInfo": mockResponseHandler("AVTransport", "GetPositionInfo",
			`<Track>2</Track><TrackDuration>0:04:24</TrackDuration><TrackMetaData>`+escaped.String()+`</TrackMetaData>`+
				`<TrackURI>x-sonos-spotify:track</TrackURI><RelTime>0:01:23.500</RelTime><AbsTime>NOT_IMPLEMENTED</AbsTime>`),
		"urn:schemas-upnp-org:service:AVTransport:1#GetMediaInfo": mockResponseHandler("AVTransport", "GetMediaInfo",
			`<NrTracks>12</NrTracks><CurrentURI>x-rincon-queue:RINCON_000E58CDCA4001400#0</CurrentURI>`),
	}
	np, err := groupOf(NewMockZonePlayer(t, handlers)).NowPlaying(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if np.State != Playing || np.QueuePosition != 2 || np.Position != 83500*time.Millisecond {
		t.Errorf("got state %q, queue position %d, position %v", np.State, np.QueuePosition, np.Position)
	}
	want := Track{
		Title: "Karma Police", Artist: "Radiohead", Album: "OK Computer",
		AlbumArtURI: "http://192.168.1.100:1400/getaa?s=1&u=x",
		URI:         "x-sonos-spotify:track", Duration: 4*time.Minute + 24*time.Second,
	}
	if np.Track == nil || *np.Track != want {
		t.Errorf("track = %+v, want %+v", np.Track, want)
	}
}

func TestParseDuration(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"0:04:58":         4*time.Minute + 58*time.Second,
		"1:02:03.250":     time.Hour + 2*time.Minute + 3250*time.Millisecond,
		"NOT_IMPLEMENTED": 0,
		"":                0,
		"4:58":            0,
	} {
		if got := parseDuration(in); got != want {
			t.Errorf("parseDuration(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestMuteUnmute(t *testing.T) {
	muteState := false

	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:RenderingControl:1#SetMute": func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			if bytes.Contains(body, []byte("<DesiredMute>true</DesiredMute>")) || bytes.Contains(body, []byte("<DesiredMute>1</DesiredMute>")) {
				muteState = true
			} else if bytes.Contains(body, []byte("<DesiredMute>false</DesiredMute>")) || bytes.Contains(body, []byte("<DesiredMute>0</DesiredMute>")) {
				muteState = false
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(soapEnvelope("RenderingControl", "SetMute", ""))),
				Header:     make(http.Header),
			}, nil
		},
		"urn:schemas-upnp-org:service:RenderingControl:1#GetMute": func(req *http.Request) (*http.Response, error) {
			val := "0"
			if muteState {
				val = "1"
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(soapEnvelope("RenderingControl", "GetMute", "<CurrentMute>"+val+"</CurrentMute>"))),
				Header:     make(http.Header),
			}, nil
		},
	}

	zp := NewMockZonePlayer(t, handlers)

	if err := roomOf(zp).Mute(t.Context()); err != nil {
		t.Fatalf("Mute failed: %v", err)
	}
	muted, _ := roomOf(zp).IsMuted(t.Context())
	if !muted {
		t.Error("Expected state to be muted after Mute()")
	}

	if err := roomOf(zp).Unmute(t.Context()); err != nil {
		t.Fatalf("Unmute failed: %v", err)
	}
	muted, _ = roomOf(zp).IsMuted(t.Context())
	if muted {
		t.Error("Expected state to be unmuted after Unmute()")
	}
}

func TestGetZoneInfo(t *testing.T) {
	body := `<SerialNumber>00-0E-58-CD-CA-40:0</SerialNumber>
      <SoftwareVersion>57.2-54070</SoftwareVersion>
      <DisplaySoftwareVersion>10.3</DisplaySoftwareVersion>
      <IPAddress>192.168.1.100</IPAddress>
      <MACAddress>00:0E:58:CD:CA:40</MACAddress>`

	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:DeviceProperties:1#GetZoneInfo": mockResponseHandler("DeviceProperties", "GetZoneInfo", body),
	}

	zp := NewMockZonePlayer(t, handlers)

	info, err := zp.DeviceProperties.GetZoneInfo(t.Context(), nil)
	if err != nil {
		t.Fatalf("GetZoneInfo failed: %v", err)
	}

	if info.SerialNumber != "00-0E-58-CD-CA-40:0" {
		t.Errorf("Unexpected SerialNumber: %s", info.SerialNumber)
	}
}

func TestZonePlayerGetters(t *testing.T) {
	zp := NewMockZonePlayer(t, nil)

	if zp.Location() == nil {
		t.Error("Location() returned nil")
	}
	if zp.RoomName() != "Kitchen" {
		t.Errorf("RoomName() = %s, expected Kitchen", zp.RoomName())
	}
	if zp.ModelName() != "Sonos One" {
		t.Errorf("ModelName() = %s, expected Sonos One", zp.ModelName())
	}
	if zp.SerialNumber() != "00-11-22-33-44-55:0" {
		t.Errorf("SerialNumber() = %s, expected 00-11-22-33-44-55:0", zp.SerialNumber())
	}
	if zp.HardwareVersion() != "1.20.1.6-1.2" {
		t.Errorf("HardwareVersion() = %s, expected 1.20.1.6-1.2", zp.HardwareVersion())
	}
	if zp.SoftwareVersion() != "57.2-54070" {
		t.Errorf("SoftwareVersion() = %s, expected 57.2-54070", zp.SoftwareVersion())
	}
	if zp.ModelDescription() != "Sonos One Audio Player" {
		t.Errorf("ModelDescription() = %s, expected Sonos One Audio Player", zp.ModelDescription())
	}
	if zp.UUID() != "RINCON_000E58CDCA4001400" {
		t.Errorf("UUID() = %s, expected RINCON_000E58CDCA4001400", zp.UUID())
	}
}

func TestIsCoordinator(t *testing.T) {
	innerXML := `&lt;ZoneGroupState&gt;&lt;ZoneGroups&gt;&lt;ZoneGroup Coordinator=&quot;RINCON_000E58CDCA4001400&quot; ID=&quot;RINCON_000E58CDCA4001400:1&quot;&gt;&lt;ZoneGroupMember UUID=&quot;RINCON_000E58CDCA4001400&quot; Location=&quot;http://192.168.1.100:1400/xml/device_description.xml&quot; ZoneName=&quot;Kitchen&quot; /&gt;&lt;/ZoneGroup&gt;&lt;/ZoneGroups&gt;&lt;/ZoneGroupState&gt;`
	handlersCoordinator := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:ZoneGroupTopology:1#GetZoneGroupState": mockResponseHandler("ZoneGroupTopology", "GetZoneGroupState", "<ZoneGroupState>"+innerXML+"</ZoneGroupState>"),
	}

	zp := NewMockZonePlayer(t, handlersCoordinator)
	if ok, err := zp.IsCoordinator(t.Context()); err != nil || !ok {
		t.Errorf("IsCoordinator() = %v, %v; want true", ok, err)
	}

	innerXMLMember := `&lt;ZoneGroupState&gt;&lt;ZoneGroups&gt;&lt;ZoneGroup Coordinator=&quot;RINCON_OTHER&quot; ID=&quot;RINCON_OTHER:1&quot;&gt;&lt;ZoneGroupMember UUID=&quot;RINCON_000E58CDCA4001400&quot; Location=&quot;http://192.168.1.100:1400/xml/device_description.xml&quot; ZoneName=&quot;Kitchen&quot; /&gt;&lt;/ZoneGroup&gt;&lt;/ZoneGroups&gt;&lt;/ZoneGroupState&gt;`
	handlersMember := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:ZoneGroupTopology:1#GetZoneGroupState": mockResponseHandler("ZoneGroupTopology", "GetZoneGroupState", "<ZoneGroupState>"+innerXMLMember+"</ZoneGroupState>"),
	}

	zpMember := NewMockZonePlayer(t, handlersMember)
	if ok, err := zpMember.IsCoordinator(t.Context()); err != nil || ok {
		t.Errorf("IsCoordinator() = %v, %v; want false", ok, err)
	}
}

func TestGroupControls(t *testing.T) {
	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:GroupRenderingControl:1#GetGroupVolume":      mockResponseHandler("GroupRenderingControl", "GetGroupVolume", "<CurrentVolume>50</CurrentVolume>"),
		"urn:schemas-upnp-org:service:GroupRenderingControl:1#SetGroupVolume":      mockSuccessHandler("GroupRenderingControl", "SetGroupVolume"),
		"urn:schemas-upnp-org:service:GroupRenderingControl:1#SnapshotGroupVolume": mockSuccessHandler("GroupRenderingControl", "SnapshotGroupVolume"),
	}

	zp := NewMockZonePlayer(t, handlers)

	vol, err := groupOf(zp).Volume(t.Context())
	if err != nil {
		t.Fatalf("GetGroupVolume failed: %v", err)
	}
	if vol != 50 {
		t.Errorf("Expected group volume 50, got %d", vol)
	}

	if err := groupOf(zp).SetVolume(t.Context(), 60); err != nil {
		t.Fatalf("SetGroupVolume failed: %v", err)
	}
}

func TestZoneGroupState(t *testing.T) {
	innerXML := `&lt;ZoneGroupState&gt;&lt;ZoneGroups&gt;&lt;ZoneGroup Coordinator=&quot;RINCON_AAA&quot; ID=&quot;RINCON_AAA:1&quot;&gt;&lt;ZoneGroupMember UUID=&quot;RINCON_AAA&quot; Location=&quot;http://192.168.1.100:1400/xml/device_description.xml&quot; ZoneName=&quot;Kitchen&quot; /&gt;&lt;/ZoneGroup&gt;&lt;/ZoneGroups&gt;&lt;/ZoneGroupState&gt;`
	attrBody := `<CurrentZoneGroupName>Kitchen</CurrentZoneGroupName>
      <CurrentZoneGroupID>RINCON_AAA:1</CurrentZoneGroupID>
      <CurrentZonePlayerUUIDsInGroup>RINCON_AAA</CurrentZonePlayerUUIDsInGroup>`

	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:ZoneGroupTopology:1#GetZoneGroupState":      mockResponseHandler("ZoneGroupTopology", "GetZoneGroupState", "<ZoneGroupState>"+innerXML+"</ZoneGroupState>"),
		"urn:schemas-upnp-org:service:ZoneGroupTopology:1#GetZoneGroupAttributes": mockResponseHandler("ZoneGroupTopology", "GetZoneGroupAttributes", attrBody),
	}

	zp := NewMockZonePlayer(t, handlers)

	state, err := zp.ZoneGroupState(t.Context())
	if err != nil {
		t.Fatalf("GetZoneGroupState failed: %v", err)
	}
	if len(state.ZoneGroups) != 1 {
		t.Errorf("Expected 1 ZoneGroup, got %d", len(state.ZoneGroups))
	}

	attrs, err := zp.ZoneGroupTopology.GetZoneGroupAttributes(t.Context(), nil)
	if err != nil {
		t.Fatalf("GetZoneGroupAttributes failed: %v", err)
	}
	if attrs.CurrentZoneGroupName != "Kitchen" {
		t.Errorf("Expected GroupName Kitchen, got %s", attrs.CurrentZoneGroupName)
	}
}

func TestInputOutput(t *testing.T) {
	handlers := map[string]func(*http.Request) (*http.Response, error){
		"urn:schemas-upnp-org:service:AudioIn:1#GetAudioInputAttributes": mockResponseHandler("AudioIn", "GetAudioInputAttributes", "<CurrentName>Turntable</CurrentName><CurrentIcon>icon.png</CurrentIcon>"),
		"urn:schemas-upnp-org:service:AudioIn:1#SetAudioInputAttributes": mockSuccessHandler("AudioIn", "SetAudioInputAttributes"),
		"urn:schemas-upnp-org:service:AudioIn:1#GetLineInLevel":          mockResponseHandler("AudioIn", "GetLineInLevel", "<CurrentLeftLineInLevel>10</CurrentLeftLineInLevel><CurrentRightLineInLevel>10</CurrentRightLineInLevel>"),
		"urn:schemas-upnp-org:service:AudioIn:1#SetLineInLevel":          mockSuccessHandler("AudioIn", "SetLineInLevel"),
		"urn:schemas-upnp-org:service:AudioIn:1#SelectAudio":             mockSuccessHandler("AudioIn", "SelectAudio"),
		"urn:schemas-upnp-org:service:AVTransport:1#SetAVTransportURI":   mockSuccessHandler("AVTransport", "SetAVTransportURI"),
	}

	zp := NewMockZonePlayer(t, handlers)

	attrs, err := zp.AudioIn.GetAudioInputAttributes(t.Context(), nil)
	if err != nil {
		t.Fatalf("GetAudioInputAttributes failed: %v", err)
	}
	if attrs.CurrentName != "Turntable" {
		t.Errorf("Expected CurrentName Turntable, got %s", attrs.CurrentName)
	}

	if _, err := zp.AudioIn.SetAudioInputAttributes(t.Context(), &ain.SetAudioInputAttributesArgs{DesiredName: "Radio", DesiredIcon: "radio.png"}); err != nil {
		t.Fatalf("SetAudioInputAttributes failed: %v", err)
	}

	levels, err := zp.AudioIn.GetLineInLevel(t.Context(), nil)
	if err != nil {
		t.Fatalf("GetLineInLevel failed: %v", err)
	}
	if levels.CurrentLeftLineInLevel != 10 {
		t.Errorf("Expected LeftLevel 10, got %d", levels.CurrentLeftLineInLevel)
	}

	if _, err := zp.AudioIn.SetLineInLevel(t.Context(), &ain.SetLineInLevelArgs{DesiredLeftLineInLevel: 5, DesiredRightLineInLevel: 5}); err != nil {
		t.Fatalf("SetLineInLevel failed: %v", err)
	}

	if _, err := zp.AudioIn.SelectAudio(t.Context(), &ain.SelectAudioArgs{ObjectID: "RINCON_AAA"}); err != nil {
		t.Fatalf("SelectAudio failed: %v", err)
	}

	g := groupOf(zp)
	if err := g.PlayLineIn(t.Context(), g.Coordinator); err != nil {
		t.Fatalf("PlayLineIn failed: %v", err)
	}

	if err := g.PlayQueue(t.Context()); err != nil {
		t.Fatalf("PlayQueue failed: %v", err)
	}

	if err := g.SetURI(t.Context(), "x-rincon-stream:uuid", ""); err != nil {
		t.Fatalf("SetURI failed: %v", err)
	}
}

func TestParsePosition(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"1:30":    90 * time.Second,
		"1:02:03": time.Hour + 2*time.Minute + 3*time.Second,
		"90s":     90 * time.Second,
		"0:00":    0,
	} {
		if got, err := ParsePosition(in); err != nil || got != want {
			t.Errorf("ParsePosition(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "-30s", "1:x", "1:2:3:4", "-1:00"} {
		if _, err := ParsePosition(in); err == nil {
			t.Errorf("ParsePosition(%q) should fail", in)
		}
	}
}
