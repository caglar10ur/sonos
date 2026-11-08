// https://github.com/ianr0bkny/go-sonos/blob/master/didl/didl.go
//
// go-sonos
// ========
//
// Copyright (c) 2012, Ian T. Richards <ianr@panix.com>
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
//
//   * Redistributions of source code must retain the above copyright notice,
//     this list of conditions and the following disclaimer.
//   * Redistributions in binary form must reproduce the above copyright
//     notice, this list of conditions and the following disclaimer in the
//     documentation and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED
// TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
// PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
// LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
// NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//

// Package didl reads and writes DIDL-Lite, the XML that describes tracks,
// queue items and containers in UPnP AV, e.g. a track's metadata.
package didl

import (
	"encoding/xml"
	"fmt"
	"slices"
	"strings"
)

// Parse parses a DIDL-Lite document, such as the metadata of a track.
func Parse(raw string) (*Lite, error) {
	var l Lite
	if err := xml.Unmarshal([]byte(raw), &l); err != nil {
		return nil, fmt.Errorf("didl: parsing %q: %w", truncate(raw, 120), err)
	}
	return &l, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// First returns the value of the first element in texts, or "" if there is
// none, e.g. didl.First(item.Title).
func First(texts []Text) string {
	if len(texts) == 0 {
		return ""
	}
	return texts[0].Value
}

// Lite is a DIDL-Lite document.
type Lite struct {
	XMLName   xml.Name    `json:"-"`
	Container []Container `xml:"container"`
	Item      []Item      `xml:"item"`
}

// Item is a playable object, such as a track or a queue entry.
type Item struct {
	XMLName xml.Name `json:"-"`
	Object
}

// Container is an object that holds others, such as an album or a playlist.
type Container struct {
	XMLName xml.Name `json:"-"`
	Object
}

// Object holds the properties shared by items and containers. Each property
// may appear more than once; First returns the first value.
type Object struct {
	ID          string `xml:"id,attr"`
	ParentID    string `xml:"parentID,attr"`
	Restricted  bool   `xml:"restricted,attr"`
	Res         []Res  `xml:"res"`
	Title       []Text `xml:"http://purl.org/dc/elements/1.1/ title"`
	Class       []Text `xml:"urn:schemas-upnp-org:metadata-1-0/upnp/ class"`
	AlbumArtURI []Text `xml:"urn:schemas-upnp-org:metadata-1-0/upnp/ albumArtURI"`
	Creator     []Text `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Album       []Text `xml:"urn:schemas-upnp-org:metadata-1-0/upnp/ album"`
	AlbumArtist []Text `xml:"urn:schemas-rinconnetworks-com:metadata-1-0/ albumArtist"`
	Desc        []Desc `xml:"desc"`
}

// Text is a property with a text value, such as dc:title.
type Text struct {
	XMLName xml.Name `json:"-"`
	Value   string   `xml:",chardata"`
}

// Res is a resource of an object: a URI to play it, with its protocol and
// duration.
type Res struct {
	XMLName      xml.Name `json:"-"`
	ProtocolInfo string   `xml:"protocolInfo,attr"`
	Duration     string   `xml:"duration,attr"`
	Value        string   `xml:",chardata"`
}

// Desc is a vendor-specific descriptor, e.g. the account token of a music
// service.
type Desc struct {
	XMLName   xml.Name `json:"-"`
	ID        string   `xml:"id,attr"`
	NameSpace string   `xml:"nameSpace,attr"`
	Value     string   `xml:",chardata"`
}

const emptyDocument = "<DIDL-Lite xmlns=\"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/\"></DIDL-Lite>"

// EmptyDocument returns a DIDL-Lite document without objects, for actions
// that take metadata when there is none.
func EmptyDocument() string {
	return emptyDocument
}

// EmptyDocuments returns num empty documents separated by spaces, as
// AddMultipleURIsToQueue expects for items without metadata. It returns ""
// for num <= 0.
func EmptyDocuments(num int) string {
	if num <= 0 {
		return ""
	}
	return strings.Join(slices.Repeat([]string{emptyDocument}, num), " ")
}
