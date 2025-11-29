package sonoscontrol

import (
	"context"
	"sync"

	"github.com/caglar10ur/sonos"
)

type SonosControllerInterface interface {
	ListSonosDevices(ctx context.Context) ([]string, error)
	CachedRoom(ctx context.Context, roomName string) (*sonos.ZonePlayer, error)
}

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
	sc.sonos.Search(ctx, func(s *sonos.Sonos, zp *sonos.ZonePlayer) {
		sc.cache.Store(zp.RoomName(), zp)
	})

	<-ctx.Done()

	var devices []string
	sc.cache.Range(func(key, value any) bool {
		if name, ok := key.(string); ok {
			devices = append(devices, name)
		}
		return true
	})

	return devices, nil
}
