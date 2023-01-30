package main

import (
	"context"
	"fmt"
	"time"

	"github.com/caglar10ur/sonos"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	son, err := sonos.NewSonos()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer son.Close()

	f := func(sonos *sonos.Sonos, player *sonos.ZonePlayer) {
		fmt.Printf("%s\t%s\t%s\n", player.RoomName(), player.ModelName(), player.SerialNumber())
	}

	err = son.Search(ctx, f)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	<-ctx.Done()
}
