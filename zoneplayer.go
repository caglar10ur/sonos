package sonos

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	clk "github.com/caglar10ur/sonos/services/alarmclock"
	ain "github.com/caglar10ur/sonos/services/audioin"
	avt "github.com/caglar10ur/sonos/services/avtransport"
	con "github.com/caglar10ur/sonos/services/connectionmanager"
	dir "github.com/caglar10ur/sonos/services/contentdirectory"
	dev "github.com/caglar10ur/sonos/services/deviceproperties"
	gmn "github.com/caglar10ur/sonos/services/groupmanagement"
	rcg "github.com/caglar10ur/sonos/services/grouprenderingcontrol"
	mus "github.com/caglar10ur/sonos/services/musicservices"
	ply "github.com/caglar10ur/sonos/services/qplay"
	que "github.com/caglar10ur/sonos/services/queue"
	ren "github.com/caglar10ur/sonos/services/renderingcontrol"
	sys "github.com/caglar10ur/sonos/services/systemproperties"
	vli "github.com/caglar10ur/sonos/services/virtuallinein"
	zgt "github.com/caglar10ur/sonos/services/zonegrouptopology"
)

// SpecVersion is the UPnP version a device description follows.
type SpecVersion struct {
	XMLName xml.Name `xml:"specVersion"`
	Major   int      `xml:"major"`
	Minor   int      `xml:"minor"`
}

// DeviceService is a UPnP service listed in a device description.
type DeviceService struct {
	XMLName     xml.Name `xml:"service"`
	ServiceType string   `xml:"serviceType"`
	ServiceID   string   `xml:"serviceId"`
	ControlURL  string   `xml:"controlURL"`
	EventSubURL string   `xml:"eventSubURL"`
	SCPDURL     string   `xml:"SCPDURL"`
}

// DeviceIcon is an icon listed in a device description. URL is relative to
// the description's location.
type DeviceIcon struct {
	XMLName  xml.Name `xml:"icon"`
	ID       string   `xml:"id"`
	Mimetype string   `xml:"mimetype"`
	Width    int      `xml:"width"`
	Height   int      `xml:"height"`
	Depth    int      `xml:"depth"`
	URL      string   `xml:"url"`
}

// Device is the <device> element of a device description. A player
// describes itself and, in Devices, its media renderer and media server.
type Device struct {
	XMLName                 xml.Name        `xml:"device"`
	DeviceType              string          `xml:"deviceType"`
	FriendlyName            string          `xml:"friendlyName"`
	Manufacturer            string          `xml:"manufacturer"`
	ManufacturerURL         string          `xml:"manufacturerURL"`
	ModelNumber             string          `xml:"modelNumber"`
	ModelDescription        string          `xml:"modelDescription"`
	ModelName               string          `xml:"modelName"`
	ModelURL                string          `xml:"modelURL"`
	SoftwareVersion         string          `xml:"softwareVersion"`
	SwGen                   string          `xml:"swGen"`
	HardwareVersion         string          `xml:"hardwareVersion"`
	SerialNum               string          `xml:"serialNum"`
	MACAddress              string          `xml:"MACAddress"`
	UDN                     string          `xml:"UDN"`
	Icons                   []DeviceIcon    `xml:"iconList>icon"`
	MinCompatibleVersion    string          `xml:"minCompatibleVersion"`
	LegacyCompatibleVersion string          `xml:"legacyCompatibleVersion"`
	APIVersion              string          `xml:"apiVersion"`
	MinAPIVersion           string          `xml:"minApiVersion"`
	DisplayVersion          string          `xml:"displayVersion"`
	ExtraVersion            string          `xml:"extraVersion"`
	RoomName                string          `xml:"roomName"`
	DisplayName             string          `xml:"displayName"`
	ZoneType                int             `xml:"zoneType"`
	Feature1                string          `xml:"feature1"`
	Feature2                string          `xml:"feature2"`
	Feature3                string          `xml:"feature3"`
	Seriesid                string          `xml:"seriesid"`
	Variant                 int             `xml:"variant"`
	InternalSpeakerSize     float32         `xml:"internalSpeakerSize"`
	BassExtension           float32         `xml:"bassExtension"`
	SatGainOffset           float32         `xml:"satGainOffset"`
	Memory                  int             `xml:"memory"`
	Flash                   int             `xml:"flash"`
	FlashRepartitioned      int             `xml:"flashRepartitioned"`
	AmpOnTime               int             `xml:"ampOnTime"`
	RetailMode              int             `xml:"retailMode"`
	Services                []DeviceService `xml:"serviceList>service"`
	Devices                 []Device        `xml:"deviceList>device"`
}

// DeviceDescription is a player's UPnP device description, the document at
// its location (see FromEndpoint).
type DeviceDescription struct {
	XMLName     xml.Name    `xml:"root"`
	Xmlns       string      `xml:"xmlns,attr"`
	SpecVersion SpecVersion `xml:"specVersion"`
	Device      Device      `xml:"device"`
}

// ZonePlayerOption configures a ZonePlayer.
type ZonePlayerOption func(*ZonePlayer)

// WithClient sets the HTTP client used to talk to the player.
// The default has a 10 second timeout.
func WithClient(c *http.Client) ZonePlayerOption {
	return func(z *ZonePlayer) {
		z.client = c
	}
}

// FromEndpoint returns the device description URL of the player at host
// (an IP address or name, without a port).
func FromEndpoint(host string) (*url.URL, error) {
	return url.Parse("http://" + net.JoinHostPort(host, "1400") + "/xml/device_description.xml")
}

// coordinatorCacheTTL controls how long the IsCoordinator result is reused
// before a fresh GetZoneGroupState call is made.
const coordinatorCacheTTL = 30 * time.Second

// ZonePlayer is one Sonos player. The embedded Services give access to
// every UPnP action; its methods cover what belongs to the player itself.
// For rooms and groups, and playback, see Household.
type ZonePlayer struct {
	Description *DeviceDescription

	client *http.Client
	// A URL that can be queried for device capabilities
	location *url.URL

	coordinatorMu    sync.Mutex
	coordinatorVal   bool
	coordinatorExpAt time.Time

	*Services
}

// Services are the UPnP services of a player.
type Services struct {
	AlarmClock  *clk.Service
	AudioIn     *ain.Service
	AVTransport *avt.Service
	// ConnectionManager is the MediaRenderer's connection manager.
	ConnectionManager *con.Service
	// MediaServerConnectionManager is the MediaServer's connection manager.
	MediaServerConnectionManager *con.Service
	ContentDirectory             *dir.Service
	DeviceProperties             *dev.Service
	GroupManagement              *gmn.Service
	GroupRenderingControl        *rcg.Service
	MusicServices                *mus.Service
	QPlay                        *ply.Service
	Queue                        *que.Service
	RenderingControl             *ren.Service
	SystemProperties             *sys.Service
	VirtualLineIn                *vli.Service
	ZoneGroupTopology            *zgt.Service
}

// NewZonePlayer fetches the device description at location (see
// FromEndpoint) and returns the player it describes.
func NewZonePlayer(ctx context.Context, location *url.URL, opts ...ZonePlayerOption) (*ZonePlayer, error) {
	if location == nil {
		return nil, errors.New("sonos: nil location")
	}
	zp := &ZonePlayer{
		Description: &DeviceDescription{},
		client:      &http.Client{Timeout: 10 * time.Second},
		location:    location,
	}
	for _, opt := range opts {
		opt(zp)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := zp.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sonos: fetching %s: %s", location, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(body, zp.Description); err != nil {
		return nil, err
	}

	zp.Services, err = newServices(location, zp.client)
	if err != nil {
		return nil, err
	}
	return zp, nil
}

func newServices(loc *url.URL, c *http.Client) (*Services, error) {
	var (
		s   Services
		err error
	)
	if s.AlarmClock, err = clk.NewService(loc, clk.WithClient(c)); err != nil {
		return nil, err
	}
	if s.AudioIn, err = ain.NewService(loc, ain.WithClient(c)); err != nil {
		return nil, err
	}
	if s.AVTransport, err = avt.NewService(loc, avt.WithClient(c)); err != nil {
		return nil, err
	}
	if s.ConnectionManager, err = con.NewService(loc, con.WithClient(c)); err != nil {
		return nil, err
	}
	if s.MediaServerConnectionManager, err = con.NewService(loc, con.WithClient(c),
		con.WithPaths("/MediaServer/ConnectionManager/Control", "/MediaServer/ConnectionManager/Event")); err != nil {
		return nil, err
	}
	if s.ContentDirectory, err = dir.NewService(loc, dir.WithClient(c)); err != nil {
		return nil, err
	}
	if s.DeviceProperties, err = dev.NewService(loc, dev.WithClient(c)); err != nil {
		return nil, err
	}
	if s.GroupManagement, err = gmn.NewService(loc, gmn.WithClient(c)); err != nil {
		return nil, err
	}
	if s.GroupRenderingControl, err = rcg.NewService(loc, rcg.WithClient(c)); err != nil {
		return nil, err
	}
	if s.MusicServices, err = mus.NewService(loc, mus.WithClient(c)); err != nil {
		return nil, err
	}
	if s.QPlay, err = ply.NewService(loc, ply.WithClient(c)); err != nil {
		return nil, err
	}
	if s.Queue, err = que.NewService(loc, que.WithClient(c)); err != nil {
		return nil, err
	}
	if s.RenderingControl, err = ren.NewService(loc, ren.WithClient(c)); err != nil {
		return nil, err
	}
	if s.SystemProperties, err = sys.NewService(loc, sys.WithClient(c)); err != nil {
		return nil, err
	}
	if s.VirtualLineIn, err = vli.NewService(loc, vli.WithClient(c)); err != nil {
		return nil, err
	}
	if s.ZoneGroupTopology, err = zgt.NewService(loc, zgt.WithClient(c)); err != nil {
		return nil, err
	}
	return &s, nil
}

// Client returns the HTTP client used to talk to the player.
func (z *ZonePlayer) Client() *http.Client {
	return z.client
}

// Location returns the URL of the player's device description.
func (z *ZonePlayer) Location() *url.URL {
	return z.location
}

// RoomName returns the name of the room the player is in.
func (z *ZonePlayer) RoomName() string {
	return z.Description.Device.RoomName
}

// ModelName returns the player's model, e.g. "Sonos Era 300".
func (z *ZonePlayer) ModelName() string {
	return z.Description.Device.ModelName
}

// HardwareVersion returns the player's hardware version.
func (z *ZonePlayer) HardwareVersion() string {
	return z.Description.Device.HardwareVersion
}

// SoftwareVersion returns the player's software version.
func (z *ZonePlayer) SoftwareVersion() string {
	return z.Description.Device.SoftwareVersion
}

// SerialNumber returns the player's serial number.
func (z *ZonePlayer) SerialNumber() string {
	return z.Description.Device.SerialNum
}

// MACAddress returns the player's MAC address.
func (z *ZonePlayer) MACAddress() string {
	return z.Description.Device.MACAddress
}

// ModelDescription returns the player's model description.
func (z *ZonePlayer) ModelDescription() string {
	return z.Description.Device.ModelDescription
}

// UUID returns the device UDN without its "uuid:" prefix, e.g.
// "RINCON_000E58CDCA4001400". Zone group topology refers to players by it.
func (z *ZonePlayer) UUID() string {
	if _, uuid, ok := strings.Cut(z.Description.Device.UDN, ":"); ok {
		return uuid
	}
	return z.Description.Device.UDN
}

// IsCoordinator reports whether the player coordinates its group. The
// answer is cached for 30 seconds.
func (z *ZonePlayer) IsCoordinator(ctx context.Context) (bool, error) {
	z.coordinatorMu.Lock()
	if time.Now().Before(z.coordinatorExpAt) {
		v := z.coordinatorVal
		z.coordinatorMu.Unlock()
		return v, nil
	}
	z.coordinatorMu.Unlock()

	state, err := z.ZoneGroupState(ctx)
	if err != nil {
		return false, err
	}
	uuid := z.UUID()
	result := slices.ContainsFunc(state.ZoneGroups, func(g ZoneGroup) bool { return g.Coordinator == uuid })

	z.coordinatorMu.Lock()
	z.coordinatorVal = result
	z.coordinatorExpAt = time.Now().Add(coordinatorCacheTTL)
	z.coordinatorMu.Unlock()

	return result, nil
}

// playerAt returns the player whose description is at location.
func playerAt(ctx context.Context, location string, c *http.Client) (*ZonePlayer, error) {
	u, err := url.Parse(location)
	if err != nil {
		return nil, err
	}
	return NewZonePlayer(ctx, u, WithClient(c))
}

// ZoneGroupState returns the household's zone group topology: every group,
// its coordinator and members, as this player sees it. Every player sees
// the same topology.
func (z *ZonePlayer) ZoneGroupState(ctx context.Context) (*ZoneGroupState, error) {
	res, err := z.ZoneGroupTopology.GetZoneGroupState(ctx, nil)
	if err != nil {
		return nil, err
	}
	var state ZoneGroupState
	if err := xml.Unmarshal([]byte(res.ZoneGroupState), &state); err != nil {
		return nil, err
	}
	return &state, nil
}
