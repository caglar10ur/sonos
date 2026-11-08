package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGeneratedCodeIsUpToDate regenerates every service the way
// makeallservices.sh does and checks the result matches services/.
// Run `go generate` from the repository root to fix a failure.
func TestGeneratedCodeIsUpToDate(t *testing.T) {
	// Same prefixes as makeallservices.sh.
	prefixes := []struct{ prefix, name string }{
		{"/MediaRenderer", "VirtualLineIn"},
		{"/MediaRenderer", "GroupRenderingControl"},
		{"/MediaRenderer", "Queue"},
		{"/MediaRenderer", "AVTransport"},
		{"/MediaRenderer", "ConnectionManager"},
		{"/MediaRenderer", "RenderingControl"},
		{"/MediaServer", "ContentDirectory"},
		{"", "AudioIn"},
		{"", "AlarmClock"},
		{"", "MusicServices"},
		{"", "DeviceProperties"},
		{"", "SystemProperties"},
		{"", "ZoneGroupTopology"},
		{"", "GroupManagement"},
		{"", "QPlay"},
	}

	for _, p := range prefixes {
		t.Run(p.name, func(t *testing.T) {
			scpd, err := os.ReadFile(filepath.Join("xml", p.name+"1.xml"))
			if err != nil {
				t.Fatal(err)
			}
			base := p.prefix + "/" + p.name
			got, err := MakeServiceApi(p.name, base+"/Control", base+"/Event", scpd)
			if err != nil {
				t.Fatal(err)
			}
			pkg := strings.ToLower(p.name)
			want, err := os.ReadFile(filepath.Join("..", "..", "services", pkg, pkg+".go"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("services/%s/%s.go is stale; run go generate", pkg, pkg)
			}
		})
	}
}

func TestValidateRejectsUnknownRelatedStateVariable(t *testing.T) {
	const scpd = `<scpd><serviceStateTable>
<stateVariable sendEvents="no"><name>A</name><dataType>string</dataType></stateVariable>
</serviceStateTable><actionList><action><name>Do</name><argumentList>
<argument><name>X</name><direction>in</direction><relatedStateVariable>Missing</relatedStateVariable></argument>
</argumentList></action></actionList></scpd>`
	if _, err := MakeServiceApi("Test", "/c", "/e", []byte(scpd)); err == nil {
		t.Fatal("expected an error for an unknown related state variable")
	}
}

func TestEnumConst(t *testing.T) {
	for _, tc := range []struct{ typ, value, want string }{
		{"TransportState", "PAUSED_PLAYBACK", "TransportStatePausedPlayback"},
		{"SeekMode", "TRACK_NR", "SeekModeTrackNR"},
		{"Channel", "LF", "ChannelLF"},
		{"Channel", "Master", "ChannelMaster"},
		{"BrowseFlag", "BrowseDirectChildren", "BrowseFlagBrowseDirectChildren"},
		{"TransportPlaySpeed", "1", "TransportPlaySpeed1"},
		{"X", "x-rincon-queue", "XXRinconQueue"},
	} {
		if got := enumConst(tc.typ, tc.value); got != tc.want {
			t.Errorf("enumConst(%q, %q) = %q, want %q", tc.typ, tc.value, got, tc.want)
		}
	}
}

// scpd builds a minimal SCPD with the given state variables and actions.
func scpd(vars, actions string) []byte {
	return []byte(`<scpd><serviceStateTable>` + vars + `</serviceStateTable><actionList>` + actions + `</actionList></scpd>`)
}

func TestValidateRejects(t *testing.T) {
	for name, tc := range map[string]struct{ vars, actions, want string }{
		"numeric enum": {
			vars: `<stateVariable sendEvents="no"><name>A</name><dataType>ui4</dataType>
<allowedValueList><allowedValue>1</allowedValue></allowedValueList></stateVariable>`,
			want: "allowed values on non-string",
		},
		"enum value collision": {
			vars: `<stateVariable sendEvents="no"><name>Mode</name><dataType>string</dataType>
<allowedValueList><allowedValue>A_B</allowedValue><allowedValue>A-B</allowedValue></allowedValueList></stateVariable>`,
			want: "both generate ModeAB",
		},
		"type collision with template": {
			vars: `<stateVariable sendEvents="yes"><name>Service</name><dataType>string</dataType></stateVariable>`,
			want: "both generate Service",
		},
		"action collides with template method": {
			vars:    `<stateVariable sendEvents="no"><name>A</name><dataType>string</dataType></stateVariable>`,
			actions: `<action><name>ParseEvent</name></action>`,
			want:    "both generate method ParseEvent",
		},
		"unsupported data type": {
			vars: `<stateVariable sendEvents="no"><name>A</name><dataType>bin.base64</dataType></stateVariable>`,
			want: "unsupported UPnP dataType",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := MakeServiceApi("Test", "/c", "/e", scpd(tc.vars, tc.actions))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

// uri and char state variables must decode as text: encoding/xml can't
// decode into *url.URL, and a rune would expect a number.
func TestTextualDataTypes(t *testing.T) {
	src, err := MakeServiceApi("Test", "/c", "/e", scpd(
		`<stateVariable sendEvents="yes"><name>Where</name><dataType>uri</dataType></stateVariable>
<stateVariable sendEvents="yes"><name>Letter</name><dataType>char</dataType></stateVariable>`, ""))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"type Where string", "type Letter string"} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("generated code lacks %q", want)
		}
	}
}
