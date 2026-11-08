package sonos

import "encoding/xml"

// PlayerAttributes are the attributes the zone group topology reports for
// each player. Which ones are set depends on the player and its software.
type PlayerAttributes struct {
	UUID                    string `xml:"UUID,attr"`
	Location                string `xml:"Location,attr"`
	ZoneName                string `xml:"ZoneName,attr"`
	Icon                    string `xml:"Icon,attr"`
	Configuration           string `xml:"Configuration,attr"`
	SoftwareVersion         string `xml:"SoftwareVersion,attr"`
	SWGen                   string `xml:"SWGen,attr"`
	MinCompatibleVersion    string `xml:"MinCompatibleVersion,attr"`
	LegacyCompatibleVersion string `xml:"LegacyCompatibleVersion,attr"`
	BootSeq                 string `xml:"BootSeq,attr"`
	TVConfigurationError    string `xml:"TVConfigurationError,attr"`
	HdmiCecAvailable        string `xml:"HdmiCecAvailable,attr"`
	WirelessMode            string `xml:"WirelessMode,attr"`
	WirelessLeafOnly        string `xml:"WirelessLeafOnly,attr"`
	HasConfiguredSSID       string `xml:"HasConfiguredSSID,attr"`
	ChannelFreq             string `xml:"ChannelFreq,attr"`
	BehindWifiExtender      string `xml:"BehindWifiExtender,attr"`
	WifiEnabled             string `xml:"WifiEnabled,attr"`
	Orientation             string `xml:"Orientation,attr"`
	RoomCalibrationState    string `xml:"RoomCalibrationState,attr"`
	SecureRegState          string `xml:"SecureRegState,attr"`
	VoiceConfigState        string `xml:"VoiceConfigState,attr"`
	MicEnabled              string `xml:"MicEnabled,attr"`
	AirPlayEnabled          string `xml:"AirPlayEnabled,attr"`
	IdleState               string `xml:"IdleState,attr"`
	MoreInfo                string `xml:"MoreInfo,attr"`
	ChannelMapSet           string `xml:"ChannelMapSet,attr"`
	HTSatChanMapSet         string `xml:"HTSatChanMapSet,attr"`
	EthLink                 string `xml:"EthLink,attr"`
	SSLPort                 string `xml:"SSLPort,attr"`
	HHSSLPort               string `xml:"HHSSLPort,attr"`
	VirtualLineInSource     string `xml:"VirtualLineInSource,attr"`
	// Invisible is "1" for players that are part of another room: the
	// second speaker of a stereo pair, a bonded sub or a surround.
	Invisible string `xml:"Invisible,attr"`
}

// VanishedDevice is a player the household knows but can't currently reach.
type VanishedDevice struct {
	XMLName xml.Name `xml:"VanishedDevice"`
	PlayerAttributes
	Reason string `xml:"Reason,attr"`
}

// Satellite is a player bonded to a home theater room, such as a sub or a
// surround speaker.
type Satellite struct {
	XMLName xml.Name `xml:"Satellite"`
	PlayerAttributes
}

// ZoneGroupMember is a player in a zone group. Members with Invisible "1"
// are part of another member's room (see PlayerAttributes.Invisible).
type ZoneGroupMember struct {
	XMLName xml.Name `xml:"ZoneGroupMember"`
	PlayerAttributes
	Satellite []Satellite `xml:"Satellite"`
}

// ZoneGroup is a set of rooms playing together. Coordinator is the UUID of
// the member that plays for the group.
type ZoneGroup struct {
	XMLName         xml.Name          `xml:"ZoneGroup"`
	Coordinator     string            `xml:"Coordinator,attr"`
	ID              string            `xml:"ID,attr"`
	ZoneGroupMember []ZoneGroupMember `xml:"ZoneGroupMember"`
}

// ZoneGroupState is a household's zone group topology.
type ZoneGroupState struct {
	XMLName         xml.Name         `xml:"ZoneGroupState"`
	ZoneGroups      []ZoneGroup      `xml:"ZoneGroups>ZoneGroup"`
	VanishedDevices []VanishedDevice `xml:"VanishedDevices>VanishedDevice"`
}
