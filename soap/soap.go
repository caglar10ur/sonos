// Package soap holds the SOAP transport shared by the generated services/*
// packages, and the UPnPError type they return when a device rejects a call.
package soap

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

const (
	EncodingSchema = "http://schemas.xmlsoap.org/soap/encoding/"
	EnvelopeSchema = "http://schemas.xmlsoap.org/soap/envelope/"
)

// UPnPError is returned when a device answers a call with a SOAP fault
// carrying a UPnP error code, e.g. 701 (transition not available).
// Use errors.As to inspect it.
type UPnPError struct {
	Action      string
	Code        int
	Description string
}

func (e *UPnPError) Error() string {
	return fmt.Sprintf("UPnP error %d calling %s: %s", e.Code, e.Action, e.Description)
}

// UPnP error codes a device may return, from the UPnP Device Architecture
// and the AVTransport service specification.
const (
	InvalidAction           = 401
	InvalidArgs             = 402
	ActionFailed            = 501
	ArgumentValueInvalid    = 600
	ArgumentValueOutOfRange = 601
	TransitionNotAvailable  = 701 // e.g. Next on the last track, Seek with nothing loaded
	NoContents              = 702
	FormatNotSupported      = 704 // for playback
	TransportLocked         = 705
	SeekModeNotSupported    = 710
	IllegalSeekTarget       = 711
	PlayModeNotSupported    = 712
	IllegalMIMEType         = 714
	ResourceNotFound        = 716
	PlaySpeedNotSupported   = 717
	InvalidInstanceID       = 718
)

// IsCode reports whether err is, or wraps, a *UPnPError with the given code.
func IsCode(err error, code int) bool {
	var e *UPnPError
	return errors.As(err, &e) && e.Code == code
}

type faultEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		Fault struct {
			Detail struct {
				UPnPError struct {
					ErrorCode        string `xml:"errorCode"`
					ErrorDescription string `xml:"errorDescription"`
				} `xml:"UPnPError"`
			} `xml:"detail"`
		} `xml:"Fault"`
	} `xml:"Body"`
}

// Call invokes action on the service at endpoint. args is the action's
// argument struct: its fields become the action element's children, and a
// nil pointer sends their zero values. A nil args sends no arguments. The <actionResponse> element is decoded into a new Resp.
func Call[Resp any](ctx context.Context, client *http.Client, endpoint *url.URL, serviceURN, action string, args any) (*Resp, error) {
	postBody, err := xml.Marshal(requestEnvelope{
		Xmlns:         EnvelopeSchema,
		EncodingStyle: EncodingSchema,
		Body:          requestBody{action: action, serviceURN: serviceURN, args: args},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(postBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("SOAPAction", serviceURN+"#"+action)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		var fe faultEnvelope
		if xml.Unmarshal(body, &fe) == nil {
			upnp := fe.Body.Fault.Detail.UPnPError
			if code, err := strconv.Atoi(strings.TrimSpace(upnp.ErrorCode)); err == nil {
				return nil, &UPnPError{Action: action, Code: code, Description: upnp.ErrorDescription}
			}
		}
		return nil, fmt.Errorf("%s calling %s: %s", res.Status, action, strings.TrimSpace(string(body)))
	}

	resp := new(Resp)
	if err := xml.Unmarshal(body, &responseEnvelope{action: action, resp: resp}); err != nil {
		return nil, fmt.Errorf("decoding %s response: %w", action, err)
	}
	return resp, nil
}

// requestEnvelope is the SOAP envelope of a call. encoding/xml has no
// namespace prefix support, so the "s:" and "u:" prefixes are spelled out
// in the element names.
type requestEnvelope struct {
	XMLName       xml.Name    `xml:"s:Envelope"`
	Xmlns         string      `xml:"xmlns:s,attr"`
	EncodingStyle string      `xml:"s:encodingStyle,attr"`
	Body          requestBody `xml:"s:Body"`
}

type requestBody struct {
	action, serviceURN string
	args               any
}

// MarshalXML writes <s:Body><u:action xmlns:u="urn">args</u:action></s:Body>.
func (b requestBody) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	el := xml.StartElement{
		Name: xml.Name{Local: "u:" + b.action},
		Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns:u"}, Value: b.serviceURN}},
	}
	args := b.args
	// A nil *Args sends the zero value of every argument, like an empty one.
	if rv := reflect.ValueOf(args); rv.Kind() == reflect.Pointer && rv.IsNil() {
		args = reflect.New(rv.Type().Elem()).Interface()
	}
	if args == nil {
		if err := e.EncodeToken(el); err != nil {
			return err
		}
		if err := e.EncodeToken(el.End()); err != nil {
			return err
		}
	} else if err := e.EncodeElement(args, el); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

// responseEnvelope decodes the <actionResponse> element of a response
// envelope into resp, and fails if there is none.
type responseEnvelope struct {
	action string
	resp   any
}

func (r *responseEnvelope) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	want := r.action + "Response"
	found := false
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == want {
				if err := d.DecodeElement(r.resp, &t); err != nil {
					return err
				}
				found = true
			}
		case xml.EndElement:
			if t.Name == start.Name {
				if !found {
					return fmt.Errorf("no %s element", want)
				}
				return nil
			}
		}
	}
}
