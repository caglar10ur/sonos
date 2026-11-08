package soap

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const fault = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><s:Fault>
<faultcode>s:Client</faultcode><faultstring>UPnPError</faultstring>
<detail><UPnPError xmlns="urn:schemas-upnp-org:control-1-0"><errorCode>701</errorCode><errorDescription>Transition not available</errorDescription></UPnPError></detail>
</s:Fault></s:Body></s:Envelope>`

type playArgs struct {
	InstanceID uint32 `xml:"InstanceID"`
	Speed      string `xml:"Speed"`
}

type volumeResponse struct {
	CurrentVolume uint16 `xml:"CurrentVolume"`
}

// serve answers every request with status and body, and records the last
// request body.
func serve(t *testing.T, status int, body string) (*url.URL, *http.Client, *string) {
	t.Helper()
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := r.Header.Get("SOAPAction"); !strings.HasPrefix(a, "urn:test#") {
			t.Errorf("SOAPAction = %q", a)
		}
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return u, srv.Client(), &got
}

func TestCallRequestAndResponse(t *testing.T) {
	u, c, got := serve(t, http.StatusOK, `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body>
<u:GetVolumeResponse xmlns:u="urn:test"><CurrentVolume>26</CurrentVolume></u:GetVolumeResponse></s:Body></s:Envelope>`)

	resp, err := Call[volumeResponse](context.Background(), c, u, "urn:test", "GetVolume", &playArgs{Speed: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.CurrentVolume != 26 {
		t.Errorf("CurrentVolume = %d, want 26", resp.CurrentVolume)
	}
	want := `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<s:Body><u:GetVolume xmlns:u="urn:test"><InstanceID>0</InstanceID><Speed>1</Speed></u:GetVolume></s:Body></s:Envelope>`
	if *got != want {
		t.Errorf("request body\n got %s\nwant %s", *got, want)
	}
}

// A nil *Args must still send every argument, with its zero value: devices
// reject calls that omit e.g. InstanceID.
func TestCallWithNilArgs(t *testing.T) {
	ok := `<s:Envelope><s:Body><u:StopResponse xmlns:u="urn:test"/></s:Body></s:Envelope>`
	for _, tc := range []struct {
		args any
		want string
	}{
		{(*playArgs)(nil), `<u:Stop xmlns:u="urn:test"><InstanceID>0</InstanceID><Speed></Speed></u:Stop>`},
		{nil, `<u:Stop xmlns:u="urn:test"></u:Stop>`},
	} {
		u, c, got := serve(t, http.StatusOK, ok)
		if _, err := Call[struct{}](context.Background(), c, u, "urn:test", "Stop", tc.args); err != nil {
			t.Fatalf("args %#v: %v", tc.args, err)
		}
		if !strings.Contains(*got, tc.want) {
			t.Errorf("args %#v: request %s lacks %s", tc.args, *got, tc.want)
		}
	}
}

func TestCallRejectsResponseWithoutActionElement(t *testing.T) {
	u, c, _ := serve(t, http.StatusOK, `<s:Envelope><s:Body><u:OtherResponse/></s:Body></s:Envelope>`)
	_, err := Call[volumeResponse](context.Background(), c, u, "urn:test", "GetVolume", nil)
	if err == nil || !strings.Contains(err.Error(), "GetVolumeResponse") {
		t.Errorf("err = %v, want one naming GetVolumeResponse", err)
	}
}

func TestCallReturnsUPnPError(t *testing.T) {
	u, c, _ := serve(t, http.StatusInternalServerError, fault)
	_, err := Call[struct{}](context.Background(), c, u, "urn:test", "Play", nil)

	var upnpErr *UPnPError
	if !errors.As(err, &upnpErr) {
		t.Fatalf("got %T (%v), want *UPnPError", err, err)
	}
	if upnpErr.Code != 701 || upnpErr.Action != "Play" || upnpErr.Description != "Transition not available" {
		t.Errorf("got %+v", upnpErr)
	}
	if !IsCode(err, TransitionNotAvailable) || IsCode(err, NoContents) || IsCode(errors.New("x"), TransitionNotAvailable) {
		t.Error("IsCode gave the wrong answer")
	}
	if want := "UPnP error 701 calling Play: Transition not available"; err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestCallReportsPlainHTTPErrors(t *testing.T) {
	u, c, _ := serve(t, http.StatusNotFound, "not here")
	_, err := Call[struct{}](context.Background(), c, u, "urn:test", "Play", nil)
	var upnpErr *UPnPError
	if err == nil || errors.As(err, &upnpErr) {
		t.Fatalf("got %v, want a non-UPnP error", err)
	}
	if !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "not here") {
		t.Errorf("error lacks status or body: %v", err)
	}
}
