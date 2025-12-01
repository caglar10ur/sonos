package sonos

import (
	"fmt"
	"strings"

	"github.com/kr/pretty"
)

// http://upnp.org/specs/av/UPnP-av-AVTransport-v1-Service.pdf
type AVTransportLastChange struct {
	InstanceID AVTransportInstanceID `xml:"InstanceID"`
}

type AVTransportInstanceID struct {
	TransportState               TransportState               `xml:"TransportState"`
	CurrentPlayMode              CurrentPlayMode              `xml:"CurrentPlayMode"`
	CurrentCrossfadeMode         CurrentCrossfadeMode         `xml:"CurrentCrossfadeMode"`
	NumberOfTracks               NumberOfTracks               `xml:"NumberOfTracks"`
	CurrentTrack                 CurrentTrack                 `xml:"CurrentTrack"`
	CurrentSection               CurrentSection               `xml:"CurrentSection"`
	CurrentTrackURI              CurrentTrackURI              `xml:"CurrentTrackURI"`
	CurrentTrackDuration         CurrentTrackDuration         `xml:"CurrentTrackDuration"`
	CurrentTrackMetaData         CurrentTrackMetaData         `xml:"CurrentTrackMetaData"`
	NextTrackURI                 NextTrackURI                 `xml:"NextTrackURI"`
	NextTrackMetaData            NextTrackMetaData            `xml:"NextTrackMetaData"`
	EnqueuedTransportURI         EnqueuedTransportURI         `xml:"EnqueuedTransportURI"`
	EnqueuedTransportURIMetaData EnqueuedTransportURIMetaData `xml:"EnqueuedTransportURIMetaData"`
	PlaybackStorageMedium        PlaybackStorageMedium        `xml:"PlaybackStorageMedium"`
	AVTransportURI               AVTransportURI               `xml:"AVTransportURI"`
	AVTransportURIMetaData       AVTransportURIMetaData       `xml:"AVTransportURIMetaData"`
	NextAVTransportURI           NextAVTransportURI           `xml:"NextAVTransportURI"`
	NextAVTransportURIMetaData   NextAVTransportURIMetaData   `xml:"NextAVTransportURIMetaData"`
	CurrentTransportActions      CurrentTransportActions      `xml:"CurrentTransportActions"`
	CurrentValidPlayModes        CurrentValidPlayModes        `xml:"CurrentValidPlayModes"`
	DirectControlClientID        DirectControlClientID        `xml:"DirectControlClientID"`
	DirectControlIsSuspended     DirectControlIsSuspended     `xml:"DirectControlIsSuspended"`
	DirectControlAccountID       DirectControlAccountID       `xml:"DirectControlAccountID"`
	TransportStatus              TransportStatus              `xml:"TransportStatus"`
	SleepTimerGeneration         SleepTimerGeneration         `xml:"SleepTimerGeneration"`
	AlarmRunning                 AlarmRunning                 `xml:"AlarmRunning"`
	SnoozeRunning                SnoozeRunning                `xml:"SnoozeRunning"`
	RestartPending               RestartPending               `xml:"RestartPending"`
	TransportPlaySpeed           TransportPlaySpeed           `xml:"TransportPlaySpeed"`
	CurrentMediaDuration         CurrentMediaDuration         `xml:"CurrentMediaDuration"`
	RecordStorageMedium          RecordStorageMedium          `xml:"RecordStorageMedium"`
	PossiblePlaybackStorageMedia PossiblePlaybackStorageMedia `xml:"PossiblePlaybackStorageMedia"`
	PossibleRecordStorageMedia   PossibleRecordStorageMedia   `xml:"PossibleRecordStorageMedia"`
	RecordMediumWriteStatus      RecordMediumWriteStatus      `xml:"RecordMediumWriteStatus"`
	CurrentRecordQualityMode     CurrentRecordQualityMode     `xml:"CurrentRecordQualityMode"`
	PossibleRecordQualityModes   PossibleRecordQualityModes   `xml:"PossibleRecordQualityModes"`
}

type TransportState struct {
	Value string `xml:"val,attr"`
}
type CurrentPlayMode struct {
	Value string `xml:"val,attr"`
}
type CurrentCrossfadeMode struct {
	Value string `xml:"val,attr"`
}
type NumberOfTracks struct {
	Value string `xml:"val,attr"`
}
type CurrentTrack struct {
	Value string `xml:"val,attr"`
}
type CurrentSection struct {
	Value string `xml:"val,attr"`
}
type CurrentTrackURI struct {
	Value string `xml:"val,attr"`
}
type CurrentTrackDuration struct {
	Value string `xml:"val,attr"`
}
type CurrentTrackMetaData struct {
	Value string `xml:"val,attr"`
}
type NextTrackURI struct {
	Value string `xml:"val,attr"`
}
type NextTrackMetaData struct {
	Value string `xml:"val,attr"`
}
type EnqueuedTransportURI struct {
	Value string `xml:"val,attr"`
}
type EnqueuedTransportURIMetaData struct {
	Value string `xml:"val,attr"`
}
type PlaybackStorageMedium struct {
	Value string `xml:"val,attr"`
}
type AVTransportURI struct {
	Value string `xml:"val,attr"`
}
type AVTransportURIMetaData struct {
	Value string `xml:"val,attr"`
}
type NextAVTransportURI struct {
	Value string `xml:"val,attr"`
}
type NextAVTransportURIMetaData struct {
	Value string `xml:"val,attr"`
}
type CurrentTransportActions struct {
	Value string `xml:"val,attr"`
}
type CurrentValidPlayModes struct {
	Value string `xml:"val,attr"`
}
type DirectControlClientID struct {
	Value string `xml:"val,attr"`
}
type DirectControlIsSuspended struct {
	Value string `xml:"val,attr"`
}
type DirectControlAccountID struct {
	Value string `xml:"val,attr"`
}
type TransportStatus struct {
	Value string `xml:"val,attr"`
}
type SleepTimerGeneration struct {
	Value string `xml:"val,attr"`
}
type AlarmRunning struct {
	Value string `xml:"val,attr"`
}
type SnoozeRunning struct {
	Value string `xml:"val,attr"`
}
type RestartPending struct {
	Value string `xml:"val,attr"`
}
type TransportPlaySpeed struct {
	Value string `xml:"val,attr"`
}
type CurrentMediaDuration struct {
	Value string `xml:"val,attr"`
}
type RecordStorageMedium struct {
	Value string `xml:"val,attr"`
}
type PossiblePlaybackStorageMedia struct {
	Value string `xml:"val,attr"`
}
type PossibleRecordStorageMedia struct {
	Value string `xml:"val,attr"`
}
type RecordMediumWriteStatus struct {
	Value string `xml:"val,attr"`
}
type CurrentRecordQualityMode struct {
	Value string `xml:"val,attr"`
}
type PossibleRecordQualityModes struct {
	Value string `xml:"val,attr"`
}

func (e *AVTransportLastChange) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "TransportState: %s\n", e.InstanceID.TransportState.Value)
	fmt.Fprintf(&b, "CurrentPlayMode %s\n", e.InstanceID.CurrentPlayMode.Value)
	fmt.Fprintf(&b, "NumberOfTracks: %s\n", e.InstanceID.NumberOfTracks.Value)
	fmt.Fprintf(&b, "CurrentTrack: %s\n", e.InstanceID.CurrentTrack.Value)
	fmt.Fprintf(&b, "CurrentTrackDuration: %s\n", e.InstanceID.CurrentTrackDuration.Value)
	fmt.Fprintf(&b, "CurrentTrackURI: %s\n", e.InstanceID.CurrentTrackURI.Value)

	metadata, err := ParseDIDL(e.InstanceID.CurrentTrackMetaData.Value)
	if err == nil && len(metadata.Item) > 0 {
		m := metadata.Item[0]

		if len(m.Title) > 0 {
			fmt.Fprintf(&b, "CurrentTrackMetaData>Title: %s\n", m.Title[0].Value)
		}
		if len(m.Album) > 0 {
			fmt.Fprintf(&b, "CurrentTrackMetaData>Album: %s\n", m.Album[0].Value)
		}
		if len(m.Creator) > 0 {
			fmt.Fprintf(&b, "CurrentTrackMetaData>Creator: %s\n", m.Creator[0].Value)
		}
		if len(m.AlbumArtURI) > 0 {
			fmt.Fprintf(&b, "CurrentTrackMetaData>AlbumArtURI: %s\n", m.AlbumArtURI[0].Value)
		}
	}

	fmt.Fprintf(&b, "NextTrackURI: %s\n", e.InstanceID.NextTrackURI.Value)
	metadata, err = ParseDIDL(e.InstanceID.NextTrackMetaData.Value)
	if err == nil && len(metadata.Item) > 0 {
		m := metadata.Item[0]

		if len(m.Title) > 0 {
			fmt.Fprintf(&b, "NextTrackMetaData>Title: %s\n", m.Title[0].Value)
		}
		if len(m.Album) > 0 {
			fmt.Fprintf(&b, "NextTrackMetaData>Album: %s\n", m.Album[0].Value)
		}
		if len(m.Creator) > 0 {
			fmt.Fprintf(&b, "NextTrackMetaData>Creator: %s\n", m.Creator[0].Value)
		}
		if len(m.AlbumArtURI) > 0 {
			fmt.Fprintf(&b, "NextTrackMetaData>AlbumArtURI: %s\n", m.AlbumArtURI[0].Value)
		}
	}

	return b.String()
}

// http://upnp.org/specs/av/UPnP-av-RenderingControl-v1-Service.pdf
type RenderingControlLastChange struct {
	InstanceID RenderingControlInstanceID `xml:"InstanceID"`
}

type RenderingControlInstanceID struct {
	Volume                    []Volume                  `xml:"Volume"`
	Mute                      []Mute                    `xml:"Mute"`
	Bass                      Bass                      `xml:"Bass"`
	Treble                    Treble                    `xml:"Treble"`
	Loudness                  Loudness                  `xml:"Loudness"`
	OutputFixed               OutputFixed               `xml:"OutputFixed"`
	SpeakerSize               SpeakerSize               `xml:"SpeakerSize"`
	SubGain                   SubGain                   `xml:"SubGain"`
	SubCrossover              SubCrossover              `xml:"SubCrossover"`
	SubPolarity               SubPolarity               `xml:"SubPolarity"`
	SubEnabled                SubEnabled                `xml:"SubEnabled"`
	SonarEnabled              SonarEnabled              `xml:"SonarEnabled"`
	SonarCalibrationAvailable SonarCalibrationAvailable `xml:"SonarCalibrationAvailable"`
	PresetNameList            PresetNameList            `xml:"PresetNameList"`
}

type Volume struct {
	Channel string `xml:"channel,attr"`
	Value   string `xml:"val,attr"`
}
type Mute struct {
	Channel string `xml:"channel,attr"`
	Value   string `xml:"val,attr"`
}
type Bass struct {
	Value string `xml:"val,attr"`
}
type Treble struct {
	Value string `xml:"val,attr"`
}
type Loudness struct {
	Channel string `xml:"channel,attr"`
	Value   string `xml:"val,attr"`
}
type OutputFixed struct {
	Value string `xml:"val,attr"`
}
type SpeakerSize struct {
	Value string `xml:"val,attr"`
}
type SubGain struct {
	Value string `xml:"val,attr"`
}
type SubCrossover struct {
	Value string `xml:"val,attr"`
}
type SubPolarity struct {
	Value string `xml:"val,attr"`
}
type SubEnabled struct {
	Value string `xml:"val,attr"`
}
type SonarEnabled struct {
	Value string `xml:"val,attr"`
}
type SonarCalibrationAvailable struct {
	Value string `xml:"val,attr"`
}
type PresetNameList struct {
	Value string `xml:"val,attr"`
}

func (e *RenderingControlLastChange) String() string {
	return pretty.Sprintf("%# v", e)
}

type ZoneGroupTopologyAvailableSoftwareUpdate struct {
	Type             string `xml:"Type,attr"`
	Version          string `xml:"Version,attr"`
	UpdateURL        string `xml:"UpdateURL,attr"`
	DownloadSize     string `xml:"DownloadSize,attr"`
	ManifestURL      string `xml:"ManifestURL,attr"`
	Swgen            string `xml:"Swgen,attr"`
	LatestSwgen      string `xml:"LatestSwgen,attr"`
	ManifestRevision string `xml:"ManifestRevision,attr"`
}

func (e *ZoneGroupTopologyAvailableSoftwareUpdate) String() string {
	return pretty.Sprintf("%# v", e)
}

type QueueLastChange struct {
	QueueID []QueueID `xml:"QueueID"`
}

type QueueID struct {
	Value        string       `xml:"val,attr"`
	UpdateID     UpdateID     `xml:"UpdateID"`
	Curated      Curated      `xml:"Curated"`
	QueueOwnerID QueueOwnerID `xml:"QueueOwnerID"`
}

type UpdateID struct {
	Value string `xml:"val,attr"`
}
type Curated struct {
	Value string `xml:"val,attr"`
}
type QueueOwnerID struct {
	Value string `xml:"val,attr"`
}

func (e *QueueLastChange) String() string {
	return pretty.Sprintf("%# v", e)
}

type ZoneGroupTopologyZoneGroupState struct {
	ZoneGroups      ZoneGroups `xml:"ZoneGroups"`
	VanishedDevices string     `xml:"VanishedDevices"`
}

type ZoneGroups struct {
	ZoneGroup []ZoneGroup `xml:"ZoneGroup"`
}

type EventZoneGroup struct {
	Coordinator     string                 `xml:"Coordinator,attr"`
	ID              string                 `xml:"ID,attr"`
	ZoneGroupMember []EventZoneGroupMember `xml:"ZoneGroupMember"`
}

type EventZoneGroupMember struct {
	UUID                    string `xml:"UUID,attr"`
	Location                string `xml:"Location,attr"`
	ZoneName                string `xml:"ZoneName,attr"`
	Icon                    string `xml:"Icon,attr"`
	Configuration           string `xml:"Configuration,attr"`
	SoftwareVersion         string `xml:"SoftwareVersion,attr"`
	SWGen                   string `xml:"SWGen,attr"`
	MinCompatibleVersion    string `xml:"MinCompatibleVersion,attr"`
	LegacyCompatibleVersion string `xml:"LegacyCompatibleVersion,attr"`
	ChannelMapSet           string `xml:"ChannelMapSet,attr"`
	BootSeq                 string `xml:"BootSeq,attr"`
	TVConfigurationError    string `xml:"TVConfigurationError,attr"`
	HdmiCecAvailable        string `xml:"HdmiCecAvailable,attr"`
	WirelessMode            string `xml:"WirelessMode,attr"`
	WirelessLeafOnly        string `xml:"WirelessLeafOnly,attr"`
	ChannelFreq             string `xml:"ChannelFreq,attr"`
	BehindWifiExtender      string `xml:"BehindWifiExtender,attr"`
	WifiEnabled             string `xml:"WifiEnabled,attr"`
	EthLink                 string `xml:"EthLink,attr"`
	Orientation             string `xml:"Orientation,attr"`
	RoomCalibrationState    string `xml:"RoomCalibrationState,attr"`
	SecureRegState          string `xml:"SecureRegState,attr"`
	VoiceConfigState        string `xml:"VoiceConfigState,attr"`
	MicEnabled              string `xml:"MicEnabled,attr"`
	AirPlayEnabled          string `xml:"AirPlayEnabled,attr"`
	IdleState               string `xml:"IdleState,attr"`
	MoreInfo                string `xml:"MoreInfo,attr"`
	SSLPort                 string `xml:"SSLPort,attr"`
	HHSSLPort               string `xml:"HHSSLPort,attr"`
	Invisible               string `xml:"Invisible,attr"`
	VirtualLineInSource     string `xml:"VirtualLineInSource,attr"`
}

func (e *ZoneGroupTopologyZoneGroupState) String() string {
	return pretty.Sprintf("%# v", e)
}
