// Code generated by makeservice. DO NOT EDIT.

// Package grouprenderingcontrol is a generated GroupRenderingControl package.
package grouprenderingcontrol

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	_ServiceURN     = "urn:schemas-upnp-org:service:GroupRenderingControl:1"
	_EncodingSchema = "http://schemas.xmlsoap.org/soap/encoding/"
	_EnvelopeSchema = "http://schemas.xmlsoap.org/soap/envelope/"
)

type ServiceOption func(*Service)

func WithClient(c *http.Client) ServiceOption {
	return func(s *Service) {
		s.client = c
	}
}

func WithLocation(u *url.URL) ServiceOption {
	return func(s *Service) {
		s.location = u
	}
}

type Service struct {
	controlEndpoint *url.URL
	eventEndpoint   *url.URL

	location *url.URL
	client   *http.Client
}

func NewService(opts ...ServiceOption) *Service {
	s := &Service{}

	c, err := url.Parse("/MediaRenderer/GroupRenderingControl/Control")
	if nil != err {
		panic(err)
	}
	e, err := url.Parse("/MediaRenderer/GroupRenderingControl/Event")
	if nil != err {
		panic(err)
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.client == nil {
		panic("no client location")
	}
	if s.location == nil {
		panic("empty location")
	}

	s.controlEndpoint = s.location.ResolveReference(c)
	s.eventEndpoint = s.location.ResolveReference(e)

	return s
}

func (s *Service) ControlEndpoint() *url.URL {
	return s.controlEndpoint
}

func (s *Service) EventEndpoint() *url.URL {
	return s.eventEndpoint
}

func (s *Service) Location() *url.URL {
	return s.location
}

func (s *Service) Client() *http.Client {
	return s.client
}

type Envelope struct {
	XMLName       xml.Name `xml:"s:Envelope"`
	Xmlns         string   `xml:"xmlns:s,attr"`
	EncodingStyle string   `xml:"s:encodingStyle,attr"`
	Body          Body     `xml:"s:Body"`
}
type Body struct {
	XMLName                xml.Name                    `xml:"s:Body"`
	GetGroupMute           *GetGroupMuteArgs           `xml:"u:GetGroupMute,omitempty"`
	SetGroupMute           *SetGroupMuteArgs           `xml:"u:SetGroupMute,omitempty"`
	GetGroupVolume         *GetGroupVolumeArgs         `xml:"u:GetGroupVolume,omitempty"`
	SetGroupVolume         *SetGroupVolumeArgs         `xml:"u:SetGroupVolume,omitempty"`
	SetRelativeGroupVolume *SetRelativeGroupVolumeArgs `xml:"u:SetRelativeGroupVolume,omitempty"`
	SnapshotGroupVolume    *SnapshotGroupVolumeArgs    `xml:"u:SnapshotGroupVolume,omitempty"`
}
type EnvelopeResponse struct {
	XMLName       xml.Name     `xml:"Envelope"`
	Xmlns         string       `xml:"xmlns:s,attr"`
	EncodingStyle string       `xml:"encodingStyle,attr"`
	Body          BodyResponse `xml:"Body"`
}
type BodyResponse struct {
	XMLName                xml.Name                        `xml:"Body"`
	GetGroupMute           *GetGroupMuteResponse           `xml:"GetGroupMuteResponse,omitempty"`
	SetGroupMute           *SetGroupMuteResponse           `xml:"SetGroupMuteResponse,omitempty"`
	GetGroupVolume         *GetGroupVolumeResponse         `xml:"GetGroupVolumeResponse,omitempty"`
	SetGroupVolume         *SetGroupVolumeResponse         `xml:"SetGroupVolumeResponse,omitempty"`
	SetRelativeGroupVolume *SetRelativeGroupVolumeResponse `xml:"SetRelativeGroupVolumeResponse,omitempty"`
	SnapshotGroupVolume    *SnapshotGroupVolumeResponse    `xml:"SnapshotGroupVolumeResponse,omitempty"`
}

func (s *Service) exec(actionName string, envelope *Envelope) (*EnvelopeResponse, error) {
	marshaled, err := xml.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	postBody := []byte("<?xml version=\"1.0\"?>")
	postBody = append(postBody, marshaled...)
	req, err := http.NewRequest("POST", s.controlEndpoint.String(), bytes.NewBuffer(postBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=\"utf-8\"")
	req.Header.Set("SOAPAction", _ServiceURN+"#"+actionName)
	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	responseBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var envelopeResponse EnvelopeResponse
	err = xml.Unmarshal(responseBody, &envelopeResponse)
	if err != nil {
		return nil, err
	}
	return &envelopeResponse, nil
}

type GetGroupMuteArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type GetGroupMuteResponse struct {
	CurrentMute bool `xml:"CurrentMute"`
}

func (s *Service) GetGroupMute(args *GetGroupMuteArgs) (*GetGroupMuteResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetGroupMute`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetGroupMute: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetGroupMute == nil {
		return nil, errors.New(`unexpected response from service calling grouprenderingcontrol.GetGroupMute()`)
	}

	return r.Body.GetGroupMute, nil
}

type SetGroupMuteArgs struct {
	Xmlns       string `xml:"xmlns:u,attr"`
	InstanceID  uint32 `xml:"InstanceID"`
	DesiredMute bool   `xml:"DesiredMute"`
}
type SetGroupMuteResponse struct {
}

func (s *Service) SetGroupMute(args *SetGroupMuteArgs) (*SetGroupMuteResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetGroupMute`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetGroupMute: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetGroupMute == nil {
		return nil, errors.New(`unexpected response from service calling grouprenderingcontrol.SetGroupMute()`)
	}

	return r.Body.SetGroupMute, nil
}

type GetGroupVolumeArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type GetGroupVolumeResponse struct {
	CurrentVolume uint16 `xml:"CurrentVolume"`
}

func (s *Service) GetGroupVolume(args *GetGroupVolumeArgs) (*GetGroupVolumeResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetGroupVolume`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetGroupVolume: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetGroupVolume == nil {
		return nil, errors.New(`unexpected response from service calling grouprenderingcontrol.GetGroupVolume()`)
	}

	return r.Body.GetGroupVolume, nil
}

type SetGroupVolumeArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Range: 0 -> 100 step: 1
	DesiredVolume uint16 `xml:"DesiredVolume"`
}
type SetGroupVolumeResponse struct {
}

func (s *Service) SetGroupVolume(args *SetGroupVolumeArgs) (*SetGroupVolumeResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetGroupVolume`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetGroupVolume: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetGroupVolume == nil {
		return nil, errors.New(`unexpected response from service calling grouprenderingcontrol.SetGroupVolume()`)
	}

	return r.Body.SetGroupVolume, nil
}

type SetRelativeGroupVolumeArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	Adjustment int32  `xml:"Adjustment"`
}
type SetRelativeGroupVolumeResponse struct {
	NewVolume uint16 `xml:"NewVolume"`
}

func (s *Service) SetRelativeGroupVolume(args *SetRelativeGroupVolumeArgs) (*SetRelativeGroupVolumeResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetRelativeGroupVolume`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetRelativeGroupVolume: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetRelativeGroupVolume == nil {
		return nil, errors.New(`unexpected response from service calling grouprenderingcontrol.SetRelativeGroupVolume()`)
	}

	return r.Body.SetRelativeGroupVolume, nil
}

type SnapshotGroupVolumeArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type SnapshotGroupVolumeResponse struct {
}

func (s *Service) SnapshotGroupVolume(args *SnapshotGroupVolumeArgs) (*SnapshotGroupVolumeResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SnapshotGroupVolume`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SnapshotGroupVolume: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SnapshotGroupVolume == nil {
		return nil, errors.New(`unexpected response from service calling grouprenderingcontrol.SnapshotGroupVolume()`)
	}

	return r.Body.SnapshotGroupVolume, nil
}
