package sonos

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultSubscriptionTimeout is the subscription lifetime requested when
	// SubscribeOptions.Timeout is zero.
	DefaultSubscriptionTimeout = time.Hour

	// renewFraction is how far into a subscription's granted timeout
	// auto-renewal renews it, e.g. 0.8 renews at 80% of the timeout.
	renewFraction = 0.8
)

// Retry bounds for auto-renewal when re-subscribing fails. Variables so
// tests can shorten them.
var (
	minResubscribeRetry = 5 * time.Second
	maxResubscribeRetry = 5 * time.Minute
)

var (
	// ErrClosed is returned when using a Sonos after Close, and is the
	// Err of subscriptions ended by Close.
	ErrClosed = errors.New("sonos: closed")

	// ErrUnsubscribed is the Err of a subscription ended by Unsubscribe.
	ErrUnsubscribed = errors.New("sonos: unsubscribed")

	// ErrSubscriptionLost is wrapped by the Err of a subscription that
	// could not be kept alive, e.g. because the device stopped answering.
	ErrSubscriptionLost = errors.New("sonos: subscription lost")
)

// errNotActive is returned when renewing a subscription that has ended.
var errNotActive = errors.New("sonos: subscription is not active")

// EventService is a UPnP service that can be subscribed to. Every generated
// service in services/* that sends events implements it.
type EventService interface {
	EventEndpoint() *url.URL
	// ParseEvent decodes an event notification body. It may return some
	// events together with an error for the properties it could not decode;
	// the events are delivered and the error is logged.
	ParseEvent(body []byte) ([]any, error)
}

// EventHandler receives the events of a subscription. LastChange and
// ZoneGroupState payloads arrive decoded (e.g. AVTransportLastChange);
// everything else arrives as its generated state variable type
// (e.g. grouprenderingcontrol.GroupVolume).
//
// Handlers run on the event listener's goroutines. The events of one
// subscription are handled one at a time, in the order they arrived; events
// of different subscriptions may be handled concurrently.
type EventHandler func(event any)

// SubscribeOptions configures Subscribe. A nil *SubscribeOptions uses the
// defaults.
type SubscribeOptions struct {
	// Timeout is the subscription lifetime to request; the device may grant
	// a different one (see Subscription.Timeout). It also bounds how long a
	// lost subscription can go unnoticed (see Subscription.Done).
	// Zero means DefaultSubscriptionTimeout.
	//
	// A device that reboots forgets its subscriptions without telling the
	// subscriber, so no events arrive until the next renewal fails (or,
	// without AutoRenew, until Timeout runs out). Renewal happens at 80% of
	// the granted lifetime, so with the default of an hour a reboot can go
	// unnoticed for about 48 minutes. Use a shorter Timeout, e.g. a few
	// minutes, when it matters that events keep flowing.
	Timeout time.Duration

	// AutoRenew keeps the subscription alive in the background until
	// Unsubscribe or Sonos.Close. If the device drops it (e.g. across a
	// reboot), it is re-created, retrying with backoff; the device then
	// sends its full state again as the first event.
	AutoRenew bool
}

// Subscription is an event subscription. Its methods are safe for
// concurrent use.
type Subscription struct {
	sonos     *Sonos
	svc       EventService
	handler   EventHandler
	requested time.Duration
	autoRenew bool
	raw       bool // pass events to handler as ParseEvent returns them

	mu      sync.Mutex
	token   string
	sid     string
	timeout time.Duration
	expires time.Time
	err     error
	done    chan struct{}

	finishOnce sync.Once

	// set once, before the monitor goroutine starts
	stopMonitor    context.CancelFunc
	monitorStopped chan struct{}

	// lastEvent is closed when the handlers of the most recently received
	// event have finished; see queueEvent.
	queueMu   sync.Mutex
	lastEvent chan struct{}
}

// queueEvent puts a just-received event at the end of the subscription's
// queue. It returns a channel that is closed when the event before it has
// been handled, and a func to call when this event has been handled.
// ServeHTTP calls it before acknowledging the event, so the events of one
// subscription are handled one at a time, in the order they arrived, even
// though each NOTIFY is served on its own goroutine.
func (sub *Subscription) queueEvent() (prev <-chan struct{}, done func()) {
	mine := make(chan struct{})
	sub.queueMu.Lock()
	prevCh := sub.lastEvent
	sub.lastEvent = mine
	sub.queueMu.Unlock()
	if prevCh == nil {
		prevCh = closedChan
	}
	return prevCh, func() { close(mine) }
}

var closedChan = func() chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}()

// Subscribe subscribes to the events of svc, e.g. zp.AVTransport, and calls
// handler for each of them until the subscription ends. ctx bounds the
// SUBSCRIBE request only.
func (s *Sonos) Subscribe(ctx context.Context, svc EventService, handler EventHandler, opts *SubscribeOptions) (*Subscription, error) {
	return s.subscribe(ctx, svc, handler, opts, false)
}

// subscribe is Subscribe; with raw, LastChange payloads reach handler
// undecoded.
func (s *Sonos) subscribe(ctx context.Context, svc EventService, handler EventHandler, opts *SubscribeOptions, raw bool) (*Subscription, error) {
	if svc == nil {
		return nil, errors.New("sonos: nil service")
	}
	if handler == nil {
		return nil, errors.New("sonos: nil handler")
	}
	if opts == nil {
		opts = &SubscribeOptions{}
	}

	sub := &Subscription{
		sonos:     s,
		svc:       svc,
		handler:   handler,
		requested: orDefault(opts.Timeout, DefaultSubscriptionTimeout),
		autoRenew: opts.AutoRenew,
		raw:       raw,
		done:      make(chan struct{}),
	}

	if err := sub.subscribe(ctx); err != nil {
		return nil, err
	}

	// Every subscription gets a monitor: it renews auto-renewed ones and
	// notices when any of them is lost.
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		sub.unsubscribe(ctx)
		return nil, ErrClosed
	}
	monitorCtx, cancel := context.WithCancel(s.ctx)
	sub.stopMonitor = cancel
	sub.monitorStopped = make(chan struct{})
	s.wg.Go(func() {
		defer close(sub.monitorStopped)
		sub.monitor(monitorCtx)
	})
	s.mu.Unlock()

	return sub, nil
}

func orDefault(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}

// SID returns the current subscription id. It changes if auto-renewal had
// to re-create the subscription, and is empty once the subscription ended.
func (sub *Subscription) SID() string {
	sub.mu.Lock()
	defer sub.mu.Unlock()
	return sub.sid
}

// Timeout returns the subscription lifetime granted by the device.
func (sub *Subscription) Timeout() time.Duration {
	sub.mu.Lock()
	defer sub.mu.Unlock()
	return sub.timeout
}

// Done returns a channel that is closed when the subscription ends: after
// Unsubscribe, after Sonos.Close, or when it is lost. A subscription is
// lost when it could not be renewed (or, with AutoRenew, re-created) before
// the lifetime granted by the device ran out, so a device that stops
// answering is noticed within Timeout at most.
func (sub *Subscription) Done() <-chan struct{} {
	return sub.done
}

// Err returns nil while the subscription is active. After Done is closed it
// returns why it ended: ErrUnsubscribed, ErrClosed, or an error wrapping
// ErrSubscriptionLost and the last renewal error.
func (sub *Subscription) Err() error {
	sub.mu.Lock()
	defer sub.mu.Unlock()
	return sub.err
}

func (sub *Subscription) expiry() time.Time {
	sub.mu.Lock()
	defer sub.mu.Unlock()
	return sub.expires
}

// finish ends the subscription with err, unless it has already ended.
func (sub *Subscription) finish(err error) {
	if err == nil {
		err = ErrUnsubscribed // Err must be non-nil once Done is closed
	}
	sub.finishOnce.Do(func() {
		sub.mu.Lock()
		sub.err = err
		sub.mu.Unlock()
		close(sub.done)
	})
}

// lose ends a subscription that could not be kept alive.
func (sub *Subscription) lose(cause error) {
	sub.mu.Lock()
	token, sid := sub.token, sub.sid
	sub.token, sub.sid = "", ""
	sub.mu.Unlock()

	if token != "" {
		sub.sonos.unregister(token)
	}
	sub.sonos.logger.Warn("sonos: subscription lost", "sid", sid, "err", cause)
	sub.finish(fmt.Errorf("%w: %w", ErrSubscriptionLost, cause))
}

// subscribe sends a new SUBSCRIBE and, on success, replaces the current
// subscription id and callback token.
func (sub *Subscription) subscribe(ctx context.Context) error {
	s := sub.sonos
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return ErrClosed
	}

	endpoint := sub.svc.EventEndpoint()
	ip, err := localIPFor(ctx, endpoint.Host)
	if err != nil {
		return err
	}

	// Register the handler under a fresh token before subscribing, so events
	// that arrive before the SUBSCRIBE response are not lost.
	token := rand.Text()
	s.register(token, sub)

	callback := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(ip.String(), strconv.Itoa(s.listener.Addr().(*net.TCPAddr).Port)),
		Path:   "/" + token,
	}
	req, err := http.NewRequestWithContext(ctx, "SUBSCRIBE", endpoint.String(), nil)
	if err != nil {
		s.unregister(token)
		return err
	}
	req.Header.Set("CALLBACK", "<"+callback.String()+">")
	req.Header.Set("NT", "upnp:event")
	req.Header.Set("TIMEOUT", timeoutHeader(sub.requested))

	// The lifetime starts no later than the request is sent.
	start := time.Now()
	res, err := s.client.Do(req)
	if err == nil {
		err = readStatus("SUBSCRIBE", endpoint, res)
	}
	sid := ""
	if err == nil {
		if sid = res.Header.Get("SID"); sid == "" {
			err = fmt.Errorf("sonos: SUBSCRIBE %s: response has no SID", endpoint)
		}
	}
	if err != nil {
		s.unregister(token)
		return err
	}

	granted, ok := parseTimeoutHeader(res.Header.Get("TIMEOUT"))
	if !ok {
		granted = sub.requested
	}

	sub.mu.Lock()
	oldToken := sub.token
	sub.token, sub.sid, sub.timeout, sub.expires = token, sid, granted, start.Add(granted)
	sub.mu.Unlock()
	if oldToken != "" {
		// Events for the replaced subscription are now rejected, which also
		// makes the device drop it.
		s.unregister(oldToken)
	}
	return nil
}

// Renew extends the subscription. Auto-renewed subscriptions don't need it.
func (sub *Subscription) Renew(ctx context.Context) error {
	sid := sub.SID()
	if sid == "" {
		return errNotActive
	}

	endpoint := sub.svc.EventEndpoint()
	req, err := http.NewRequestWithContext(ctx, "SUBSCRIBE", endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("SID", sid)
	req.Header.Set("TIMEOUT", timeoutHeader(sub.requested))

	start := time.Now()
	res, err := sub.sonos.client.Do(req)
	if err != nil {
		return err
	}
	if err := readStatus("SUBSCRIBE", endpoint, res); err != nil {
		return err
	}

	granted, ok := parseTimeoutHeader(res.Header.Get("TIMEOUT"))
	if !ok {
		granted = sub.requested
	}
	sub.mu.Lock()
	sub.timeout, sub.expires = granted, start.Add(granted)
	sub.mu.Unlock()
	return nil
}

// Unsubscribe cancels the subscription and ends it with ErrUnsubscribed.
// Its handler is removed even if the request to the device fails; any
// further events are rejected, which makes the device drop the
// subscription. Calling it again, or after the subscription was lost, is a
// no-op.
func (sub *Subscription) Unsubscribe(ctx context.Context) error {
	sub.stopMonitor()
	<-sub.monitorStopped
	err := sub.unsubscribe(ctx)
	sub.finish(ErrUnsubscribed)
	return err
}

func (sub *Subscription) unsubscribe(ctx context.Context) error {
	sub.mu.Lock()
	token, sid := sub.token, sub.sid
	sub.token, sub.sid = "", ""
	sub.mu.Unlock()

	if token != "" {
		sub.sonos.unregister(token)
	}
	if sid == "" {
		return nil
	}

	endpoint := sub.svc.EventEndpoint()
	req, err := http.NewRequestWithContext(ctx, "UNSUBSCRIBE", endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("SID", sid)

	res, err := sub.sonos.client.Do(req)
	if err != nil {
		return err
	}
	return readStatus("UNSUBSCRIBE", endpoint, res)
}

// monitor renews an auto-renewed subscription, re-creating it if renewal
// fails, and ends any subscription that expires without being renewed.
func (sub *Subscription) monitor(ctx context.Context) {
	logger := sub.sonos.logger
	renewIn := func() time.Duration {
		return time.Duration(float64(sub.Timeout()) * renewFraction)
	}

	first := time.Until(sub.expiry())
	if sub.autoRenew {
		first = renewIn()
	}
	timer := time.NewTimer(first)
	defer timer.Stop()
	retry := minResubscribeRetry

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		if !sub.autoRenew {
			// Renew may have extended the lifetime since the timer was set.
			if left := time.Until(sub.expiry()); left > 0 {
				timer.Reset(left)
				continue
			}
			sub.lose(errors.New("not renewed before it expired"))
			return
		}

		action := "renewed"
		err := sub.Renew(ctx)
		if err != nil && ctx.Err() == nil {
			logger.Warn("sonos: renewing subscription failed, re-subscribing", "sid", sub.SID(), "err", err)
			action = "re-subscribed"
			err = sub.subscribe(ctx)
		}
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			left := time.Until(sub.expiry())
			if left <= 0 {
				sub.lose(err)
				return
			}
			// Retry well before the next regular renewal, and once more
			// right at expiry before giving up.
			logger.Warn("sonos: re-subscribing failed", "retry_in", min(retry, left), "err", err)
			timer.Reset(min(retry, left))
			retry = min(retry*2, maxResubscribeRetry)
			continue
		}

		logger.Debug("sonos: subscription "+action, "sid", sub.SID(), "timeout", sub.Timeout(), "next_renewal_in", renewIn())
		retry = minResubscribeRetry
		timer.Reset(renewIn())
	}
}

// timeoutHeader formats a TIMEOUT header, rounding up to whole seconds:
// "Second-0" is not a valid lifetime.
func timeoutHeader(d time.Duration) string {
	sec := max(int64((d+time.Second-1)/time.Second), 1)
	return "Second-" + strconv.FormatInt(sec, 10)
}

// parseTimeoutHeader parses a UPnP TIMEOUT header such as "Second-1800".
func parseTimeoutHeader(header string) (time.Duration, bool) {
	header = strings.TrimSpace(header)
	if len(header) < len("Second-") || !strings.EqualFold(header[:len("Second-")], "Second-") {
		return 0, false
	}
	sec, err := strconv.ParseUint(strings.TrimSpace(header[len("Second-"):]), 10, 32)
	if err != nil || sec == 0 {
		return 0, false
	}
	return time.Duration(sec) * time.Second, true
}

// localIPFor returns the local address used to reach hostport, so the device
// is given a callback URL on the right interface. Dialing UDP sends nothing.
func localIPFor(ctx context.Context, hostport string) (net.IP, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", hostport)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP, nil
}

// readStatus reads and closes res.Body and returns an error describing any
// non-200 response.
func readStatus(method string, endpoint *url.URL, res *http.Response) error {
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("sonos: %s %s: %s", method, endpoint, res.Status)
		if b := strings.TrimSpace(string(body)); b != "" {
			msg += ": " + b
		}
		return errors.New(msg)
	}
	return nil
}
