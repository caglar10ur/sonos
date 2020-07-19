// Code generated by makeservice. DO NOT EDIT.

// Package renderingcontrol is a generated RenderingControl package.
package renderingcontrol

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	_ServiceURN     = "urn:schemas-upnp-org:service:RenderingControl:1"
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

	c, err := url.Parse("/MediaRenderer/RenderingControl/Control")
	if nil != err {
		panic(err)
	}
	e, err := url.Parse("/MediaRenderer/RenderingControl/Event")
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
	XMLName                  xml.Name                      `xml:"s:Body"`
	GetMute                  *GetMuteArgs                  `xml:"u:GetMute,omitempty"`
	SetMute                  *SetMuteArgs                  `xml:"u:SetMute,omitempty"`
	ResetBasicEQ             *ResetBasicEQArgs             `xml:"u:ResetBasicEQ,omitempty"`
	ResetExtEQ               *ResetExtEQArgs               `xml:"u:ResetExtEQ,omitempty"`
	GetVolume                *GetVolumeArgs                `xml:"u:GetVolume,omitempty"`
	SetVolume                *SetVolumeArgs                `xml:"u:SetVolume,omitempty"`
	SetRelativeVolume        *SetRelativeVolumeArgs        `xml:"u:SetRelativeVolume,omitempty"`
	GetVolumeDB              *GetVolumeDBArgs              `xml:"u:GetVolumeDB,omitempty"`
	SetVolumeDB              *SetVolumeDBArgs              `xml:"u:SetVolumeDB,omitempty"`
	GetVolumeDBRange         *GetVolumeDBRangeArgs         `xml:"u:GetVolumeDBRange,omitempty"`
	GetBass                  *GetBassArgs                  `xml:"u:GetBass,omitempty"`
	SetBass                  *SetBassArgs                  `xml:"u:SetBass,omitempty"`
	GetTreble                *GetTrebleArgs                `xml:"u:GetTreble,omitempty"`
	SetTreble                *SetTrebleArgs                `xml:"u:SetTreble,omitempty"`
	GetEQ                    *GetEQArgs                    `xml:"u:GetEQ,omitempty"`
	SetEQ                    *SetEQArgs                    `xml:"u:SetEQ,omitempty"`
	GetLoudness              *GetLoudnessArgs              `xml:"u:GetLoudness,omitempty"`
	SetLoudness              *SetLoudnessArgs              `xml:"u:SetLoudness,omitempty"`
	GetSupportsOutputFixed   *GetSupportsOutputFixedArgs   `xml:"u:GetSupportsOutputFixed,omitempty"`
	GetOutputFixed           *GetOutputFixedArgs           `xml:"u:GetOutputFixed,omitempty"`
	SetOutputFixed           *SetOutputFixedArgs           `xml:"u:SetOutputFixed,omitempty"`
	GetHeadphoneConnected    *GetHeadphoneConnectedArgs    `xml:"u:GetHeadphoneConnected,omitempty"`
	RampToVolume             *RampToVolumeArgs             `xml:"u:RampToVolume,omitempty"`
	RestoreVolumePriorToRamp *RestoreVolumePriorToRampArgs `xml:"u:RestoreVolumePriorToRamp,omitempty"`
	SetChannelMap            *SetChannelMapArgs            `xml:"u:SetChannelMap,omitempty"`
	SetRoomCalibrationX      *SetRoomCalibrationXArgs      `xml:"u:SetRoomCalibrationX,omitempty"`
	GetRoomCalibrationStatus *GetRoomCalibrationStatusArgs `xml:"u:GetRoomCalibrationStatus,omitempty"`
	SetRoomCalibrationStatus *SetRoomCalibrationStatusArgs `xml:"u:SetRoomCalibrationStatus,omitempty"`
}
type EnvelopeResponse struct {
	XMLName       xml.Name     `xml:"Envelope"`
	Xmlns         string       `xml:"xmlns:s,attr"`
	EncodingStyle string       `xml:"encodingStyle,attr"`
	Body          BodyResponse `xml:"Body"`
}
type BodyResponse struct {
	XMLName                  xml.Name                          `xml:"Body"`
	GetMute                  *GetMuteResponse                  `xml:"GetMuteResponse,omitempty"`
	SetMute                  *SetMuteResponse                  `xml:"SetMuteResponse,omitempty"`
	ResetBasicEQ             *ResetBasicEQResponse             `xml:"ResetBasicEQResponse,omitempty"`
	ResetExtEQ               *ResetExtEQResponse               `xml:"ResetExtEQResponse,omitempty"`
	GetVolume                *GetVolumeResponse                `xml:"GetVolumeResponse,omitempty"`
	SetVolume                *SetVolumeResponse                `xml:"SetVolumeResponse,omitempty"`
	SetRelativeVolume        *SetRelativeVolumeResponse        `xml:"SetRelativeVolumeResponse,omitempty"`
	GetVolumeDB              *GetVolumeDBResponse              `xml:"GetVolumeDBResponse,omitempty"`
	SetVolumeDB              *SetVolumeDBResponse              `xml:"SetVolumeDBResponse,omitempty"`
	GetVolumeDBRange         *GetVolumeDBRangeResponse         `xml:"GetVolumeDBRangeResponse,omitempty"`
	GetBass                  *GetBassResponse                  `xml:"GetBassResponse,omitempty"`
	SetBass                  *SetBassResponse                  `xml:"SetBassResponse,omitempty"`
	GetTreble                *GetTrebleResponse                `xml:"GetTrebleResponse,omitempty"`
	SetTreble                *SetTrebleResponse                `xml:"SetTrebleResponse,omitempty"`
	GetEQ                    *GetEQResponse                    `xml:"GetEQResponse,omitempty"`
	SetEQ                    *SetEQResponse                    `xml:"SetEQResponse,omitempty"`
	GetLoudness              *GetLoudnessResponse              `xml:"GetLoudnessResponse,omitempty"`
	SetLoudness              *SetLoudnessResponse              `xml:"SetLoudnessResponse,omitempty"`
	GetSupportsOutputFixed   *GetSupportsOutputFixedResponse   `xml:"GetSupportsOutputFixedResponse,omitempty"`
	GetOutputFixed           *GetOutputFixedResponse           `xml:"GetOutputFixedResponse,omitempty"`
	SetOutputFixed           *SetOutputFixedResponse           `xml:"SetOutputFixedResponse,omitempty"`
	GetHeadphoneConnected    *GetHeadphoneConnectedResponse    `xml:"GetHeadphoneConnectedResponse,omitempty"`
	RampToVolume             *RampToVolumeResponse             `xml:"RampToVolumeResponse,omitempty"`
	RestoreVolumePriorToRamp *RestoreVolumePriorToRampResponse `xml:"RestoreVolumePriorToRampResponse,omitempty"`
	SetChannelMap            *SetChannelMapResponse            `xml:"SetChannelMapResponse,omitempty"`
	SetRoomCalibrationX      *SetRoomCalibrationXResponse      `xml:"SetRoomCalibrationXResponse,omitempty"`
	GetRoomCalibrationStatus *GetRoomCalibrationStatusResponse `xml:"GetRoomCalibrationStatusResponse,omitempty"`
	SetRoomCalibrationStatus *SetRoomCalibrationStatusResponse `xml:"SetRoomCalibrationStatusResponse,omitempty"`
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

type GetMuteArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	// Allowed Value: SpeakerOnly
	Channel string `xml:"Channel"`
}
type GetMuteResponse struct {
	CurrentMute bool `xml:"CurrentMute"`
}

func (s *Service) GetMute(args *GetMuteArgs) (*GetMuteResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetMute`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetMute: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetMute == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetMute()`)
	}

	return r.Body.GetMute, nil
}

type SetMuteArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	// Allowed Value: SpeakerOnly
	Channel     string `xml:"Channel"`
	DesiredMute bool   `xml:"DesiredMute"`
}
type SetMuteResponse struct {
}

func (s *Service) SetMute(args *SetMuteArgs) (*SetMuteResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetMute`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetMute: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetMute == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetMute()`)
	}

	return r.Body.SetMute, nil
}

type ResetBasicEQArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type ResetBasicEQResponse struct {
	Bass        int16  `xml:"Bass"`
	Treble      int16  `xml:"Treble"`
	Loudness    bool   `xml:"Loudness"`
	LeftVolume  uint16 `xml:"LeftVolume"`
	RightVolume uint16 `xml:"RightVolume"`
}

func (s *Service) ResetBasicEQ(args *ResetBasicEQArgs) (*ResetBasicEQResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`ResetBasicEQ`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{ResetBasicEQ: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.ResetBasicEQ == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.ResetBasicEQ()`)
	}

	return r.Body.ResetBasicEQ, nil
}

type ResetExtEQArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	EQType     string `xml:"EQType"`
}
type ResetExtEQResponse struct {
}

func (s *Service) ResetExtEQ(args *ResetExtEQArgs) (*ResetExtEQResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`ResetExtEQ`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{ResetExtEQ: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.ResetExtEQ == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.ResetExtEQ()`)
	}

	return r.Body.ResetExtEQ, nil
}

type GetVolumeArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel string `xml:"Channel"`
}
type GetVolumeResponse struct {
	CurrentVolume uint16 `xml:"CurrentVolume"`
}

func (s *Service) GetVolume(args *GetVolumeArgs) (*GetVolumeResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetVolume`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetVolume: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetVolume == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetVolume()`)
	}

	return r.Body.GetVolume, nil
}

type SetVolumeArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel string `xml:"Channel"`
	// Allowed Range: 0 -> 100 step: 1
	DesiredVolume uint16 `xml:"DesiredVolume"`
}
type SetVolumeResponse struct {
}

func (s *Service) SetVolume(args *SetVolumeArgs) (*SetVolumeResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetVolume`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetVolume: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetVolume == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetVolume()`)
	}

	return r.Body.SetVolume, nil
}

type SetRelativeVolumeArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel    string `xml:"Channel"`
	Adjustment int32  `xml:"Adjustment"`
}
type SetRelativeVolumeResponse struct {
	NewVolume uint16 `xml:"NewVolume"`
}

func (s *Service) SetRelativeVolume(args *SetRelativeVolumeArgs) (*SetRelativeVolumeResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetRelativeVolume`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetRelativeVolume: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetRelativeVolume == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetRelativeVolume()`)
	}

	return r.Body.SetRelativeVolume, nil
}

type GetVolumeDBArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel string `xml:"Channel"`
}
type GetVolumeDBResponse struct {
	CurrentVolume int16 `xml:"CurrentVolume"`
}

func (s *Service) GetVolumeDB(args *GetVolumeDBArgs) (*GetVolumeDBResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetVolumeDB`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetVolumeDB: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetVolumeDB == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetVolumeDB()`)
	}

	return r.Body.GetVolumeDB, nil
}

type SetVolumeDBArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel       string `xml:"Channel"`
	DesiredVolume int16  `xml:"DesiredVolume"`
}
type SetVolumeDBResponse struct {
}

func (s *Service) SetVolumeDB(args *SetVolumeDBArgs) (*SetVolumeDBResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetVolumeDB`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetVolumeDB: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetVolumeDB == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetVolumeDB()`)
	}

	return r.Body.SetVolumeDB, nil
}

type GetVolumeDBRangeArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel string `xml:"Channel"`
}
type GetVolumeDBRangeResponse struct {
	MinValue int16 `xml:"MinValue"`
	MaxValue int16 `xml:"MaxValue"`
}

func (s *Service) GetVolumeDBRange(args *GetVolumeDBRangeArgs) (*GetVolumeDBRangeResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetVolumeDBRange`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetVolumeDBRange: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetVolumeDBRange == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetVolumeDBRange()`)
	}

	return r.Body.GetVolumeDBRange, nil
}

type GetBassArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type GetBassResponse struct {
	CurrentBass int16 `xml:"CurrentBass"`
}

func (s *Service) GetBass(args *GetBassArgs) (*GetBassResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetBass`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetBass: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetBass == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetBass()`)
	}

	return r.Body.GetBass, nil
}

type SetBassArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Range: -10 -> 10 step: 1
	DesiredBass int16 `xml:"DesiredBass"`
}
type SetBassResponse struct {
}

func (s *Service) SetBass(args *SetBassArgs) (*SetBassResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetBass`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetBass: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetBass == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetBass()`)
	}

	return r.Body.SetBass, nil
}

type GetTrebleArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type GetTrebleResponse struct {
	CurrentTreble int16 `xml:"CurrentTreble"`
}

func (s *Service) GetTreble(args *GetTrebleArgs) (*GetTrebleResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetTreble`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetTreble: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetTreble == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetTreble()`)
	}

	return r.Body.GetTreble, nil
}

type SetTrebleArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Range: -10 -> 10 step: 1
	DesiredTreble int16 `xml:"DesiredTreble"`
}
type SetTrebleResponse struct {
}

func (s *Service) SetTreble(args *SetTrebleArgs) (*SetTrebleResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetTreble`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetTreble: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetTreble == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetTreble()`)
	}

	return r.Body.SetTreble, nil
}

type GetEQArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	EQType     string `xml:"EQType"`
}
type GetEQResponse struct {
	CurrentValue int16 `xml:"CurrentValue"`
}

func (s *Service) GetEQ(args *GetEQArgs) (*GetEQResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetEQ`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetEQ: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetEQ == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetEQ()`)
	}

	return r.Body.GetEQ, nil
}

type SetEQArgs struct {
	Xmlns        string `xml:"xmlns:u,attr"`
	InstanceID   uint32 `xml:"InstanceID"`
	EQType       string `xml:"EQType"`
	DesiredValue int16  `xml:"DesiredValue"`
}
type SetEQResponse struct {
}

func (s *Service) SetEQ(args *SetEQArgs) (*SetEQResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetEQ`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetEQ: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetEQ == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetEQ()`)
	}

	return r.Body.SetEQ, nil
}

type GetLoudnessArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel string `xml:"Channel"`
}
type GetLoudnessResponse struct {
	CurrentLoudness bool `xml:"CurrentLoudness"`
}

func (s *Service) GetLoudness(args *GetLoudnessArgs) (*GetLoudnessResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetLoudness`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetLoudness: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetLoudness == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetLoudness()`)
	}

	return r.Body.GetLoudness, nil
}

type SetLoudnessArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel         string `xml:"Channel"`
	DesiredLoudness bool   `xml:"DesiredLoudness"`
}
type SetLoudnessResponse struct {
}

func (s *Service) SetLoudness(args *SetLoudnessArgs) (*SetLoudnessResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetLoudness`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetLoudness: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetLoudness == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetLoudness()`)
	}

	return r.Body.SetLoudness, nil
}

type GetSupportsOutputFixedArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type GetSupportsOutputFixedResponse struct {
	CurrentSupportsFixed bool `xml:"CurrentSupportsFixed"`
}

func (s *Service) GetSupportsOutputFixed(args *GetSupportsOutputFixedArgs) (*GetSupportsOutputFixedResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetSupportsOutputFixed`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetSupportsOutputFixed: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetSupportsOutputFixed == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetSupportsOutputFixed()`)
	}

	return r.Body.GetSupportsOutputFixed, nil
}

type GetOutputFixedArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type GetOutputFixedResponse struct {
	CurrentFixed bool `xml:"CurrentFixed"`
}

func (s *Service) GetOutputFixed(args *GetOutputFixedArgs) (*GetOutputFixedResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetOutputFixed`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetOutputFixed: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetOutputFixed == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetOutputFixed()`)
	}

	return r.Body.GetOutputFixed, nil
}

type SetOutputFixedArgs struct {
	Xmlns        string `xml:"xmlns:u,attr"`
	InstanceID   uint32 `xml:"InstanceID"`
	DesiredFixed bool   `xml:"DesiredFixed"`
}
type SetOutputFixedResponse struct {
}

func (s *Service) SetOutputFixed(args *SetOutputFixedArgs) (*SetOutputFixedResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetOutputFixed`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetOutputFixed: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetOutputFixed == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetOutputFixed()`)
	}

	return r.Body.SetOutputFixed, nil
}

type GetHeadphoneConnectedArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type GetHeadphoneConnectedResponse struct {
	CurrentHeadphoneConnected bool `xml:"CurrentHeadphoneConnected"`
}

func (s *Service) GetHeadphoneConnected(args *GetHeadphoneConnectedArgs) (*GetHeadphoneConnectedResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetHeadphoneConnected`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetHeadphoneConnected: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetHeadphoneConnected == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetHeadphoneConnected()`)
	}

	return r.Body.GetHeadphoneConnected, nil
}

type RampToVolumeArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel string `xml:"Channel"`
	// Allowed Value: SLEEP_TIMER_RAMP_TYPE
	// Allowed Value: ALARM_RAMP_TYPE
	// Allowed Value: AUTOPLAY_RAMP_TYPE
	RampType string `xml:"RampType"`
	// Allowed Range: 0 -> 100 step: 1
	DesiredVolume    uint16 `xml:"DesiredVolume"`
	ResetVolumeAfter bool   `xml:"ResetVolumeAfter"`
	ProgramURI       string `xml:"ProgramURI"`
}
type RampToVolumeResponse struct {
	RampTime uint32 `xml:"RampTime"`
}

func (s *Service) RampToVolume(args *RampToVolumeArgs) (*RampToVolumeResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`RampToVolume`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{RampToVolume: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.RampToVolume == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.RampToVolume()`)
	}

	return r.Body.RampToVolume, nil
}

type RestoreVolumePriorToRampArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	// Allowed Value: Master
	// Allowed Value: LF
	// Allowed Value: RF
	Channel string `xml:"Channel"`
}
type RestoreVolumePriorToRampResponse struct {
}

func (s *Service) RestoreVolumePriorToRamp(args *RestoreVolumePriorToRampArgs) (*RestoreVolumePriorToRampResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`RestoreVolumePriorToRamp`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{RestoreVolumePriorToRamp: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.RestoreVolumePriorToRamp == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.RestoreVolumePriorToRamp()`)
	}

	return r.Body.RestoreVolumePriorToRamp, nil
}

type SetChannelMapArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
	ChannelMap string `xml:"ChannelMap"`
}
type SetChannelMapResponse struct {
}

func (s *Service) SetChannelMap(args *SetChannelMapArgs) (*SetChannelMapResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetChannelMap`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetChannelMap: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetChannelMap == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetChannelMap()`)
	}

	return r.Body.SetChannelMap, nil
}

type SetRoomCalibrationXArgs struct {
	Xmlns           string `xml:"xmlns:u,attr"`
	InstanceID      uint32 `xml:"InstanceID"`
	CalibrationID   string `xml:"CalibrationID"`
	Coefficients    string `xml:"Coefficients"`
	CalibrationMode string `xml:"CalibrationMode"`
}
type SetRoomCalibrationXResponse struct {
}

func (s *Service) SetRoomCalibrationX(args *SetRoomCalibrationXArgs) (*SetRoomCalibrationXResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetRoomCalibrationX`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetRoomCalibrationX: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetRoomCalibrationX == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetRoomCalibrationX()`)
	}

	return r.Body.SetRoomCalibrationX, nil
}

type GetRoomCalibrationStatusArgs struct {
	Xmlns      string `xml:"xmlns:u,attr"`
	InstanceID uint32 `xml:"InstanceID"`
}
type GetRoomCalibrationStatusResponse struct {
	RoomCalibrationEnabled   bool `xml:"RoomCalibrationEnabled"`
	RoomCalibrationAvailable bool `xml:"RoomCalibrationAvailable"`
}

func (s *Service) GetRoomCalibrationStatus(args *GetRoomCalibrationStatusArgs) (*GetRoomCalibrationStatusResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`GetRoomCalibrationStatus`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{GetRoomCalibrationStatus: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.GetRoomCalibrationStatus == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.GetRoomCalibrationStatus()`)
	}

	return r.Body.GetRoomCalibrationStatus, nil
}

type SetRoomCalibrationStatusArgs struct {
	Xmlns                  string `xml:"xmlns:u,attr"`
	InstanceID             uint32 `xml:"InstanceID"`
	RoomCalibrationEnabled bool   `xml:"RoomCalibrationEnabled"`
}
type SetRoomCalibrationStatusResponse struct {
}

func (s *Service) SetRoomCalibrationStatus(args *SetRoomCalibrationStatusArgs) (*SetRoomCalibrationStatusResponse, error) {
	args.Xmlns = _ServiceURN
	r, err := s.exec(`SetRoomCalibrationStatus`,
		&Envelope{
			EncodingStyle: _EncodingSchema,
			Xmlns:         _EnvelopeSchema,
			Body:          Body{SetRoomCalibrationStatus: args},
		})
	if err != nil {
		return nil, err
	}
	if r.Body.SetRoomCalibrationStatus == nil {
		return nil, errors.New(`unexpected response from service calling renderingcontrol.SetRoomCalibrationStatus()`)
	}

	return r.Body.SetRoomCalibrationStatus, nil
}
