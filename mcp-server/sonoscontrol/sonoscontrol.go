package sonoscontrol

import (
	"context"
	"sync"
	"time"

	"github.com/caglar10ur/sonos"
)

const (
	defaultTimeout = 10 * time.Second
)

type SonosController struct {
	sonos *sonos.Sonos
	cache sync.Map
}

func NewSonosController() (*SonosController, error) {
	s, err := sonos.NewSonos()
	if err != nil {
		return nil, err
	}
	return &SonosController{
		sonos: s,
	}, nil
}

func (sc *SonosController) CachedRoom(ctx context.Context, roomName string) (*sonos.ZonePlayer, error) {
	if zp, ok := sc.cache.Load(roomName); ok {
		return zp.(*sonos.ZonePlayer), nil
	}

	zp, err := sc.sonos.FindRoom(ctx, roomName)
	if err != nil {
		return nil, err
	}

	sc.cache.Store(roomName, zp)
	return zp, nil
}

func (sc *SonosController) ListSonosDevices(ctx context.Context) ([]string, error) {
	var devices []string
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	sc.sonos.Search(ctx, func(s *sonos.Sonos, zp *sonos.ZonePlayer) {
		devices = append(devices, zp.RoomName())
		sc.cache.Store(zp.RoomName(), zp)
	})

	<-ctx.Done()

	return devices, nil
}
