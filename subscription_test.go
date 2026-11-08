package sonos

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	avt "github.com/caglar10ur/sonos/services/avtransport"
	cds "github.com/caglar10ur/sonos/services/contentdirectory"
	grc "github.com/caglar10ur/sonos/services/grouprenderingcontrol"
	ren "github.com/caglar10ur/sonos/services/renderingcontrol"
	zgt "github.com/caglar10ur/sonos/services/zonegrouptopology"
)

// eventService is an EventService whose ParseEvent returns the raw body as
// a single string event, so tests can see exactly what was delivered.
type eventService struct {
	endpoint *url.URL
}

func (e *eventService) EventEndpoint() *url.URL { return e.endpoint }
func (e *eventService) ParseEvent(b []byte) ([]any, error) {
	return []any{string(b)}, nil
}

// fakeDevice answers SUBSCRIBE, renewal and UNSUBSCRIBE requests and
// records them. onSubscribe, if set, runs before a new subscription is
// answered and can fail it by returning a status other than 200.
type fakeDevice struct {
	srv *httptest.Server

	timeout     string // TIMEOUT header to grant, e.g. "Second-1"
	renewStatus int    // status for renewals; 0 means 200
	onSubscribe func(n int32, callback string) int

	subscribes, renewals, unsubscribes atomic.Int32

	mu        sync.Mutex
	callbacks []string
}

func newFakeEventDevice(t *testing.T, configure func(*fakeDevice)) *fakeDevice {
	t.Helper()
	d := &fakeDevice{}
	if configure != nil {
		configure(d)
	}
	d.srv = httptest.NewServer(http.HandlerFunc(d.serve))
	t.Cleanup(d.srv.Close)
	return d
}

func (d *fakeDevice) serve(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "SUBSCRIBE":
		if r.Header.Get("SID") != "" {
			d.renewals.Add(1)
			if d.renewStatus != 0 {
				w.WriteHeader(d.renewStatus)
				return
			}
			w.Header().Set("TIMEOUT", d.timeout)
			return
		}
		n := d.subscribes.Add(1)
		callback := strings.Trim(r.Header.Get("CALLBACK"), "<>")
		d.mu.Lock()
		d.callbacks = append(d.callbacks, callback)
		d.mu.Unlock()
		if d.onSubscribe != nil {
			if status := d.onSubscribe(n, callback); status != http.StatusOK {
				w.WriteHeader(status)
				return
			}
		}
		w.Header().Set("SID", fmt.Sprintf("uuid:sid-%d", n))
		w.Header().Set("TIMEOUT", d.timeout)
	case "UNSUBSCRIBE":
		d.unsubscribes.Add(1)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (d *fakeDevice) service() *eventService {
	u, _ := url.Parse(d.srv.URL)
	return &eventService{endpoint: u}
}

func (d *fakeDevice) callback(i int) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.callbacks[i]
}

// sendNotify sends a NOTIFY to a callback URL.
func sendNotify(callback, body string) (int, error) {
	req, err := http.NewRequest("NOTIFY", callback, strings.NewReader(body))
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	res.Body.Close()
	return res.StatusCode, nil
}

func notify(t *testing.T, callback, body string) int {
	t.Helper()
	code, err := sendNotify(callback, body)
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func newTestSonos(t *testing.T) *Sonos {
	t.Helper()
	s, err := New()
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func ignore(any) {}

func TestParseTimeoutHeader(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		ok    bool
	}{
		{"Second-3600", time.Hour, true},
		{"second-1800", 30 * time.Minute, true},
		{"  Second-60  ", time.Minute, true},
		{"Second-infinite", 0, false},
		{"invalid", 0, false},
		{"", 0, false},
		{"Second-0", 0, false},
		{"Second--10", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseTimeoutHeader(tt.input)
		if ok != tt.ok || got != tt.want {
			t.Errorf("parseTimeoutHeader(%q) = (%v, %t), want (%v, %t)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSubscribeUsesGrantedTimeout(t *testing.T) {
	d := newFakeEventDevice(t, func(d *fakeDevice) { d.timeout = "Second-1800" })
	s := newTestSonos(t)

	sub, err := s.Subscribe(t.Context(), d.service(), ignore, &SubscribeOptions{Timeout: time.Hour})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if sub.SID() != "uuid:sid-1" {
		t.Errorf("SID() = %q", sub.SID())
	}
	if sub.Timeout() != 30*time.Minute {
		t.Errorf("Timeout() = %v, want 30m granted by the device", sub.Timeout())
	}
	if err := sub.Renew(t.Context()); err != nil {
		t.Errorf("Renew failed: %v", err)
	}
}

func TestSubscribeRejectsBadArguments(t *testing.T) {
	s := newTestSonos(t)
	if _, err := s.Subscribe(t.Context(), nil, ignore, nil); err == nil {
		t.Error("nil service: expected an error")
	}
	d := newFakeEventDevice(t, nil)
	if _, err := s.Subscribe(t.Context(), d.service(), nil, nil); err == nil {
		t.Error("nil handler: expected an error")
	}
}

// The device may send the initial event before it answers SUBSCRIBE. That
// event carries the full state and must reach the handler.
func TestFirstEventBeforeSubscribeResponse(t *testing.T) {
	var notifyStatus atomic.Int32
	d := newFakeEventDevice(t, func(d *fakeDevice) {
		d.onSubscribe = func(_ int32, callback string) int {
			if code, err := sendNotify(callback, "initial-state"); err == nil {
				notifyStatus.Store(int32(code))
			}
			return http.StatusOK
		}
	})
	s := newTestSonos(t)

	got := make(chan any, 1)
	if _, err := s.Subscribe(t.Context(), d.service(), func(evt any) { got <- evt }, nil); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if code := notifyStatus.Load(); code != http.StatusOK {
		t.Fatalf("early NOTIFY got status %d, want 200", code)
	}
	select {
	case evt := <-got:
		if evt != "initial-state" {
			t.Fatalf("got event %v, want initial-state", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("initial event was not delivered")
	}
}

func TestServeHTTPRejectsUnknownAndNonNotify(t *testing.T) {
	s := newTestSonos(t)
	srv := httptest.NewServer(s)
	defer srv.Close()

	if code := notify(t, srv.URL+"/no-such-token", "body"); code != http.StatusPreconditionFailed {
		t.Errorf("unknown subscription: got %d, want 412", code)
	}

	res, err := http.Get(srv.URL + "/anything")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d, want 405", res.StatusCode)
	}
}

// A slow handler must not delay the device's 200, or it may drop the
// subscription.
func TestServeHTTPAcksBeforeSlowHandler(t *testing.T) {
	d := newFakeEventDevice(t, nil)
	s := newTestSonos(t)
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	defer close(release)
	handler := func(any) {
		started <- struct{}{}
		<-release
	}
	if _, err := s.Subscribe(t.Context(), d.service(), handler, nil); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	code, err := sendNotify(d.callback(0), "evt")
	if err != nil {
		t.Fatalf("NOTIFY was not acknowledged while the handler ran: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("got %d, want 200", code)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("ack took %v; it should not wait for the handler", elapsed)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was never called")
	}
}

// The device sends the next event as soon as it gets the 200 for the
// previous one, so acking early must not let a subscription's handlers run
// concurrently or out of order.
func TestEventsOfOneSubscriptionAreHandledInOrder(t *testing.T) {
	d := newFakeEventDevice(t, nil)
	s := newTestSonos(t)

	var (
		mu      sync.Mutex
		got     []any
		running int
		overlap bool
	)
	firstStarted := make(chan struct{})
	release := make(chan struct{})
	handled := make(chan struct{}, 3)
	handler := func(evt any) {
		mu.Lock()
		running++
		overlap = overlap || running > 1
		got = append(got, evt)
		first := len(got) == 1
		mu.Unlock()
		if first {
			close(firstStarted)
			<-release
		}
		mu.Lock()
		running--
		mu.Unlock()
		handled <- struct{}{}
	}
	if _, err := s.Subscribe(t.Context(), d.service(), handler, nil); err != nil {
		t.Fatal(err)
	}

	// e1 blocks in its handler; e2 and e3 are acknowledged meanwhile.
	notify(t, d.callback(0), "e1")
	<-firstStarted
	notify(t, d.callback(0), "e2")
	notify(t, d.callback(0), "e3")
	time.Sleep(100 * time.Millisecond) // give e2/e3 a chance to overtake e1
	close(release)
	for range 3 {
		select {
		case <-handled:
		case <-time.After(2 * time.Second):
			t.Fatal("not every event was handled")
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if overlap {
		t.Error("handlers of one subscription ran concurrently")
	}
	if fmt.Sprint(got) != "[e1 e2 e3]" {
		t.Errorf("handled %v, want [e1 e2 e3]", got)
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	d := newFakeEventDevice(t, nil)
	s := newTestSonos(t)
	var calls atomic.Int32
	sub, err := s.Subscribe(t.Context(), d.service(), func(any) { calls.Add(1) }, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := sub.Unsubscribe(t.Context()); err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}
	if err := sub.Unsubscribe(t.Context()); err != nil {
		t.Fatalf("second Unsubscribe failed: %v", err)
	}
	if n := d.unsubscribes.Load(); n != 1 {
		t.Errorf("got %d UNSUBSCRIBE requests, want 1", n)
	}
	if code := notify(t, d.callback(0), "late"); code != http.StatusPreconditionFailed {
		t.Errorf("event after Unsubscribe: got %d, want 412", code)
	}
	if n := calls.Load(); n != 0 {
		t.Errorf("handler called %d times after Unsubscribe", n)
	}
	if err := sub.Renew(t.Context()); err == nil {
		t.Error("Renew after Unsubscribe: expected an error")
	}
}

func TestAutoRenewRenewsUntilUnsubscribe(t *testing.T) {
	d := newFakeEventDevice(t, func(d *fakeDevice) { d.timeout = "Second-1" })
	s := newTestSonos(t)

	sub, err := s.Subscribe(t.Context(), d.service(), ignore, &SubscribeOptions{AutoRenew: true})
	if err != nil {
		t.Fatal(err)
	}
	// Renewal runs at 80% of the granted 1s timeout.
	waitFor(t, "a renewal", func() bool { return d.renewals.Load() > 0 })

	if err := sub.Unsubscribe(t.Context()); err != nil {
		t.Fatal(err)
	}
	renewals := d.renewals.Load()
	time.Sleep(1200 * time.Millisecond)
	if n := d.renewals.Load(); n != renewals {
		t.Errorf("renewed %d more times after Unsubscribe", n-renewals)
	}
}

// After a failed renewal, a replacement subscription takes over and events
// for the old one are rejected.
func TestAutoRenewResubscribesAfterFailedRenewal(t *testing.T) {
	d := newFakeEventDevice(t, func(d *fakeDevice) {
		d.timeout = "Second-1"
		d.renewStatus = http.StatusPreconditionFailed
	})
	s := newTestSonos(t)
	got := make(chan any, 1)

	sub, err := s.Subscribe(t.Context(), d.service(), func(e any) { got <- e }, &SubscribeOptions{AutoRenew: true})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the replacement subscription", func() bool { return sub.SID() == "uuid:sid-2" })

	if code := notify(t, d.callback(0), "old"); code != http.StatusPreconditionFailed {
		t.Errorf("event for replaced subscription: got %d, want 412", code)
	}
	if code := notify(t, d.callback(1), "new"); code != http.StatusOK {
		t.Errorf("event for replacement subscription: got %d, want 200", code)
	}
	if evt := <-got; evt != "new" {
		t.Errorf("got %v, want the replacement's event", evt)
	}
}

func TestAutoRenewRetriesFailedResubscribe(t *testing.T) {
	old := minResubscribeRetry
	minResubscribeRetry = 50 * time.Millisecond
	t.Cleanup(func() { minResubscribeRetry = old })

	d := newFakeEventDevice(t, func(d *fakeDevice) {
		d.timeout = "Second-2"
		d.renewStatus = http.StatusPreconditionFailed
		// The first two re-subscribe attempts fail.
		d.onSubscribe = func(n int32, _ string) int {
			if n == 2 || n == 3 {
				return http.StatusServiceUnavailable
			}
			return http.StatusOK
		}
	})
	s := newTestSonos(t)

	sub, err := s.Subscribe(t.Context(), d.service(), ignore, &SubscribeOptions{AutoRenew: true})
	if err != nil {
		t.Fatal(err)
	}
	// Without retries, the next attempt would only come a full renewal
	// interval later.
	waitFor(t, "a successful retry", func() bool { return sub.SID() == "uuid:sid-4" })
}

func TestCloseUnsubscribesEverything(t *testing.T) {
	d := newFakeEventDevice(t, func(d *fakeDevice) { d.timeout = "Second-3600" })
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Subscribe(t.Context(), d.service(), ignore, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Subscribe(t.Context(), d.service(), ignore, &SubscribeOptions{AutoRenew: true}); err != nil {
		t.Fatal(err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
	if n := d.unsubscribes.Load(); n != 2 {
		t.Errorf("got %d UNSUBSCRIBE requests on Close, want 2", n)
	}
	if _, err := s.Subscribe(t.Context(), d.service(), ignore, nil); !errors.Is(err, ErrClosed) {
		t.Errorf("Subscribe after Close = %v, want ErrClosed", err)
	}
}

func TestDispatchDecodesLastChangeAndPassesOthersThrough(t *testing.T) {
	s := newTestSonos(t)

	var got any
	s.dispatch(avt.LastChange(`<Event><InstanceID val="0"><TransportState val="PLAYING"/></InstanceID></Event>`),
		func(e any) { got = e })
	lc, ok := got.(AVTransportLastChange)
	if !ok {
		t.Fatalf("got %T, want AVTransportLastChange", got)
	}
	if lc.InstanceID.TransportState.Value != "PLAYING" {
		t.Errorf("TransportState = %q, want PLAYING", lc.InstanceID.TransportState.Value)
	}
	if !strings.Contains(fmt.Sprint(lc), "PLAYING") {
		t.Errorf("String() of value missing state: %v", lc)
	}

	type custom string
	s.dispatch(custom("x"), func(e any) { got = e })
	if got != custom("x") {
		t.Errorf("unknown event type not passed through, got %v", got)
	}
}

func TestDispatchZoneGroupStateEvent(t *testing.T) {
	s := newTestSonos(t)

	var got any
	s.dispatch(zgt.ZoneGroupState(`<ZoneGroupState><ZoneGroups>
<ZoneGroup Coordinator="RINCON_A" ID="RINCON_A:1"><ZoneGroupMember UUID="RINCON_A" ZoneName="Kitchen" Icon="x-rincon-roomicon:kitchen"/></ZoneGroup>
</ZoneGroups><VanishedDevices><VanishedDevice UUID="RINCON_V" ZoneName="Garage"/></VanishedDevices></ZoneGroupState>`),
		func(e any) { got = e })
	st, ok := got.(ZoneGroupTopologyZoneGroupState)
	if !ok {
		t.Fatalf("got %T, want ZoneGroupTopologyZoneGroupState", got)
	}
	if len(st.ZoneGroups) != 1 || st.ZoneGroups[0].ZoneGroupMember[0].Icon != "x-rincon-roomicon:kitchen" {
		t.Errorf("zone groups not parsed: %+v", st.ZoneGroups)
	}
	if len(st.VanishedDevices) != 1 || st.VanishedDevices[0].ZoneName != "Garage" {
		t.Errorf("vanished devices not parsed: %+v", st.VanishedDevices)
	}
	if !strings.Contains(st.String(), "Garage") {
		t.Errorf("String() missing vanished device: %s", st)
	}
}

func TestGeneratedParseEvent(t *testing.T) {
	loc, _ := url.Parse("http://192.168.1.100:1400/xml/device_description.xml")
	svc, err := ren.NewService(loc)
	if err != nil {
		t.Fatal(err)
	}
	if got := svc.EventEndpoint().String(); got != "http://192.168.1.100:1400/MediaRenderer/RenderingControl/Event" {
		t.Errorf("EventEndpoint() = %s", got)
	}

	events, err := svc.ParseEvent([]byte(`<e:propertyset xmlns:e="urn:schemas-upnp-org:event-1-0">
<e:property><LastChange>&lt;Event/&gt;</LastChange></e:property></e:propertyset>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0] != ren.LastChange("<Event/>") {
		t.Errorf("events = %#v", events)
	}

	if _, err := svc.ParseEvent([]byte("<not xml")); err == nil {
		t.Error("malformed body: expected an error")
	}
}

func shortRetries(t *testing.T) {
	old := minResubscribeRetry
	minResubscribeRetry = 50 * time.Millisecond
	t.Cleanup(func() { minResubscribeRetry = old })
}

func isDone(sub *Subscription) bool {
	select {
	case <-sub.Done():
		return true
	default:
		return false
	}
}

func waitDone(t *testing.T, sub *Subscription, within time.Duration) {
	t.Helper()
	select {
	case <-sub.Done():
	case <-time.After(within):
		t.Fatalf("subscription not done after %v", within)
	}
}

func TestDoneAfterUnsubscribe(t *testing.T) {
	d := newFakeEventDevice(t, func(d *fakeDevice) { d.timeout = "Second-3600" })
	s := newTestSonos(t)
	sub, err := s.Subscribe(t.Context(), d.service(), ignore, &SubscribeOptions{AutoRenew: true})
	if err != nil {
		t.Fatal(err)
	}
	if isDone(sub) || sub.Err() != nil {
		t.Fatalf("new subscription: done=%v err=%v", isDone(sub), sub.Err())
	}

	if err := sub.Unsubscribe(t.Context()); err != nil {
		t.Fatal(err)
	}
	waitDone(t, sub, time.Second)
	if !errors.Is(sub.Err(), ErrUnsubscribed) {
		t.Errorf("Err() = %v, want ErrUnsubscribed", sub.Err())
	}
}

func TestDoneAfterClose(t *testing.T) {
	d := newFakeEventDevice(t, func(d *fakeDevice) { d.timeout = "Second-3600" })
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.Subscribe(t.Context(), d.service(), ignore, nil)
	if err != nil {
		t.Fatal(err)
	}

	s.Close()
	waitDone(t, sub, time.Second)
	if !errors.Is(sub.Err(), ErrClosed) {
		t.Errorf("Err() = %v, want ErrClosed", sub.Err())
	}
}

// When the device stops answering, an auto-renewed subscription keeps
// retrying until its lifetime runs out, then reports itself lost.
func TestAutoRenewedSubscriptionLostWhenDeviceGone(t *testing.T) {
	shortRetries(t)
	d := newFakeEventDevice(t, func(d *fakeDevice) { d.timeout = "Second-1" })
	s := newTestSonos(t)
	sub, err := s.Subscribe(t.Context(), d.service(), ignore, &SubscribeOptions{AutoRenew: true})
	if err != nil {
		t.Fatal(err)
	}

	d.srv.Close() // the device goes away
	waitDone(t, sub, 3*time.Second)

	if !errors.Is(sub.Err(), ErrSubscriptionLost) {
		t.Errorf("Err() = %v, want ErrSubscriptionLost", sub.Err())
	}
	if sub.SID() != "" {
		t.Errorf("SID() = %q after loss, want empty", sub.SID())
	}
	if code := notify(t, d.callback(0), "late"); code != http.StatusPreconditionFailed {
		t.Errorf("event after loss: got %d, want 412", code)
	}
	// Unsubscribe after loss is a no-op and keeps the reason.
	if err := sub.Unsubscribe(t.Context()); err != nil {
		t.Errorf("Unsubscribe after loss: %v", err)
	}
	if !errors.Is(sub.Err(), ErrSubscriptionLost) {
		t.Errorf("Err() after Unsubscribe = %v, want ErrSubscriptionLost", sub.Err())
	}
}

// A device that drops the subscription but stays reachable is not a loss:
// the subscription is re-created and stays active.
func TestAutoRenewedSubscriptionSurvivesDeviceReboot(t *testing.T) {
	d := newFakeEventDevice(t, func(d *fakeDevice) {
		d.timeout = "Second-1"
		d.renewStatus = http.StatusPreconditionFailed
	})
	s := newTestSonos(t)
	sub, err := s.Subscribe(t.Context(), d.service(), ignore, &SubscribeOptions{AutoRenew: true})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2500 * time.Millisecond) // two lifetimes
	if isDone(sub) {
		t.Fatalf("subscription ended: %v", sub.Err())
	}
}

func TestManualSubscriptionLostWhenNotRenewed(t *testing.T) {
	d := newFakeEventDevice(t, func(d *fakeDevice) { d.timeout = "Second-1" })
	s := newTestSonos(t)
	sub, err := s.Subscribe(t.Context(), d.service(), ignore, nil)
	if err != nil {
		t.Fatal(err)
	}

	waitDone(t, sub, 3*time.Second)
	if !errors.Is(sub.Err(), ErrSubscriptionLost) {
		t.Errorf("Err() = %v, want ErrSubscriptionLost", sub.Err())
	}
}

func TestManualRenewKeepsSubscriptionAlive(t *testing.T) {
	d := newFakeEventDevice(t, func(d *fakeDevice) { d.timeout = "Second-1" })
	s := newTestSonos(t)
	sub, err := s.Subscribe(t.Context(), d.service(), ignore, nil)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(600 * time.Millisecond)
	if err := sub.Renew(t.Context()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(700 * time.Millisecond) // past the original expiry
	if isDone(sub) {
		t.Fatalf("renewed subscription ended: %v", sub.Err())
	}
	waitDone(t, sub, 2*time.Second) // the renewed lifetime runs out
}

// finish must close Done exactly once, keep the first reason, and never
// leave Err nil after Done.
func TestFinishIsIdempotent(t *testing.T) {
	sub := &Subscription{done: make(chan struct{})}
	sub.finish(nil)
	sub.finish(ErrClosed)
	select {
	case <-sub.Done():
	default:
		t.Fatal("Done not closed")
	}
	if !errors.Is(sub.Err(), ErrUnsubscribed) {
		t.Errorf("Err() = %v, want ErrUnsubscribed", sub.Err())
	}
}

// A property that fails to decode must not drop the others.
func TestParseEventKeepsGoodProperties(t *testing.T) {
	loc, _ := url.Parse("http://192.168.1.100:1400/xml/device_description.xml")
	svc, err := grc.NewService(loc)
	if err != nil {
		t.Fatal(err)
	}
	events, err := svc.ParseEvent([]byte(`<e:propertyset xmlns:e="urn:schemas-upnp-org:event-1-0">
<e:property><GroupVolume>loud</GroupVolume></e:property>
<e:property><GroupMute>1</GroupMute></e:property>
</e:propertyset>`))
	if err == nil || !strings.Contains(err.Error(), "GroupVolume") {
		t.Errorf("err = %v, want an error naming GroupVolume", err)
	}
	if len(events) != 1 || events[0] != grc.GroupMute(true) {
		t.Errorf("events = %#v, want [GroupMute(true)]", events)
	}
}

// Values in the form a real Sonos (Era 300, 2026-11) sends them for
// ContentDirectory; the IDs are made up.
func TestParseContentDirectoryUpdateIDs(t *testing.T) {
	loc, _ := url.Parse("http://192.168.1.100:1400/xml/device_description.xml")
	svc, err := cds.NewService(loc)
	if err != nil {
		t.Fatal(err)
	}
	events, err := svc.ParseEvent([]byte(`<e:propertyset xmlns:e="urn:schemas-upnp-org:event-1-0">
<e:property><SystemUpdateID>8</SystemUpdateID></e:property>
<e:property><RadioFavoritesUpdateID>RINCON_000000000AAA01400,54</RadioFavoritesUpdateID></e:property>
<e:property><RadioLocationUpdateID>RINCON_000000000BBB01400,171</RadioLocationUpdateID></e:property>
</e:propertyset>`))
	if err != nil {
		t.Fatal(err)
	}
	want := []any{
		cds.SystemUpdateID(8),
		cds.RadioFavoritesUpdateID("RINCON_000000000AAA01400,54"),
		cds.RadioLocationUpdateID("RINCON_000000000BBB01400,171"),
	}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("events = %#v, want %#v", events, want)
	}
}
