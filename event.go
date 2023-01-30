package sonos

import (
	"fmt"
	"strings"

	"github.com/kr/pretty"
)

// http://upnp.org/specs/av/UPnP-av-AVTransport-v1-Service.pdf
type AVTransportLastChange struct {
	InstanceID struct {
		TransportState struct {
			Value string `xml:"val,attr"`
		} `xml:"TransportState"`
		CurrentPlayMode struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentPlayMode"`
		CurrentCrossfadeMode struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentCrossfadeMode"`
		NumberOfTracks struct {
			Value string `xml:"val,attr"`
		} `xml:"NumberOfTracks"`
		CurrentTrack struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentTrack"`
		CurrentSection struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentSection"`
		CurrentTrackURI struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentTrackURI"`
		CurrentTrackDuration struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentTrackDuration"`
		CurrentTrackMetaData struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentTrackMetaData"`
		NextTrackURI struct {
			Value string `xml:"val,attr"`
		} `xml:"NextTrackURI"`
		NextTrackMetaData struct {
			Value string `xml:"val,attr"`
		} `xml:"NextTrackMetaData"`
		EnqueuedTransportURI struct {
			Value string `xml:"val,attr"`
		} `xml:"EnqueuedTransportURI"`
		EnqueuedTransportURIMetaData struct {
			Value string `xml:"val,attr"`
		} `xml:"EnqueuedTransportURIMetaData"`
		PlaybackStorageMedium struct {
			Value string `xml:"val,attr"`
		} `xml:"PlaybackStorageMedium"`
		AVTransportURI struct {
			Value string `xml:"val,attr"`
		} `xml:"AVTransportURI"`
		AVTransportURIMetaData struct {
			Value string `xml:"val,attr"`
		} `xml:"AVTransportURIMetaData"`
		NextAVTransportURI struct {
			Value string `xml:"val,attr"`
		} `xml:"NextAVTransportURI"`
		NextAVTransportURIMetaData struct {
			Value string `xml:"val,attr"`
		} `xml:"NextAVTransportURIMetaData"`
		CurrentTransportActions struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentTransportActions"`
		CurrentValidPlayModes struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentValidPlayModes"`
		DirectControlClientID struct {
			Value string `xml:"val,attr"`
		} `xml:"DirectControlClientID"`
		DirectControlIsSuspended struct {
			Value string `xml:"val,attr"`
		} `xml:"DirectControlIsSuspended"`
		DirectControlAccountID struct {
			Value string `xml:"val,attr"`
		} `xml:"DirectControlAccountID"`
		TransportStatus struct {
			Value string `xml:"val,attr"`
		} `xml:"TransportStatus"`
		SleepTimerGeneration struct {
			Value string `xml:"val,attr"`
		} `xml:"SleepTimerGeneration"`
		AlarmRunning struct {
			Value string `xml:"val,attr"`
		} `xml:"AlarmRunning"`
		SnoozeRunning struct {
			Value string `xml:"val,attr"`
		} `xml:"SnoozeRunning"`
		RestartPending struct {
			Value string `xml:"val,attr"`
		} `xml:"RestartPending"`
		TransportPlaySpeed struct {
			Value string `xml:"val,attr"`
		} `xml:"TransportPlaySpeed"`
		CurrentMediaDuration struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentMediaDuration"`
		RecordStorageMedium struct {
			Value string `xml:"val,attr"`
		} `xml:"RecordStorageMedium"`
		PossiblePlaybackStorageMedia struct {
			Value string `xml:"val,attr"`
		} `xml:"PossiblePlaybackStorageMedia"`
		PossibleRecordStorageMedia struct {
			Value string `xml:"val,attr"`
		} `xml:"PossibleRecordStorageMedia"`
		RecordMediumWriteStatus struct {
			Value string `xml:"val,attr"`
		} `xml:"RecordMediumWriteStatus"`
		CurrentRecordQualityMode struct {
			Value string `xml:"val,attr"`
		} `xml:"CurrentRecordQualityMode"`
		PossibleRecordQualityModes struct {
			Value string `xml:"val,attr"`
		} `xml:"PossibleRecordQualityModes"`
	} `xml:"InstanceID"`
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
	InstanceID struct {
		Volume []struct {
			Channel string `xml:"channel,attr"`
			Value   string `xml:"val,attr"`
		} `xml:"Volume"`
		Mute []struct {
			Channel string `xml:"channel,attr"`
			Value   string `xml:"val,attr"`
		} `xml:"Mute"`
		Bass struct {
			Value string `xml:"val,attr"`
		} `xml:"Bass"`
		Treble struct {
			Value string `xml:"val,attr"`
		} `xml:"Treble"`
		Loudness struct {
			Channel string `xml:"channel,attr"`
			Value   string `xml:"val,attr"`
		} `xml:"Loudness"`
		OutputFixed struct {
			Value string `xml:"val,attr"`
		} `xml:"OutputFixed"`
		SpeakerSize struct {
			Value string `xml:"val,attr"`
		} `xml:"SpeakerSize"`
		SubGain struct {
			Value string `xml:"val,attr"`
		} `xml:"SubGain"`
		SubCrossover struct {
			Value string `xml:"val,attr"`
		} `xml:"SubCrossover"`
		SubPolarity struct {
			Value string `xml:"val,attr"`
		} `xml:"SubPolarity"`
		SubEnabled struct {
			Value string `xml:"val,attr"`
		} `xml:"SubEnabled"`
		SonarEnabled struct {
			Value string `xml:"val,attr"`
		} `xml:"SonarEnabled"`
		SonarCalibrationAvailable struct {
			Value string `xml:"val,attr"`
		} `xml:"SonarCalibrationAvailable"`
		PresetNameList struct {
			Value string `xml:"val,attr"`
		} `xml:"PresetNameList"`
	} `xml:"InstanceID"`
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
	QueueID []struct {
		Value    string `xml:"val,attr"`
		UpdateID struct {
			Value string `xml:"val,attr"`
		} `xml:"UpdateID"`
		Curated struct {
			Value string `xml:"val,attr"`
		} `xml:"Curated"`
		QueueOwnerID struct {
			Value string `xml:"val,attr"`
		} `xml:"QueueOwnerID"`
	} `xml:"QueueID"`
}

func (e *QueueLastChange) String() string {
	return pretty.Sprintf("%# v", e)
}

type ZoneGroupTopologyZoneGroupState struct {
	ZoneGroups struct {
		ZoneGroup []struct {
			Coordinator     string `xml:"Coordinator,attr"`
			ID              string `xml:"ID,attr"`
			ZoneGroupMember []struct {
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
			} `xml:"ZoneGroupMember"`
		} `xml:"ZoneGroup"`
	} `xml:"ZoneGroups"`
	VanishedDevices string `xml:"VanishedDevices"`
}

func (e *ZoneGroupTopologyZoneGroupState) String() string {
	return pretty.Sprintf("%# v", e)
}
