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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	fmt.Printf("Connected to %s\t%s\t%s\n", zp.RoomName(), zp.ModelName(), zp.SerialNum())

	sid, err := son.Subscribe(ctx, zp, zp.AVTransport)
	if err != nil {
		log.Fatalf("%s", err)
	}

	time.Sleep(10 * time.Second)

	err = son.Renew(ctx, zp, zp.AVTransport, sid)
	if err != nil {
		log.Fatalf("%s", err)
	}

	time.Sleep(10 * time.Second)

	err = son.Unsubscribe(ctx, zp, zp.AVTransport, sid)
	if err != nil {
		log.Fatalf("%s", err)
	}

	<-ctx.Done()
}
