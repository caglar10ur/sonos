package main

import (
	"context"
	"flag"
	"fmt"
	"os"
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
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	son, err := sonos.NewSonos()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer son.Close()

	zp, err := son.FindRoom(ctx, *room)
	if err != nil {
		fmt.Printf("FindRoom Error: %v\n", err)
		return
	}

	if err = zp.SetAVTransportURI(os.Args[2]); err != nil {
		fmt.Printf("SetAVTransportURI Error: %v\n", err)
		return
	}

	if err = zp.Play(); err != nil {
		fmt.Printf("Play Error: %v\n", err)
		return
	}
}
