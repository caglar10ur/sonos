package main

import (
	"context"
	"fmt"
	"time"

	"github.com/caglar10ur/sonos"
	avtransport "github.com/caglar10ur/sonos/services/AVTransport"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := sonos.NewSonos()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer s.Close()

	f := func(s *sonos.Sonos, player *sonos.ZonePlayer) {
		fmt.Printf("%s\t%s\t%s\n", player.RoomName(), player.ModelName(), player.SerialNumber())

		az, err := player.AVTransport.GetPositionInfo(&avtransport.GetPositionInfoArgs{})
		if err != nil {
			fmt.Printf("%s", err)
			return

		}

		if len(az.TrackMetaData) == 0 {
			return
		}

		metadata, err := sonos.ParseDIDL(az.TrackMetaData)
		if err != nil {
			fmt.Printf("%s", err)
			return
		}

		fmt.Printf("### Now playing ###\n")
		for _, m := range metadata.Item {
			if len(m.Title) > 0 {
				fmt.Printf("Title: %s\n", m.Title[0].Value)
			}
			if len(m.Album) > 0 {
				fmt.Printf("Album: %s\n", m.Album[0].Value)
			}
			if len(m.Creator) > 0 {
				fmt.Printf("Creator: %s\n\n", m.Creator[0].Value)
			}
		}
	}

	err = s.Search(ctx, f)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	<-ctx.Done()
}
