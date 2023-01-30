package sonos

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type FoundZonePlayerFunc func(*Sonos, *ZonePlayer)
type EventHandlerFunc func(interface{})

type Sonos struct {
	udpListener *net.UDPConn
	tcpListener net.Listener

	// map of coordinators
	zonePlayers sync.Map
	// map of subscription ids to event handler function
	subscriptions sync.Map
}

type SubscriptionOptions struct {
	ZonePlayer *ZonePlayer
	Service    SonosService
	Timeout    uint64

	EventHandler EventHandlerFunc

	Sid string
}

func (o *SubscriptionOptions) Validate() error {
	if o.ZonePlayer == nil {
		return fmt.Errorf("missing ZonePlayer")
	}
	if o.Service == nil {
		return fmt.Errorf("missing Service")
	}

	if o.Service.EventEndpoint().Path == "/QPlay/Event" {
		return fmt.Errorf("not supported Service")
	}
	if o.Timeout == 0 {
		o.Timeout = 86400
	}
	return nil
}

func (o *SubscriptionOptions) SetSid(sid string) {
	o.Sid = sid
}

func NewSonos() (*Sonos, error) {
	// Create listener for M-SEARCH
	udpListener, err := net.ListenUDP("udp", &net.UDPAddr{IP: []byte{0, 0, 0, 0}, Port: 0, Zone: ""})
	if err != nil {
		return nil, err
	}

	// create listener for events
	tcpListener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	s := &Sonos{
		udpListener: udpListener,
		tcpListener: tcpListener,
	}

	go func() {
		http.Serve(s.tcpListener, s)
	}()

	return s, nil
}

func (s *Sonos) Close() {
	s.udpListener.Close()
	s.tcpListener.Close()
}

func (s *Sonos) Search(ctx context.Context, fn FoundZonePlayerFunc) error {
	go func(ctx context.Context) {
		for {
			if ctx.Err() != nil {
				break
			}
			response, err := http.ReadResponse(bufio.NewReader(s.udpListener), nil)
			if err != nil {
				continue
			}

			location, err := url.Parse(response.Header.Get("Location"))
			if err != nil {
				continue
			}
			zp, err := NewZonePlayer(WithLocation(location))
			if err != nil {
				continue
			}
			if zp.IsCoordinator() {
				zp, loaded := s.zonePlayers.LoadOrStore(zp.SerialNumber(), zp)
				if !loaded {
					fn(s, zp.(*ZonePlayer))
				}
			}
		}
	}(ctx)

	// https://svrooij.io/sonos-api-docs/sonos-communication.html#auto-discovery
	// MX should be set to use timeout value in integer seconds
	pkt := []byte("M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 1\r\nST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n")
	for _, bcastaddr := range []string{"239.255.255.250:1900", "255.255.255.255:1900"} {
		bcast, err := net.ResolveUDPAddr("udp", bcastaddr)
		if err != nil {
			return err
		}
		_, err = s.udpListener.WriteTo(pkt, bcast)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Sonos) Register(zp *ZonePlayer) error {
	if zp.IsCoordinator() {
		_, loaded := s.zonePlayers.LoadOrStore(zp.SerialNumber(), zp)
		if loaded {
			return fmt.Errorf("ZonePlayer already registered")
		}
		return nil
	}
	return fmt.Errorf("ZonePlayer is not coordinator")
}

func (s *Sonos) FindRoom(ctx context.Context, room string) (*ZonePlayer, error) {
	c := make(chan *ZonePlayer)
	defer close(c)

	s.Search(ctx, func(s *Sonos, zp *ZonePlayer) {
		if zp.RoomName() == room {
			c <- zp
		}
	})

	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("timeout")
		case zp := <-c:
			return zp, nil
		}
	}
}

func (s *Sonos) Subscribe(ctx context.Context, opts *SubscriptionOptions) (string, error) {
	conn, err := net.Dial("tcp", opts.Service.EventEndpoint().Host)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	host := fmt.Sprintf("%s:%d", conn.LocalAddr().(*net.TCPAddr).IP.String(), s.tcpListener.Addr().(*net.TCPAddr).Port)

	calbackUrl := url.URL{
		Scheme:   "http",
		Host:     host,
		RawQuery: "sn=" + opts.ZonePlayer.SerialNumber(),
		Path:     opts.Service.EventEndpoint().Path,
	}

	req, err := http.NewRequestWithContext(ctx, "SUBSCRIBE", opts.Service.EventEndpoint().String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("HOST", opts.Service.EventEndpoint().Host)
	req.Header.Add("CALLBACK", fmt.Sprintf("<%s>", calbackUrl.String()))
	req.Header.Add("NT", "upnp:event")
	req.Header.Add("TIMEOUT", fmt.Sprintf("Second-%d", opts.Timeout))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.New(string(body))
	}
	sid := res.Header.Get("sid")

	// Add the sid to the subscriptions
	s.subscriptions.LoadOrStore(sid, opts.EventHandler)

	return sid, nil
}

func (s *Sonos) Renew(ctx context.Context, opts *SubscriptionOptions) error {
	req, err := http.NewRequestWithContext(ctx, "SUBSCRIBE", opts.Service.EventEndpoint().String(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("HOST", opts.Service.EventEndpoint().Host)
	req.Header.Add("SID", opts.Sid)
	req.Header.Add("TIMEOUT", fmt.Sprintf("Second-%d", opts.Timeout))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	return nil
}

func (s *Sonos) Unsubscribe(ctx context.Context, opts *SubscriptionOptions) error {
	req, err := http.NewRequestWithContext(ctx, "UNSUBSCRIBE", opts.Service.EventEndpoint().String(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("HOST", opts.Service.EventEndpoint().Host)
	req.Header.Add("SID", opts.Sid)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	return nil
}

func (s *Sonos) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	query := request.URL.Query()
	sn, ok := query["sn"]
	if !ok {
		response.WriteHeader(http.StatusNotFound)
		return
	}

	p, ok := s.zonePlayers.Load(sn[0])
	if !ok {
		response.WriteHeader(http.StatusNotFound)
		return
	}
	zp := p.(*ZonePlayer)

	data, err := ioutil.ReadAll(request.Body)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var events []interface{}
	switch request.URL.Path {
	case zp.AlarmClock.EventEndpoint().Path:
		events = zp.AlarmClock.ParseEvent(data)
	case zp.AudioIn.EventEndpoint().Path:
		events = zp.AudioIn.ParseEvent(data)
	case zp.AVTransport.EventEndpoint().Path:
		events = zp.AVTransport.ParseEvent(data)
	case zp.ConnectionManager.EventEndpoint().Path:
		events = zp.ConnectionManager.ParseEvent(data)
	case zp.ContentDirectory.EventEndpoint().Path:
		events = zp.ContentDirectory.ParseEvent(data)
	case zp.DeviceProperties.EventEndpoint().Path:
		events = zp.DeviceProperties.ParseEvent(data)
	case zp.GroupManagement.EventEndpoint().Path:
		events = zp.GroupManagement.ParseEvent(data)
	case zp.GroupRenderingControl.EventEndpoint().Path:
		events = zp.GroupRenderingControl.ParseEvent(data)
	case zp.MusicServices.EventEndpoint().Path:
		events = zp.MusicServices.ParseEvent(data)
	case zp.QPlay.EventEndpoint().Path:
		events = zp.QPlay.ParseEvent(data)
	case zp.Queue.EventEndpoint().Path:
		events = zp.Queue.ParseEvent(data)
	case zp.RenderingControl.EventEndpoint().Path:
		events = zp.RenderingControl.ParseEvent(data)
	case zp.SystemProperties.EventEndpoint().Path:
		events = zp.SystemProperties.ParseEvent(data)
	case zp.VirtualLineIn.EventEndpoint().Path:
		events = zp.VirtualLineIn.ParseEvent(data)
	case zp.ZoneGroupTopology.EventEndpoint().Path:
		events = zp.ZoneGroupTopology.ParseEvent(data)
	}

	sid := request.Header.Get("sid")
	seq := request.Header.Get("seq")

	// Response to the Subscription request comes before Subscription calls reads the sid
	// give it a second and try again for the first update, if this won't work well enough a channel to syncronize could work.
	fn, ok := s.subscriptions.Load(sid)
	if !ok && seq == "0" {
		time.Sleep(1 * time.Second)

		fn, ok = s.subscriptions.Load(sid)
		if !ok {
			fn = func(evt interface{}) {}
		}
	}

	for _, evt := range events {
		zp.Event(evt, fn.(EventHandlerFunc))
	}
	response.WriteHeader(http.StatusOK)
}
