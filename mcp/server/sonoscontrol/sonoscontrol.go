package sonoscontrol

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/caglar10ur/sonos"
)

type SonosControllerInterface interface {
	// Household returns a current snapshot of the household.
	Household(ctx context.Context) (*sonos.Household, error)
	// Room returns a room from a current snapshot of the household.
	Room(ctx context.Context, name string) (*sonos.Room, error)
}

// SonosController keeps the last household snapshot and refreshes it on
// every use. A refresh is a single request, so lookups are cheap and always
// reflect the current grouping; only a household that can't be refreshed
// is discovered again.
type SonosController struct {
	sonos *sonos.Sonos

	mu        sync.Mutex
	household *sonos.Household
}

func NewSonosController() (*SonosController, error) {
	s, err := sonos.New(sonos.WithLogger(slog.Default()))
	if err != nil {
		return nil, err
	}
	return &SonosController{sonos: s}, nil
}

func (sc *SonosController) Household(ctx context.Context) (*sonos.Household, error) {
	sc.mu.Lock()
	last := sc.household
	sc.mu.Unlock()

	var (
		h   *sonos.Household
		err error
	)
	if last != nil {
		// Leave half of the time for discovery, should no known player answer.
		refreshCtx, cancel := ctx, context.CancelFunc(func() {})
		if deadline, ok := ctx.Deadline(); ok {
			refreshCtx, cancel = context.WithTimeout(ctx, time.Until(deadline)/2)
		}
		h, err = last.Refresh(refreshCtx)
		cancel()
	}
	if last == nil || err != nil {
		// Nothing to refresh, or none of its players answered: discover.
		if h, err = sc.sonos.Household(ctx); err != nil {
			return nil, err
		}
	}

	sc.mu.Lock()
	sc.household = h
	sc.mu.Unlock()
	return h, nil
}

func (sc *SonosController) Room(ctx context.Context, name string) (*sonos.Room, error) {
	h, err := sc.Household(ctx)
	if err != nil {
		return nil, err
	}
	r := h.Room(name)
	if r == nil {
		return nil, fmt.Errorf("%w: %q", sonos.ErrRoomNotFound, name)
	}
	return r, nil
}
