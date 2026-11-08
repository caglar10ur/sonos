package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/caglar10ur/sonos"
)

func TestParsePosition(t *testing.T) {
	for _, tt := range []struct {
		in       string
		want     time.Duration
		relative bool
	}{
		{"1:30", 90 * time.Second, false},
		{"1:02:03", time.Hour + 2*time.Minute + 3*time.Second, false},
		{"90s", 90 * time.Second, false},
		{"+30s", 30 * time.Second, true},
		{"-1:00", -time.Minute, true},
		{"-15s", -15 * time.Second, true},
	} {
		got, relative, err := parsePosition(tt.in)
		if err != nil || got != tt.want || relative != tt.relative {
			t.Errorf("parsePosition(%q) = %v, %v, %v; want %v, %v", tt.in, got, relative, err, tt.want, tt.relative)
		}
	}
	for _, in := range []string{"", "abc", "1:x", "1:2:3:4", "--5s", "-1:-2"} {
		if _, _, err := parsePosition(in); err == nil {
			t.Errorf("parsePosition(%q) should fail", in)
		}
	}
}

func TestFindRoom(t *testing.T) {
	h := &sonos.Household{Rooms: []*sonos.Room{{Name: "Living Room"}, {Name: "Kitchen"}}}
	r, err := findRoom(h, "living room")
	if err != nil || r.Name != "Living Room" {
		t.Errorf("findRoom(living room) = %v, %v", r, err)
	}
	_, err = findRoom(h, "Garage")
	if !errors.Is(err, sonos.ErrRoomNotFound) || !strings.Contains(err.Error(), "Living Room, Kitchen") {
		t.Errorf("findRoom(Garage) = %v; want ErrRoomNotFound listing the rooms", err)
	}
}

// Bad command lines must fail before looking for players.
func TestUsageErrors(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"dance"}, `unknown command "dance"`},
		{[]string{"pause"}, "usage: sonosctl pause ROOM"},
		{[]string{"join", "Kitchen"}, "usage: sonosctl join ROOM TARGET"},
		{[]string{"volume", "Kitchen", "5", "6"}, "usage: sonosctl volume"},
		{[]string{"-nope", "ls"}, "flag provided but not defined"},
	} {
		var stdout, stderr bytes.Buffer
		if code := run(tt.args, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), tt.want) {
			t.Errorf("run(%q) = %d, stderr %q; want 2 and %q", tt.args, code, stderr.String(), tt.want)
		}
	}
}

func TestHelpListsEveryCommand(t *testing.T) {
	var stdout bytes.Buffer
	if code := run([]string{"help"}, &stdout, &stdout); code != 0 {
		t.Fatalf("help exited %d", code)
	}
	for _, c := range commands {
		if !strings.Contains(stdout.String(), "  "+c.name) {
			t.Errorf("help doesn't list %s", c.name)
		}
	}
}

// A negative volume change must reach the command, not the flag parser.
func TestNegativeArgumentsAreNotFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// No players answer within 1ns, so this fails after parsing.
	code := run([]string{"-timeout", "1ns", "volume", "Kitchen", "-5"}, &stdout, &stderr)
	if code != 1 || strings.Contains(stderr.String(), "usage") {
		t.Errorf("run = %d, stderr %q; want a lookup failure", code, stderr.String())
	}
}

func TestAnnouncementDone(t *testing.T) {
	state := func(s sonos.PlaybackState, uri string) *sonos.GroupState {
		st := &sonos.GroupState{}
		st.State = s
		if uri != "" {
			st.Track = &sonos.Track{URI: uri}
		}
		return st
	}
	for _, tt := range []struct {
		name   string
		states []*sonos.GroupState
		done   int // index of the state that ends it, -1 for none
	}{
		{"plays to the end", []*sonos.GroupState{
			state(sonos.Playing, "old"), // the group was playing; not ours yet
			state(sonos.Transitioning, "ding"),
			state(sonos.Playing, "ding"),
			state(sonos.Stopped, "ding"),
		}, 3},
		{"over before seen playing", []*sonos.GroupState{
			state(sonos.Transitioning, "ding"),
			state(sonos.Stopped, "ding"),
		}, 1},
		{"something else is played", []*sonos.GroupState{
			state(sonos.Playing, "ding"),
			state(sonos.Playing, "other"),
		}, 1},
		{"not loaded yet", []*sonos.GroupState{
			state(sonos.Stopped, "old"),
			state(sonos.Transitioning, "ding"),
		}, -1},
	} {
		a := announcement{before: "old"}
		got := -1
		for i, st := range tt.states {
			if a.done(st) {
				got = i
				break
			}
		}
		if got != tt.done {
			t.Errorf("%s: done at state %d, want %d", tt.name, got, tt.done)
		}
	}
}
