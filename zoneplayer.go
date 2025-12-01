package sonos

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/caglar10ur/sonos/didl"
	avt "github.com/caglar10ur/sonos/services/AVTransport"
	clk "github.com/caglar10ur/sonos/services/AlarmClock"
	ain "github.com/caglar10ur/sonos/services/AudioIn"
	con "github.com/caglar10ur/sonos/services/ConnectionManager"
	dir "github.com/caglar10ur/sonos/services/ContentDirectory"
	dev "github.com/caglar10ur/sonos/services/DeviceProperties"
	gmn "github.com/caglar10ur/sonos/services/GroupManagement"
	rcg "github.com/caglar10ur/sonos/services/GroupRenderingControl"
	mus "github.com/caglar10ur/sonos/services/MusicServices"
	ply "github.com/caglar10ur/sonos/services/QPlay"
	que "github.com/caglar10ur/sonos/services/Queue"
	ren "github.com/caglar10ur/sonos/services/RenderingControl"
	sys "github.com/caglar10ur/sonos/services/SystemProperties"
	vli "github.com/caglar10ur/sonos/services/VirtualLineIn"
	zgt "github.com/caglar10ur/sonos/services/ZoneGroupTopology"
)

type SonosService interface {
	ControlEndpoint() *url.URL
	EventEndpoint() *url.URL
	ParseEvent([]byte) []interface{}
}

type SpecVersion struct {
	XMLName xml.Name `xml:"specVersion"`
	Major   int      `xml:"major"`
	Minor   int      `xml:"minor"`
}

type Service struct {
	XMLName     xml.Name `xml:"service"`
	ServiceType string   `xml:"serviceType"`
	ServiceID   string   `xml:"serviceId"`
	ControlURL  string   `xml:"controlURL"`
	EventSubURL string   `xml:"eventSubURL"`
	SCPDURL     string   `xml:"SCPDURL"`
}

type Icon struct {
	XMLName  xml.Name `xml:"icon"`
	ID       string   `xml:"id"`
	Mimetype string   `xml:"mimetype"`
	Width    int      `xml:"width"`
	Height   int      `xml:"height"`
	Depth    int      `xml:"depth"`
	URL      url.URL  `xml:"url"`
}

type Device struct {
	XMLName                 xml.Name  `xml:"device"`
	DeviceType              string    `xml:"deviceType"`
	FriendlyName            string    `xml:"friendlyName"`
	Manufacturer            string    `xml:"manufacturer"`
	ManufacturerURL         string    `xml:"manufacturerURL"`
	ModelNumber             string    `xml:"modelNumber"`
	ModelDescription        string    `xml:"modelDescription"`
	ModelName               string    `xml:"modelName"`
	ModelURL                string    `xml:"modelURL"`
	SoftwareVersion         string    `xml:"softwareVersion"`
	SwGen                   string    `xml:"swGen"`
	HardwareVersion         string    `xml:"hardwareVersion"`
	SerialNum               string    `xml:"serialNum"`
	MACAddress              string    `xml:"MACAddress"`
	UDN                     string    `xml:"UDN"`
	Icons                   []Icon    `xml:"iconList>icon"`
	MinCompatibleVersion    string    `xml:"minCompatibleVersion"`
	LegacyCompatibleVersion string    `xml:"legacyCompatibleVersion"`
	APIVersion              string    `xml:"apiVersion"`
	MinAPIVersion           string    `xml:"minApiVersion"`
	DisplayVersion          string    `xml:"displayVersion"`
	ExtraVersion            string    `xml:"extraVersion"`
	RoomName                string    `xml:"roomName"`
	DisplayName             string    `xml:"displayName"`
	ZoneType                int       `xml:"zoneType"`
	Feature1                string    `xml:"feature1"`
	Feature2                string    `xml:"feature2"`
	Feature3                string    `xml:"feature3"`
	Seriesid                string    `xml:"seriesid"`
	Variant                 int       `xml:"variant"`
	InternalSpeakerSize     float32   `xml:"internalSpeakerSize"`
	BassExtension           float32   `xml:"bassExtension"`
	SatGainOffset           float32   `xml:"satGainOffset"`
	Memory                  int       `xml:"memory"`
	Flash                   int       `xml:"flash"`
	FlashRepartitioned      int       `xml:"flashRepartitioned"`
	AmpOnTime               int       `xml:"ampOnTime"`
	RetailMode              int       `xml:"retailMode"`
	Services                []Service `xml:"serviceList>service"`
	Devices                 []Device  `xml:"deviceList>device"`
}

type Root struct {
	XMLName     xml.Name    `xml:"root"`
	Xmlns       string      `xml:"xmlns,attr"`
	SpecVersion SpecVersion `xml:"specVersion"`
	Device      Device      `xml:"device"`
}

type ZonePlayerOption func(*ZonePlayer)

func WithClient(c *http.Client) ZonePlayerOption {
	return func(z *ZonePlayer) {
		z.client = c
	}
}

func WithLocation(u *url.URL) ZonePlayerOption {
	return func(z *ZonePlayer) {
		z.location = u
	}
}

func FromEndpoint(endpoint string) (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s:1400/xml/device_description.xml", endpoint))
}

func FromLocation(location string) (*url.URL, error) {
	return url.Parse(location)
}

type ZonePlayer struct {
	Root *Root

	client *http.Client
	// A URL that can be queried for device capabilities
	location *url.URL

	*Services
}

type Services struct {
	// services
	AlarmClock            *clk.Service
	AudioIn               *ain.Service
	AVTransport           *avt.Service
	ConnectionManager     *con.Service
	ContentDirectory      *dir.Service
	DeviceProperties      *dev.Service
	GroupManagement       *gmn.Service
	GroupRenderingControl *rcg.Service
	MusicServices         *mus.Service
	QPlay                 *ply.Service
	Queue                 *que.Service
	RenderingControl      *ren.Service
	SystemProperties      *sys.Service
	VirtualLineIn         *vli.Service
	ZoneGroupTopology     *zgt.Service
}

// NewZonePlayer returns a new ZonePlayer instance.
func NewZonePlayer(opts ...ZonePlayerOption) (*ZonePlayer, error) {
	zp := &ZonePlayer{
		Root: &Root{},
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated *ZonePlayer as the argument
		opt(zp)
	}

	if zp.location == nil {
		return nil, fmt.Errorf("empty location")
	}

	resp, err := zp.client.Get(zp.location.String())
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = xml.Unmarshal(body, zp.Root)
	if err != nil {
		return nil, err
	}

	zp.Services = &Services{
		AlarmClock: clk.NewService(
			clk.WithLocation(zp.location),
			clk.WithClient(zp.client),
		),
		AudioIn: ain.NewService(
			ain.WithLocation(zp.location),
			ain.WithClient(zp.client),
		),
		AVTransport: avt.NewService(
			avt.WithLocation(zp.location),
			avt.WithClient(zp.client),
		),
		ConnectionManager: con.NewService(
			con.WithLocation(zp.location),
			con.WithClient(zp.client),
		),
		ContentDirectory: dir.NewService(
			dir.WithLocation(zp.location),
			dir.WithClient(zp.client),
		),
		DeviceProperties: dev.NewService(
			dev.WithLocation(zp.location),
			dev.WithClient(zp.client),
		),
		GroupManagement: gmn.NewService(
			gmn.WithLocation(zp.location),
			gmn.WithClient(zp.client),
		),
		GroupRenderingControl: rcg.NewService(
			rcg.WithLocation(zp.location),
			rcg.WithClient(zp.client),
		),
		MusicServices: mus.NewService(
			mus.WithLocation(zp.location),
			mus.WithClient(zp.client),
		),
		QPlay: ply.NewService(
			ply.WithLocation(zp.location),
			ply.WithClient(zp.client),
		),
		Queue: que.NewService(
			que.WithLocation(zp.location),
			que.WithClient(zp.client),
		),
		RenderingControl: ren.NewService(
			ren.WithLocation(zp.location),
			ren.WithClient(zp.client),
		),
		SystemProperties: sys.NewService(
			sys.WithLocation(zp.location),
			sys.WithClient(zp.client),
		),
		VirtualLineIn: vli.NewService(
			vli.WithLocation(zp.location),
			vli.WithClient(zp.client),
		),
		ZoneGroupTopology: zgt.NewService(
			zgt.WithLocation(zp.location),
			zgt.WithClient(zp.client),
		),
	}

	return zp, nil
}

// Client returns the underlying http client.
func (z *ZonePlayer) Client() *http.Client {
	return z.client
}

func (z *ZonePlayer) Location() *url.URL {
	return z.location
}

func (z *ZonePlayer) RoomName() string {
	return z.Root.Device.RoomName
}

func (z *ZonePlayer) ModelName() string {
	return z.Root.Device.ModelName
}

func (z *ZonePlayer) HardwareVersion() string {
	return z.Root.Device.HardwareVersion
}

func (z *ZonePlayer) SoftwareVersion() string {
	return z.Root.Device.SoftwareVersion
}

func (z *ZonePlayer) SerialNumber() string {
	return z.Root.Device.SerialNum
}

func (z *ZonePlayer) MACAddress() string {
	return z.Root.Device.MACAddress
}

func (z *ZonePlayer) ModelDescription() string {
	return z.Root.Device.ModelDescription
}

func (z *ZonePlayer) UUID() string {
	return strings.Split(z.Root.Device.UDN, ":")[1]
}

func (z *ZonePlayer) IsCoordinator() bool {
	zoneGroupState, err := z.GetZoneGroupState()
	if err != nil {
		return false
	}
	for _, group := range zoneGroupState.ZoneGroups {
		if "uuid:"+group.Coordinator == z.Root.Device.UDN {
			return true
		}
	}

	return false
}

func (z *ZonePlayer) GetZoneGroupState() (*ZoneGroupState, error) {
	zoneGroupStateResponse, err := z.ZoneGroupTopology.GetZoneGroupState(&zgt.GetZoneGroupStateArgs{})
	if err != nil {
		return nil, err
	}
	var zoneGroupState ZoneGroupState
	err = xml.Unmarshal([]byte(zoneGroupStateResponse.ZoneGroupState), &zoneGroupState)
	if err != nil {
		return nil, err
	}

	return &zoneGroupState, nil
}

func (z *ZonePlayer) GetVolume() (int, error) {
	res, err := z.RenderingControl.GetVolume(&ren.GetVolumeArgs{Channel: "Master"})
	if err != nil {
		return 0, err
	}

	return int(res.CurrentVolume), err
}

func (z *ZonePlayer) GetGroupVolume() (int, error) {
	res, err := z.GroupRenderingControl.GetGroupVolume(&rcg.GetGroupVolumeArgs{InstanceID: 0})
	if err != nil {
		return 0, err
	}

	return int(res.CurrentVolume), err
}

func (z *ZonePlayer) SetVolume(desiredVolume int) error {
	_, err := z.RenderingControl.SetVolume(&ren.SetVolumeArgs{
		Channel:       "Master",
		DesiredVolume: uint16(desiredVolume)})
	return err
}

func (z *ZonePlayer) SetGroupVolume(desiredVolume int) error {
	_, err := z.GroupRenderingControl.SnapshotGroupVolume(&rcg.SnapshotGroupVolumeArgs{})
	if err != nil {
		return err
	}
	_, err = z.GroupRenderingControl.SetGroupVolume(&rcg.SetGroupVolumeArgs{
		InstanceID:    0,
		DesiredVolume: uint16(desiredVolume),
	})
	return err
}

func (z *ZonePlayer) Play() error {
	_, err := z.AVTransport.Play(&avt.PlayArgs{
		Speed: "1",
	})
	return err
}

func (z *ZonePlayer) Stop() error {
	_, err := z.AVTransport.Stop(&avt.StopArgs{})
	return err
}

func (z *ZonePlayer) Pause() error {
	_, err := z.AVTransport.Pause(&avt.PauseArgs{InstanceID: 0})
	return err
}

func (z *ZonePlayer) Next() error {
	_, err := z.AVTransport.Next(&avt.NextArgs{InstanceID: 0})
	return err
}

func (z *ZonePlayer) Previous() error {
	_, err := z.AVTransport.Previous(&avt.PreviousArgs{InstanceID: 0})
	return err
}

func (z *ZonePlayer) GetPositionInfo() (*avt.GetPositionInfoResponse, error) {
	return z.AVTransport.GetPositionInfo(&avt.GetPositionInfoArgs{InstanceID: 0})
}

func (z *ZonePlayer) GetZoneGroupAttributes() (*zgt.GetZoneGroupAttributesResponse, error) {
	return z.ZoneGroupTopology.GetZoneGroupAttributes(&zgt.GetZoneGroupAttributesArgs{})
}

func (z *ZonePlayer) ListQueue() ([]didl.Item, error) {
	browseRes, err := z.Queue.Browse(&que.BrowseArgs{QueueID: 0, StartingIndex: 0, RequestedCount: 100})
	if err != nil {
		return nil, err
	}

	var lite didl.Lite
	if err := xml.Unmarshal([]byte(browseRes.Result), &lite); err != nil {
		return nil, err
	}

	return lite.Item, nil
}

func (z *ZonePlayer) Mute() error {
	_, err := z.RenderingControl.SetMute(&ren.SetMuteArgs{InstanceID: 0, Channel: "Master", DesiredMute: true})
	return err
}

func (z *ZonePlayer) Unmute() error {
	_, err := z.RenderingControl.SetMute(&ren.SetMuteArgs{InstanceID: 0, Channel: "Master", DesiredMute: false})
	return err
}

func (z *ZonePlayer) IsMuted() (bool, error) {
	res, err := z.RenderingControl.GetMute(&ren.GetMuteArgs{InstanceID: 0, Channel: "Master"})
	if err != nil {
		return false, err
	}
	return res.CurrentMute, nil
}

func (z *ZonePlayer) GetAudioInputAttributes() (*ain.GetAudioInputAttributesResponse, error) {
	return z.AudioIn.GetAudioInputAttributes(&ain.GetAudioInputAttributesArgs{})
}

func (z *ZonePlayer) SetAudioInputAttributes(desiredName, desiredIcon string) error {
	_, err := z.AudioIn.SetAudioInputAttributes(&ain.SetAudioInputAttributesArgs{DesiredName: desiredName, DesiredIcon: desiredIcon})
	return err
}

func (z *ZonePlayer) GetLineInLevel() (*ain.GetLineInLevelResponse, error) {
	return z.AudioIn.GetLineInLevel(&ain.GetLineInLevelArgs{})
}

func (z *ZonePlayer) SetLineInLevel(desiredLeftLineInLevel, desiredRightLineInLevel int32) error {
	_, err := z.AudioIn.SetLineInLevel(&ain.SetLineInLevelArgs{DesiredLeftLineInLevel: desiredLeftLineInLevel, DesiredRightLineInLevel: desiredRightLineInLevel})
	return err
}

func (z *ZonePlayer) SelectAudio(objectID string) error {
	_, err := z.AudioIn.SelectAudio(&ain.SelectAudioArgs{ObjectID: objectID})
	return err
}

func (z *ZonePlayer) GetZoneInfo() (*dev.GetZoneInfoResponse, error) {
	return z.DeviceProperties.GetZoneInfo(&dev.GetZoneInfoArgs{})
}

func (z *ZonePlayer) SwitchToLineIn() error {
	uuid := z.UUID()
	_, err := z.AVTransport.SetAVTransportURI(&avt.SetAVTransportURIArgs{InstanceID: 0, CurrentURI: fmt.Sprintf("x-rincon-stream:%s", uuid), CurrentURIMetaData: ""})
	return err
}

func (z *ZonePlayer) SwitchToQueue() error {
	uuid := z.UUID()
	_, err := z.AVTransport.SetAVTransportURI(&avt.SetAVTransportURIArgs{InstanceID: 0, CurrentURI: fmt.Sprintf("x-rincon-queue:%s#0", uuid), CurrentURIMetaData: ""})
	return err
}

func (z *ZonePlayer) SetAVTransportURI(url string) error {
	_, err := z.AVTransport.SetAVTransportURI(&avt.SetAVTransportURIArgs{
		CurrentURI: url,
	})
	return err
}

func (zp *ZonePlayer) Event(evt interface{}, fn EventHandlerFunc) {
	switch e := evt.(type) {
	case avt.LastChange:
		var levt AVTransportLastChange
		err := xml.Unmarshal([]byte(e), &levt)
		if err != nil {
			fmt.Printf("Unmarshal failure: %s", err)
			return
		}
		fn(levt)
	case ren.LastChange:
		var levt RenderingControlLastChange
		err := xml.Unmarshal([]byte(e), &levt)
		if err != nil {
			fmt.Printf("Unmarshal failure: %s", err)
			return
		}
		fn(levt)

	case que.LastChange:
		var levt QueueLastChange
		err := xml.Unmarshal([]byte(e), &levt)
		if err != nil {
			fmt.Printf("Unmarshal failure: %s", err)
			return
		}
		fn(levt)
	case zgt.AvailableSoftwareUpdate:
		var levt ZoneGroupTopologyAvailableSoftwareUpdate
		err := xml.Unmarshal([]byte(e), &levt)
		if err != nil {
			fmt.Printf("Unmarshal failure: %s", err)
			return
		}
		fn(levt)
	case zgt.ZoneGroupState:
		var levt ZoneGroupTopologyZoneGroupState
		err := xml.Unmarshal([]byte(e), &levt)
		if err != nil {
			fmt.Printf("Unmarshal failure: %s", err)
			return
		}
		fn(levt)
	// AlarmClock
	case clk.TimeZone:
		fn(e)
	case clk.TimeServer:
		fn(e)
	case clk.TimeGeneration:
		fn(e)
	case clk.AlarmListVersion:
		fn(e)
	case clk.DailyIndexRefreshTime:
		fn(e)
	case clk.TimeFormat:
		fn(e)
	case clk.DateFormat:
		fn(e)
	// AudioIn
	case ain.AudioInputName:
		fn(e)
	case ain.Icon:
		fn(e)
	case ain.LineInConnected:
		fn(e)
	case ain.LeftLineInLevel:
		fn(e)
	case ain.RightLineInLevel:
		fn(e)
	case ain.Playing:
		fn(e)
	// ConnectionManager
	case con.SourceProtocolInfo:
		fn(e)
	case con.SinkProtocolInfo:
		fn(e)
	case con.CurrentConnectionIDs:
		fn(e)
	// ContentDirectory
	case dir.SystemUpdateID:
		fn(e)
	case dir.ContainerUpdateIDs:
		fn(e)
	case dir.ShareIndexInProgress:
		fn(e)
	case dir.ShareIndexLastError:
		fn(e)
	case dir.UserRadioUpdateID:
		fn(e)
	case dir.SavedQueuesUpdateID:
		fn(e)
	case dir.ShareListUpdateID:
		fn(e)
	case dir.RecentlyPlayedUpdateID:
		fn(e)
	case dir.Browseable:
		fn(e)
	case dir.RadioFavoritesUpdateID:
		fn(e)
	case dir.RadioLocationUpdateID:
		fn(e)
	case dir.FavoritesUpdateID:
		fn(e)
	case dir.FavoritePresetsUpdateID:
		fn(e)
	// DeviceProperties
	case dev.SettingsReplicationState:
		fn(e)
	case dev.ZoneName:
		fn(e)
	case dev.Icon:
		fn(e)
	case dev.Configuration:
		fn(e)
	case dev.Invisible:
		fn(e)
	case dev.IsZoneBridge:
		fn(e)
	case dev.AirPlayEnabled:
		fn(e)
	case dev.SupportsAudioIn:
		fn(e)
	case dev.SupportsAudioClip:
		fn(e)
	case dev.IsIdle:
		fn(e)
	case dev.MoreInfo:
		fn(e)
	case dev.ChannelMapSet:
		fn(e)
	case dev.HTSatChanMapSet:
		fn(e)
	case dev.HTBondedZoneCommitState:
		fn(e)
	case dev.Orientation:
		fn(e)
	case dev.LastChangedPlayState:
		fn(e)
	case dev.RoomCalibrationState:
		fn(e)
	case dev.AvailableRoomCalibration:
		fn(e)
	case dev.TVConfigurationError:
		fn(e)
	case dev.HdmiCecAvailable:
		fn(e)
	case dev.WirelessMode:
		fn(e)
	case dev.WirelessLeafOnly:
		fn(e)
	case dev.HasConfiguredSSID:
		fn(e)
	case dev.ChannelFreq:
		fn(e)
	case dev.BehindWifiExtender:
		fn(e)
	case dev.WifiEnabled:
		fn(e)
	case dev.EthLink:
		fn(e)
	case dev.ConfigMode:
		fn(e)
	case dev.SecureRegState:
		fn(e)
	case dev.VoiceConfigState:
		fn(e)
	case dev.MicEnabled:
		fn(e)
	// GroupManagement
	case gmn.GroupCoordinatorIsLocal:
		fn(e)
	case gmn.LocalGroupUUID:
		fn(e)
	case gmn.VirtualLineInGroupID:
		fn(e)
	case gmn.ResetVolumeAfter:
		fn(e)
	case gmn.VolumeAVTransportURI:
		fn(e)
	// GroupRenderingControl
	case rcg.GroupMute:
		fn(e)
	case rcg.GroupVolume:
		fn(e)
	case rcg.GroupVolumeChangeable:
		fn(e)
	// MusicServices
	case mus.ServiceListVersion:
		fn(e)
	// SystemProperties
	case sys.CustomerID:
		fn(e)
	case sys.UpdateID:
		fn(e)
	case sys.UpdateIDX:
		fn(e)
	case sys.VoiceUpdateID:
		fn(e)
	case sys.ThirdPartyHash:
		fn(e)
	// VirtualLineIn
	case vli.CurrentTrackMetaData:
		fn(e)
	// ZoneGroupTopology
	case zgt.ThirdPartyMediaServersX:
		fn(e)
	case zgt.AlarmRunSequence:
		fn(e)
	case zgt.MuseHouseholdId:
		fn(e)
	case zgt.ZoneGroupName:
		fn(e)
	case zgt.ZoneGroupID:
		fn(e)
	case zgt.ZonePlayerUUIDsInGroup:
		fn(e)
	case zgt.AreasUpdateID:
		fn(e)
	case zgt.SourceAreasUpdateID:
		fn(e)
	case zgt.NetsettingsUpdateID:
		fn(e)
	default:
		fmt.Printf("Unhandeld event %T: %s\n", e, e)
	}
}
