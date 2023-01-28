package sonos

import (
	"fmt"
	"strings"
)

type InstanceID struct {
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
}

// http://upnp.org/specs/av/UPnP-av-RenderingControl-v1-Service.pdf
type RenderingControlLastChange struct {
	InstanceID InstanceID `xml:"InstanceID"`
}

// http://upnp.org/specs/av/UPnP-av-AVTransport-v1-Service.pdf
type AVTransportLastChange struct {
	InstanceID InstanceID `xml:"InstanceID"`
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
