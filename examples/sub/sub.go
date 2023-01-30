package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/caglar10ur/sonos"
)

var (
	room = flag.String("room", "Living Room", "Room name")
)

func init() {
	flag.Parse()
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	son, err := sonos.NewSonos()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer son.Close()

	zp, err := son.FindRoom(ctx, *room)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if !zp.IsCoordinator() {
		log.Fatalf("Not a coordinator")
	}

	f := func(evt interface{}) {
		aevt, ok := evt.(sonos.AVTransportLastChange)
		if ok {
			fmt.Printf("### AVTransport/LastChange\n\n\t%s\n", aevt.String())
		}
		revt, ok := evt.(sonos.RenderingControlLastChange)
		if ok {
			fmt.Printf("### RenderingControl/LastChange\n\n\t%s\n", revt.String())
		}
		qevt, ok := evt.(sonos.QueueLastChange)
		if ok {
			fmt.Printf("### Queue/QueueLastChange\n\n\t%s\n", qevt.String())
		}
		uevt, ok := evt.(sonos.ZoneGroupTopologyAvailableSoftwareUpdate)
		if ok {
			fmt.Printf("### ZoneGroupTopology/AvailableSoftwareUpdate\n\n\t%s\n", uevt.String())
		}
		zevt, ok := evt.(sonos.ZoneGroupTopologyZoneGroupState)
		if ok {
			fmt.Printf("### ZoneGroupTopology/ZoneGroupState\n\n\t%s\n", zevt.String())
		}
	}

	fmt.Printf("Connected to %s\t%s\t%s\n", zp.RoomName(), zp.ModelName(), zp.SerialNumber())

	services := []sonos.SonosService{
		zp.AlarmClock,
		zp.AudioIn,
		zp.AVTransport,
		zp.ConnectionManager,
		zp.ContentDirectory,
		zp.DeviceProperties,
		zp.GroupManagement,
		zp.GroupRenderingControl,
		zp.MusicServices,
		// zp.QPlay, // Not supported
		zp.Queue,
		zp.RenderingControl,
		zp.SystemProperties,
		zp.VirtualLineIn,
		zp.ZoneGroupTopology,
	}

	o2s := make(map[string]*sonos.SubscriptionOptions)
	for i := range services {
		opts := &sonos.SubscriptionOptions{
			ZonePlayer:   zp,
			Service:      services[i],
			EventHandler: f,
		}
		if err := opts.Validate(); err != nil {
			log.Fatalf("%s", err)
		}

		sid, err := son.Subscribe(ctx, opts)
		if err != nil {
			log.Fatalf("%s", err)
		}
		opts.SetSid(sid)

		o2s[sid] = opts
	}

	time.Sleep(10 * time.Second)
	for _, v := range o2s {
		fmt.Printf("Renewing %T\n", v.Service)
		err = son.Renew(ctx, v)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	time.Sleep(10 * time.Second)
	for _, v := range o2s {
		fmt.Printf("Unsubscribing %T\n", v.Service)
		err = son.Unsubscribe(ctx, v)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	fmt.Printf("Waiting...\n")
	<-ctx.Done()
}
