package main

import (
	"fmt"
	"log"

	"github.com/caglar10ur/sonos"
	avtransport "github.com/caglar10ur/sonos/services/AVTransport"
	contentdirectory "github.com/caglar10ur/sonos/services/ContentDirectory"
	grouprenderingcontrol "github.com/caglar10ur/sonos/services/GroupRenderingControl"
)

func main() {
	ips := []string{}
	for i := 13; i <= 18; i++ {
		ips = append(ips, fmt.Sprintf("192.168.10.%d", i))
	}

	var zps []*sonos.ZonePlayer
	for _, ip := range ips {
		u, err := sonos.FromEndpoint(ip)
		if err != nil {
			log.Fatalf("%s", err)
		}

		zp, err := sonos.NewZonePlayer(
			sonos.WithLocation(u),
		)
		if err != nil {
			log.Fatalf("%s", err)
		}
		fmt.Printf("Trying %s\t%s\t%s (coordinator %t)\n", zp.RoomName(), zp.ModelName(), zp.SerialNumber(), zp.IsCoordinator())

		if zp.IsCoordinator() {
			zps = append(zps, zp)
		}
	}

	for _, zp := range zps {
		fmt.Printf("Connected to %s\t%s\t%s (coordinator %t)\n", zp.RoomName(), zp.ModelName(), zp.SerialNumber(), zp.IsCoordinator())

		zp.GroupRenderingControl.SetGroupVolume(&grouprenderingcontrol.SetGroupVolumeArgs{
			DesiredVolume: 10,
		})

		az, err := zp.AVTransport.GetPositionInfo(&avtransport.GetPositionInfoArgs{})
		if err != nil {
			log.Fatalf("%s", err)
		}

		metadata, err := sonos.ParseDIDL(az.TrackMetaData)
		if err != nil {
			log.Fatalf("%s", err)
		}

		fmt.Printf("### Now playing ###\n")
		for _, m := range metadata.Item {
			if m.Title != nil {
				fmt.Printf("Title: %s\n", m.Title[0].Value)
			}
			if m.Album != nil {
				fmt.Printf("Album: %s\n", m.Album[0].Value)
			}
			if m.Creator != nil {
				fmt.Printf("Creator: %s\n\n", m.Creator[0].Value)
			}
		}

		ac, err := zp.ContentDirectory.Browse(
			&contentdirectory.BrowseArgs{
				ObjectID:       "Q:0",
				BrowseFlag:     "BrowseDirectChildren",
				Filter:         "dc:title,res,dc:creator,upnp:artist,upnp:album,upnp:albumArtURI",
				StartingIndex:  az.Track,
				RequestedCount: 3,
			})
		if err != nil {
			log.Fatalf("%s", err)
		}

		metadata, err = sonos.ParseDIDL(ac.Result)
		if err != nil {
			log.Fatalf("%s", err)
		}

		fmt.Printf("### Next ###\n")
		for _, m := range metadata.Item {
			if m.Title != nil {
				fmt.Printf("Title: %s\n", m.Title[0].Value)
			}
			if m.Album != nil {
				fmt.Printf("Album: %s\n", m.Album[0].Value)
			}
			if m.Creator != nil {
				fmt.Printf("Creator: %s\n\n", m.Creator[0].Value)
			}
		}
	}
}
