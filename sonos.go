package sonos

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// maxEventSize caps the body of an incoming NOTIFY. ZoneGroupState events
	// for large households are the biggest and stay well below this.
	maxEventSize = 4 << 20

	// closeTimeout bounds the best-effort unsubscribe calls made by Close.
	closeTimeout = 5 * time.Second

	// maxParallelSetups bounds how many players a search sets up at once.
	maxParallelSetups = 8
)

// searchResends are when the M-SEARCH is sent again after the first time:
// SSDP runs over UDP, which may drop it or the answers.
var searchResends = []time.Duration{250 * time.Millisecond, time.Second}

var searchAddrs = []string{"239.255.255.250:1900", "255.255.255.255:1900"}

// https://svrooij.io/sonos-api-docs/sonos-communication.html#auto-discovery
// MX should be set to use timeout value in integer seconds
var searchPacket = []byte("M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 1\r\nST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n")

// Sonos discovers players on the local network and receives the events of
// subscriptions made through it. Call Close when done.
type Sonos struct {
	listener net.Listener
	server   *http.Server
	client   *http.Client
	logger   *slog.Logger

	// ctx is cancelled by Close; searches and auto-renewal stop with it.
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once
	closeErr  error

	mu     sync.Mutex
	closed bool // set by Close; guards wg.Add against wg.Wait
	// subs maps the callback token embedded in each subscription's CALLBACK
	// URL to the subscription. Tokens are registered before SUBSCRIBE is
	// sent, so even the first event (which can beat the response) is routed.
	subs map[string]*Subscription
}

type options struct {
	listenAddr string
	client     *http.Client
	logger     *slog.Logger
}

// Option configures a Sonos.
type Option func(*options)

// WithListenAddr sets the address the event listener binds to.
// The default is ":0": every interface, on a random port.
func WithListenAddr(addr string) Option {
	return func(o *options) { o.listenAddr = addr }
}

// WithHTTPClient sets the HTTP client used for subscriptions and for the
// players found by Search. The default has a 15 second timeout.
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) { o.client = c }
}

// WithLogger sets where the library logs problems it can't return as
// errors, such as a failed auto-renewal. The default discards them.
func WithLogger(l *slog.Logger) Option {
	return func(o *options) { o.logger = l }
}

// New starts the event listener and returns a Sonos.
func New(opts ...Option) (*Sonos, error) {
	o := options{
		listenAddr: ":0",
		client:     &http.Client{Timeout: 15 * time.Second},
		logger:     slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		opt(&o)
	}

	listener, err := net.Listen("tcp", o.listenAddr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Sonos{
		listener: listener,
		client:   o.client,
		logger:   o.logger,
		ctx:      ctx,
		cancel:   cancel,
		subs:     make(map[string]*Subscription),
	}
	s.server = &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(o.logger.Handler(), slog.LevelWarn),
	}

	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("sonos: event listener stopped", "err", err)
		}
	}()

	return s, nil
}

// Close stops searches and auto-renewal, makes a best-effort attempt to
// unsubscribe every active subscription (ending it with ErrClosed), and
// stops the event listener.
// It is safe to call more than once.
func (s *Sonos) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		s.cancel()
		s.wg.Wait()

		s.mu.Lock()
		subs := make([]*Subscription, 0, len(s.subs))
		for _, sub := range s.subs {
			subs = append(subs, sub)
		}
		s.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		var wg sync.WaitGroup
		for _, sub := range subs {
			wg.Go(func() {
				sid := sub.SID()
				if err := sub.unsubscribe(ctx); err != nil {
					s.logger.Warn("sonos: unsubscribe on close failed", "sid", sid, "err", err)
				}
				sub.finish(ErrClosed)
			})
		}
		wg.Wait()
		cancel()

		s.closeErr = s.server.Close()
	})
	return s.closeErr
}

// Search sends an SSDP discovery request and calls fn once for every group
// coordinator that answers, until ctx is done. Each call uses its own
// socket, so concurrent or repeated searches don't interfere. fn is called
// from one goroutine at a time, and not after Close returns; it must not
// call Close itself.
func (s *Sonos) Search(ctx context.Context, fn func(*ZonePlayer)) error {
	return s.search(ctx, true, fn)
}

// search is Search, optionally reporting every player instead of only the
// group coordinators.
func (s *Sonos) search(ctx context.Context, coordinatorsOnly bool, fn func(*ZonePlayer)) error {
	s.mu.Lock()
	defer s.mu.Unlock() // held until the goroutines are added to s.wg
	if s.closed {
		return ErrClosed
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero})
	if err != nil {
		return err
	}
	if err := sendSearch(conn); err != nil {
		conn.Close()
		return err
	}

	// Stop when the caller's ctx is done or the Sonos is closed; closing the
	// socket unblocks the pending read.
	ctx, cancel := context.WithCancel(ctx)
	stopOnClose := context.AfterFunc(s.ctx, cancel)
	stopConn := context.AfterFunc(ctx, func() { conn.Close() })

	// Close waits for these, so fn isn't called after Close returns.
	s.wg.Go(func() {
		for _, at := range searchResends {
			select {
			case <-ctx.Done():
				return
			case <-time.After(at):
			}
			_ = sendSearch(conn) // the first send succeeded; this is a bonus
		}
	})
	s.wg.Go(func() {
		defer stopConn()
		defer stopOnClose()
		defer cancel()
		defer conn.Close()
		s.readSearchResponses(ctx, conn, coordinatorsOnly, fn)
	})

	return nil
}

func sendSearch(conn *net.UDPConn) error {
	for _, addr := range searchAddrs {
		dst, err := net.ResolveUDPAddr("udp4", addr)
		if err != nil {
			return err
		}
		if _, err := conn.WriteTo(searchPacket, dst); err != nil {
			return err
		}
	}
	return nil
}

func (s *Sonos) readSearchResponses(ctx context.Context, conn *net.UDPConn, coordinatorsOnly bool, fn func(*ZonePlayer)) {
	// Players answer every M-SEARCH, on both addresses, so skip locations
	// this search has already handled before fetching anything.
	seen := make(map[string]bool)
	buf := make([]byte, 64<<10)

	// Setting a player up takes a request or two, so do it in parallel and
	// only serialize the calls to fn.
	var (
		wg   sync.WaitGroup
		fnMu sync.Mutex
		sem  = make(chan struct{}, maxParallelSetups)

		// The topology names every coordinator, so one fetch per search
		// answers for every player that replies.
		topoMu       sync.Mutex
		coordinators map[string]bool
	)
	defer wg.Wait()
	isCoordinator := func(zp *ZonePlayer) (bool, error) {
		topoMu.Lock()
		defer topoMu.Unlock()
		if coordinators == nil {
			state, err := zp.ZoneGroupState(ctx)
			if err != nil {
				return false, err
			}
			coordinators = make(map[string]bool)
			for _, zg := range state.ZoneGroups {
				coordinators[zg.Coordinator] = true
			}
		}
		return coordinators[zp.UUID()], nil
	}

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			// e.g. "no route to host" when the OS blocks local network access
			s.logger.Debug("sonos: reading search response", "err", err)
			continue
		}

		response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(buf[:n])), nil)
		if err != nil {
			continue
		}
		response.Body.Close()

		loc := response.Header.Get("Location")
		if loc == "" || seen[loc] {
			continue
		}
		seen[loc] = true

		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			zp, err := playerAt(ctx, loc, s.client)
			if err != nil {
				s.logger.Debug("sonos: skipping device", "location", loc, "err", err)
				return
			}
			if coordinatorsOnly {
				if ok, err := isCoordinator(zp); err != nil || !ok {
					return
				}
			}
			fnMu.Lock()
			defer fnMu.Unlock()
			if ctx.Err() == nil {
				fn(zp)
			}
		})
	}
}

// firstPlayer returns the first player that answers a search.
func (s *Sonos) firstPlayer(ctx context.Context) (*ZonePlayer, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	found := make(chan *ZonePlayer, 1) // buffered: fn may run once more after we return
	err := s.search(ctx, false, func(zp *ZonePlayer) {
		select {
		case found <- zp:
		default:
		}
	})
	if err != nil {
		return nil, err
	}
	select {
	case zp := <-found:
		return zp, nil
	case <-s.ctx.Done():
		return nil, ErrClosed
	case <-ctx.Done():
		return nil, fmt.Errorf("no player answered: %w", ctx.Err())
	}
}

func (s *Sonos) register(token string, sub *Subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[token] = sub
}

func (s *Sonos) unregister(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subs, token)
}

// ServeHTTP receives event notifications (NOTIFY) from devices and passes
// them to the matching subscription's handler.
func (s *Sonos) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "NOTIFY" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	s.mu.Lock()
	sub := s.subs[token]
	s.mu.Unlock()
	if sub == nil {
		// Per UPnP, 412 tells the device this subscription is unknown and
		// makes it stop sending events for it.
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxEventSize))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Take this event's place in the subscription's queue before
	// acknowledging it: the device only sends the next event after the
	// acknowledgement, so the queue keeps them in order.
	prev, done := sub.queueEvent()
	defer done()

	// Acknowledge before running handlers so a slow handler doesn't make the
	// device time out. Without the flush, net/http would hold the status
	// line until this function returns. Connection: close keeps the device
	// from sending its next event on this connection, which isn't read
	// again until the handlers return.
	w.Header().Set("Content-Length", "0")
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// ParseEvent returns what it could decode alongside err, so deliver that.
	events, err := sub.svc.ParseEvent(data)
	if err != nil {
		s.logger.Warn("sonos: dropping malformed event data", "sid", r.Header.Get("SID"), "err", err)
	}
	<-prev
	for _, evt := range events {
		if sub.raw {
			sub.handler(evt)
		} else {
			s.dispatch(evt, sub.handler)
		}
	}
}
