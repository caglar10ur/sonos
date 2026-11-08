package sonos

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/caglar10ur/sonos/didl"
	avt "github.com/caglar10ur/sonos/services/avtransport"
	que "github.com/caglar10ur/sonos/services/queue"
	ren "github.com/caglar10ur/sonos/services/renderingcontrol"
	zgt "github.com/caglar10ur/sonos/services/zonegrouptopology"
)

// dispatch passes an event to fn, decoding the XML payload of LastChange,
// AvailableSoftwareUpdate and ZoneGroupState events first.
func (s *Sonos) dispatch(evt any, fn EventHandler) {
	switch e := evt.(type) {
	case avt.LastChange:
		decodeEvent[AVTransportLastChange](s, string(e), fn)
	case ren.LastChange:
		decodeEvent[RenderingControlLastChange](s, string(e), fn)
	case que.LastChange:
		decodeEvent[QueueLastChange](s, string(e), fn)
	case zgt.AvailableSoftwareUpdate:
		decodeEvent[ZoneGroupTopologyAvailableSoftwareUpdate](s, string(e), fn)
	case zgt.ZoneGroupState:
		decodeEvent[ZoneGroupTopologyZoneGroupState](s, string(e), fn)
	default:
		fn(evt)
	}
}

// decodeEvent unmarshals an event's XML payload into T and passes it to fn
// by value.
func decodeEvent[T any](s *Sonos, raw string, fn EventHandler) {
	var v T
	if err := xml.Unmarshal([]byte(raw), &v); err != nil {
		s.logger.Warn("sonos: dropping event with malformed payload", "type", fmt.Sprintf("%T", v), "err", err)
		return
	}
	fn(v)
}

// dump renders an event as indented JSON for its String method.
func dump(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}

// Val is a state variable value in a LastChange event, e.g.
// <TransportState val="PLAYING"/>.
type Val struct {
	Value string `xml:"val,attr"`
}

// ChannelVal is a per-channel state variable value in a LastChange event,
// e.g. <Volume channel="Master" val="20"/>.
type ChannelVal struct {
	Channel string `xml:"channel,attr"`
	Value   string `xml:"val,attr"`
}

// http://upnp.org/specs/av/UPnP-av-AVTransport-v1-Service.pdf
type AVTransportLastChange struct {
	InstanceID AVTransportInstanceID `xml:"InstanceID"`
}

type AVTransportInstanceID struct {
	TransportState               Val `xml:"TransportState"`
	CurrentPlayMode              Val `xml:"CurrentPlayMode"`
	CurrentCrossfadeMode         Val `xml:"CurrentCrossfadeMode"`
	NumberOfTracks               Val `xml:"NumberOfTracks"`
	CurrentTrack                 Val `xml:"CurrentTrack"`
	CurrentSection               Val `xml:"CurrentSection"`
	CurrentTrackURI              Val `xml:"CurrentTrackURI"`
	CurrentTrackDuration         Val `xml:"CurrentTrackDuration"`
	CurrentTrackMetaData         Val `xml:"CurrentTrackMetaData"`
	NextTrackURI                 Val `xml:"NextTrackURI"`
	NextTrackMetaData            Val `xml:"NextTrackMetaData"`
	EnqueuedTransportURI         Val `xml:"EnqueuedTransportURI"`
	EnqueuedTransportURIMetaData Val `xml:"EnqueuedTransportURIMetaData"`
	PlaybackStorageMedium        Val `xml:"PlaybackStorageMedium"`
	AVTransportURI               Val `xml:"AVTransportURI"`
	AVTransportURIMetaData       Val `xml:"AVTransportURIMetaData"`
	NextAVTransportURI           Val `xml:"NextAVTransportURI"`
	NextAVTransportURIMetaData   Val `xml:"NextAVTransportURIMetaData"`
	CurrentTransportActions      Val `xml:"CurrentTransportActions"`
	CurrentValidPlayModes        Val `xml:"CurrentValidPlayModes"`
	DirectControlClientID        Val `xml:"DirectControlClientID"`
	DirectControlIsSuspended     Val `xml:"DirectControlIsSuspended"`
	DirectControlAccountID       Val `xml:"DirectControlAccountID"`
	TransportStatus              Val `xml:"TransportStatus"`
	SleepTimerGeneration         Val `xml:"SleepTimerGeneration"`
	AlarmRunning                 Val `xml:"AlarmRunning"`
	SnoozeRunning                Val `xml:"SnoozeRunning"`
	RestartPending               Val `xml:"RestartPending"`
	TransportPlaySpeed           Val `xml:"TransportPlaySpeed"`
	CurrentMediaDuration         Val `xml:"CurrentMediaDuration"`
	RecordStorageMedium          Val `xml:"RecordStorageMedium"`
	PossiblePlaybackStorageMedia Val `xml:"PossiblePlaybackStorageMedia"`
	PossibleRecordStorageMedia   Val `xml:"PossibleRecordStorageMedia"`
	RecordMediumWriteStatus      Val `xml:"RecordMediumWriteStatus"`
	CurrentRecordQualityMode     Val `xml:"CurrentRecordQualityMode"`
	PossibleRecordQualityModes   Val `xml:"PossibleRecordQualityModes"`
}

func (e AVTransportLastChange) String() string {
	var b strings.Builder
	st := e.InstanceID
	fmt.Fprintf(&b, "TransportState: %s\n", st.TransportState.Value)
	fmt.Fprintf(&b, "CurrentPlayMode: %s\n", st.CurrentPlayMode.Value)
	fmt.Fprintf(&b, "NumberOfTracks: %s\n", st.NumberOfTracks.Value)
	fmt.Fprintf(&b, "CurrentTrack: %s\n", st.CurrentTrack.Value)
	fmt.Fprintf(&b, "CurrentTrackDuration: %s\n", st.CurrentTrackDuration.Value)
	fmt.Fprintf(&b, "CurrentTrackURI: %s\n", st.CurrentTrackURI.Value)
	writeTrackMetaData(&b, "CurrentTrackMetaData", st.CurrentTrackMetaData.Value)
	fmt.Fprintf(&b, "NextTrackURI: %s\n", st.NextTrackURI.Value)
	writeTrackMetaData(&b, "NextTrackMetaData", st.NextTrackMetaData.Value)
	return b.String()
}

// writeTrackMetaData writes the title, album, creator and album art of a
// track's DIDL-Lite metadata, skipping what's missing.
func writeTrackMetaData(b *strings.Builder, label, raw string) {
	meta, err := didl.Parse(raw)
	if err != nil || len(meta.Item) == 0 {
		return
	}
	m := meta.Item[0]
	for _, f := range []struct {
		name  string
		texts []didl.Text
	}{
		{"Title", m.Title},
		{"Album", m.Album},
		{"Creator", m.Creator},
		{"AlbumArtURI", m.AlbumArtURI},
	} {
		if v := didl.First(f.texts); v != "" {
			fmt.Fprintf(b, "%s>%s: %s\n", label, f.name, v)
		}
	}
}

// http://upnp.org/specs/av/UPnP-av-RenderingControl-v1-Service.pdf
type RenderingControlLastChange struct {
	InstanceID RenderingControlInstanceID `xml:"InstanceID"`
}

type RenderingControlInstanceID struct {
	Volume                    []ChannelVal `xml:"Volume"`
	Mute                      []ChannelVal `xml:"Mute"`
	Bass                      Val          `xml:"Bass"`
	Treble                    Val          `xml:"Treble"`
	Loudness                  ChannelVal   `xml:"Loudness"`
	OutputFixed               Val          `xml:"OutputFixed"`
	SpeakerSize               Val          `xml:"SpeakerSize"`
	SubGain                   Val          `xml:"SubGain"`
	SubCrossover              Val          `xml:"SubCrossover"`
	SubPolarity               Val          `xml:"SubPolarity"`
	SubEnabled                Val          `xml:"SubEnabled"`
	SonarEnabled              Val          `xml:"SonarEnabled"`
	SonarCalibrationAvailable Val          `xml:"SonarCalibrationAvailable"`
	PresetNameList            Val          `xml:"PresetNameList"`
}

func (e RenderingControlLastChange) String() string {
	type plain RenderingControlLastChange
	return dump(plain(e))
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

func (e ZoneGroupTopologyAvailableSoftwareUpdate) String() string {
	type plain ZoneGroupTopologyAvailableSoftwareUpdate
	return dump(plain(e))
}

type QueueLastChange struct {
	QueueID []QueueID `xml:"QueueID"`
}

type QueueID struct {
	Value        string `xml:"val,attr"`
	UpdateID     Val    `xml:"UpdateID"`
	Curated      Val    `xml:"Curated"`
	QueueOwnerID Val    `xml:"QueueOwnerID"`
}

func (e QueueLastChange) String() string {
	type plain QueueLastChange
	return dump(plain(e))
}

// ZoneGroupTopologyZoneGroupState is the decoded ZoneGroupState event. It
// has the same layout as the GetZoneGroupState answer.
type ZoneGroupTopologyZoneGroupState ZoneGroupState

func (e ZoneGroupTopologyZoneGroupState) String() string {
	return dump(ZoneGroupState(e))
}
