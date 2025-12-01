package sonos

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// MockRoundTripper implements http.RoundTripper to intercept requests
type MockRoundTripper struct {
	Handlers map[string]func(*http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// 1. Check for SOAPAction header
	soapAction := req.Header.Get("SOAPAction")
	// Clean quotes if present
	soapAction = strings.Trim(soapAction, "\"")

	if handler, ok := m.Handlers[soapAction]; ok {
		return handler(req)
	}

	// 2. Check for Device Description request (used in NewZonePlayer)
	if strings.Contains(req.URL.Path, "device_description.xml") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(mockDeviceDescription)),
			Header:     make(http.Header),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewBufferString("Not found")),
		Header:     make(http.Header),
	}, nil
}

// Helper to create a mocked ZonePlayer
func NewMockZonePlayer(t *testing.T, handlers map[string]func(*http.Request) (*http.Response, error)) *ZonePlayer {
	mockClient := &http.Client{
		Transport: &MockRoundTripper{Handlers: handlers},
	}

	// The location doesn't matter much as we intercept requests, but it needs to be valid
	loc, _ := url.Parse("http://192.168.1.100:1400/xml/device_description.xml")

	zp, err := NewZonePlayer(WithClient(mockClient), WithLocation(loc))
	if err != nil {
		t.Fatalf("Failed to create mock ZonePlayer: %v", err)
	}
	return zp
}

const mockDeviceDescription = `<?xml version="1.0" encoding="utf-8" ?>
<root xmlns="urn:schemas-upnp-org:device-1-0">
    <specVersion>
        <major>1</major>
        <minor>0</minor>
    </specVersion>
    <device>
        <deviceType>urn:schemas-upnp-org:device:ZonePlayer:1</deviceType>
        <friendlyName>Kitchen</friendlyName>
        <manufacturer>Sonos, Inc.</manufacturer>
        <modelName>Sonos One</modelName>
        <modelDescription>Sonos One Audio Player</modelDescription>
        <UDN>uuid:RINCON_000E58CDCA4001400</UDN>
        <serialNum>00-11-22-33-44-55:0</serialNum>
        <softwareVersion>57.2-54070</softwareVersion>
        <hardwareVersion>1.20.1.6-1.2</hardwareVersion>
        <roomName>Kitchen</roomName>
        <serviceList>
            <service>
                <serviceType>urn:schemas-upnp-org:service:RenderingControl:1</serviceType>
                <serviceId>urn:upnp-org:serviceId:RenderingControl</serviceId>
                <controlURL>/MediaRenderer/RenderingControl/Control</controlURL>
                <eventSubURL>/MediaRenderer/RenderingControl/Event</eventSubURL>
                <SCPDURL>/xml/RenderingControl1.xml</SCPDURL>
            </service>
			<service>
                <serviceType>urn:schemas-upnp-org:service:AVTransport:1</serviceType>
                <serviceId>urn:upnp-org:serviceId:AVTransport</serviceId>
                <controlURL>/MediaRenderer/AVTransport/Control</controlURL>
                <eventSubURL>/MediaRenderer/AVTransport/Event</eventSubURL>
                <SCPDURL>/xml/AVTransport1.xml</SCPDURL>
            </service>
        </serviceList>
    </device>
</root>`
