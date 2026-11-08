package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/caglar10ur/sonos"
	"github.com/caglar10ur/sonos/didl"
)

type roomInfo struct {
	Name     string `json:"name"`
	Model    string `json:"model,omitempty"`
	Speakers int    `json:"speakers"`
	Group    string `json:"group"`
	Playing  string `json:"playing,omitempty"`
}

func cmdList(ctx context.Context, c *cli, _ []string) error {
	h, err := c.household(ctx)
	if err != nil {
		return err
	}
	var rooms []roomInfo
	for _, g := range h.Groups {
		playing := ""
		if np, err := g.NowPlaying(ctx); err == nil {
			playing = describe(np)
		}
		for _, r := range g.Rooms {
			info := roomInfo{Name: r.Name, Speakers: r.Speakers, Group: g.Name(), Playing: playing}
			if zp, err := r.Player(ctx); err == nil {
				info.Model = zp.ModelName()
			}
			rooms = append(rooms, info)
		}
	}
	if c.json {
		return c.printJSON(rooms)
	}
	w := tabwriter.NewWriter(c.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ROOM\tMODEL\tSPEAKERS\tGROUP\tPLAYING")
	for _, r := range rooms {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", r.Name, r.Model, r.Speakers, r.Group, r.Playing)
	}
	return w.Flush()
}

type status struct {
	Group         string      `json:"group"`
	State         string      `json:"state"`
	Track         *trackInfo  `json:"track,omitempty"`
	Position      string      `json:"position,omitempty"`
	QueuePosition int         `json:"queue_position,omitempty"`
	Shuffle       bool        `json:"shuffle"`
	Repeat        string      `json:"repeat"`
	Volume        int         `json:"volume"`
	Rooms         []roomLevel `json:"rooms"`
}

type trackInfo struct {
	Title    string `json:"title,omitempty"`
	Artist   string `json:"artist,omitempty"`
	Album    string `json:"album,omitempty"`
	Duration string `json:"duration,omitempty"`
	URI      string `json:"uri"`
}

type roomLevel struct {
	Name   string `json:"name"`
	Volume int    `json:"volume"`
	Muted  bool   `json:"muted"`
}

func cmdStatus(ctx context.Context, c *cli, args []string) error {
	r, err := c.room(ctx, args[0])
	if err != nil {
		return err
	}
	g := r.Group()
	np, err := g.NowPlaying(ctx)
	if err != nil {
		return err
	}
	mode, err := g.PlayMode(ctx)
	if err != nil {
		return err
	}
	vol, err := g.Volume(ctx)
	if err != nil {
		return err
	}
	st := status{
		Group:         g.Name(),
		State:         stateName(np.State),
		QueuePosition: np.QueuePosition,
		Shuffle:       mode.Shuffle,
		Repeat:        repeatNames[mode.Repeat],
		Volume:        vol,
	}
	if t := np.Track; t != nil {
		st.Track = &trackInfo{Title: t.Title, Artist: t.Artist, Album: t.Album, URI: t.URI}
		if t.Duration > 0 {
			st.Track.Duration = clock(t.Duration)
			st.Position = clock(np.Position)
		}
	}
	for _, room := range g.Rooms {
		l := roomLevel{Name: room.Name}
		if l.Volume, err = room.Volume(ctx); err != nil {
			return err
		}
		if l.Muted, err = room.IsMuted(ctx); err != nil {
			return err
		}
		st.Rooms = append(st.Rooms, l)
	}
	if c.json {
		return c.printJSON(st)
	}

	c.printf("%s: %s\n", st.Group, describe(np))
	if t := st.Track; t != nil {
		if t.Album != "" {
			c.printf("  album     %s\n", t.Album)
		}
		if t.Duration != "" {
			c.printf("  position  %s / %s\n", st.Position, t.Duration)
		}
		if st.QueuePosition > 0 {
			c.printf("  queue     track %d\n", st.QueuePosition)
		}
		c.printf("  uri       %s\n", t.URI)
	}
	c.printf("  mode      shuffle %s, repeat %s\n", onOff(st.Shuffle), st.Repeat)
	c.printf("  volume    %d\n", st.Volume)
	for _, l := range st.Rooms {
		muted := ""
		if l.Muted {
			muted = " (muted)"
		}
		c.printf("    %-14s %d%s\n", l.Name, l.Volume, muted)
	}
	return nil
}

func cmdPlay(ctx context.Context, c *cli, args []string) error {
	r, err := c.room(ctx, args[0])
	if err != nil {
		return err
	}
	g := r.Group()
	if len(args) == 2 {
		if err := g.SetURI(ctx, args[1], ""); err != nil {
			return err
		}
	}
	return g.Play(ctx)
}

func cmdSeek(ctx context.Context, c *cli, args []string) error {
	r, err := c.room(ctx, args[0])
	if err != nil {
		return err
	}
	g := r.Group()
	pos, relative, err := parsePosition(args[1])
	if err != nil {
		return err
	}
	if relative {
		np, err := g.NowPlaying(ctx)
		if err != nil {
			return err
		}
		pos = max(np.Position+pos, 0)
	}
	return g.Seek(ctx, pos)
}

// parsePosition parses an absolute position, 1:30 or 90s, or one relative
// to the current position, +30s or -1:00.
func parsePosition(s string) (d time.Duration, relative bool, err error) {
	sign := time.Duration(1)
	switch {
	case strings.HasPrefix(s, "+"):
		s, relative = s[1:], true
	case strings.HasPrefix(s, "-"):
		s, relative, sign = s[1:], true, -1
	}
	d, err = sonos.ParsePosition(s)
	return sign * d, relative, err
}

var repeatNames = map[sonos.Repeat]string{sonos.RepeatOff: "off", sonos.RepeatAll: "all", sonos.RepeatOne: "one"}

func cmdShuffle(ctx context.Context, c *cli, args []string) error {
	var on bool
	switch args[1] {
	case "on":
		on = true
	case "off":
	default:
		return fmt.Errorf("shuffle is on or off, not %q", args[1])
	}
	return setPlayMode(ctx, c, args[0], func(m *sonos.PlayMode) { m.Shuffle = on })
}

func cmdRepeat(ctx context.Context, c *cli, args []string) error {
	for r, name := range repeatNames {
		if name == args[1] {
			return setPlayMode(ctx, c, args[0], func(m *sonos.PlayMode) { m.Repeat = r })
		}
	}
	return fmt.Errorf("repeat is off, all or one, not %q", args[1])
}

// setPlayMode changes one part of the play mode of a room's group.
func setPlayMode(ctx context.Context, c *cli, room string, change func(*sonos.PlayMode)) error {
	r, err := c.room(ctx, room)
	if err != nil {
		return err
	}
	mode, err := r.Group().PlayMode(ctx)
	if err != nil {
		return err
	}
	change(&mode)
	return r.Group().SetPlayMode(ctx, mode)
}

func cmdLineIn(ctx context.Context, c *cli, args []string) error {
	h, err := c.household(ctx)
	if err != nil {
		return err
	}
	r, err := findRoom(h, args[0])
	if err != nil {
		return err
	}
	from := r
	if len(args) == 2 {
		if from, err = findRoom(h, args[1]); err != nil {
			return err
		}
	}
	if err := r.Group().PlayLineIn(ctx, from); err != nil {
		return err
	}
	return r.Group().Play(ctx)
}

func cmdVolume(ctx context.Context, c *cli, args []string) error {
	r, err := c.room(ctx, args[0])
	if err != nil {
		return err
	}
	return volume(ctx, c, args, r.Name, r.Volume, r.SetVolume)
}

func cmdGroupVolume(ctx context.Context, c *cli, args []string) error {
	r, err := c.room(ctx, args[0])
	if err != nil {
		return err
	}
	g := r.Group()
	return volume(ctx, c, args, g.Name(), g.Volume, g.SetVolume)
}

// volume prints the volume, or sets it to args[1]: a level, or a change
// such as +5 or -5.
func volume(ctx context.Context, c *cli, args []string, name string,
	get func(context.Context) (int, error), set func(context.Context, int) error) error {
	vol, err := get(ctx)
	if err != nil {
		return err
	}
	if len(args) == 2 {
		n, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid volume %q: want 0 to 100, or a change such as +5 or -5", args[1])
		}
		if strings.HasPrefix(args[1], "+") || strings.HasPrefix(args[1], "-") {
			n = min(max(vol+n, 0), 100)
		}
		if err := set(ctx, n); err != nil {
			return err
		}
		vol = n
	}
	if c.json {
		return c.printJSON(map[string]any{"name": name, "volume": vol})
	}
	c.printf("%s: %d\n", name, vol)
	return nil
}

type queueItem struct {
	Position int    `json:"position"`
	Title    string `json:"title"`
	Artist   string `json:"artist,omitempty"`
	Album    string `json:"album,omitempty"`
}

func cmdQueue(ctx context.Context, c *cli, args []string) error {
	r, err := c.room(ctx, args[0])
	if err != nil {
		return err
	}
	g := r.Group()
	if len(args) == 1 {
		return listQueue(ctx, c, g)
	}

	op, rest := args[1], args[2:]
	position := func(s string) (int, error) {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			return 0, fmt.Errorf("invalid queue position %q: want 1 or more", s)
		}
		return n, nil
	}
	switch {
	case op == "add" && len(rest) >= 1:
		at := 0
		if len(rest) == 2 {
			if at, err = position(rest[1]); err != nil {
				return err
			}
		}
		first, err := g.AddToQueue(ctx, rest[0], "", at)
		if err != nil {
			return err
		}
		c.printf("added at %d\n", first)
		return nil
	case op == "rm" && len(rest) == 1:
		n, err := position(rest[0])
		if err != nil {
			return err
		}
		return g.RemoveFromQueue(ctx, n)
	case op == "clear" && len(rest) == 0:
		return g.ClearQueue(ctx)
	case op == "play" && len(rest) == 1:
		n, err := position(rest[0])
		if err != nil {
			return err
		}
		if err := g.PlayQueue(ctx); err != nil {
			return err
		}
		if err := g.SeekTrack(ctx, n); err != nil {
			return err
		}
		return g.Play(ctx)
	}
	return fmt.Errorf("usage: sonosctl queue ROOM [add URI [POS] | rm POS | clear | play POS]")
}

func listQueue(ctx context.Context, c *cli, g *sonos.Group) error {
	items, err := g.Queue(ctx)
	if err != nil {
		return err
	}
	np, err := g.NowPlaying(ctx)
	if err != nil {
		return err
	}
	queue := make([]queueItem, 0, len(items))
	for i, item := range items {
		queue = append(queue, queueItem{
			Position: i + 1,
			Title:    didl.First(item.Title),
			Artist:   didl.First(item.Creator),
			Album:    didl.First(item.Album),
		})
	}
	if c.json {
		return c.printJSON(queue)
	}
	if len(queue) == 0 {
		c.printf("%s: the queue is empty\n", g.Name())
		return nil
	}
	w := tabwriter.NewWriter(c.out, 0, 0, 2, ' ', 0)
	for _, q := range queue {
		mark := " "
		if q.Position == np.QueuePosition {
			mark = ">"
		}
		fmt.Fprintf(w, "%s%4d\t%s\t%s\t%s\n", mark, q.Position, q.Title, q.Artist, q.Album)
	}
	return w.Flush()
}

func cmdJoin(ctx context.Context, c *cli, args []string) error {
	h, err := c.household(ctx)
	if err != nil {
		return err
	}
	r, err := findRoom(h, args[0])
	if err != nil {
		return err
	}
	target, err := findRoom(h, args[1])
	if err != nil {
		return err
	}
	return r.Join(ctx, target.Group())
}

// describe says what a group plays, e.g. "playing  Artist - Title".
func describe(np *sonos.NowPlaying) string {
	state := stateName(np.State)
	if np.Track == nil {
		return state
	}
	var parts []string
	for _, v := range []string{np.Track.Artist, np.Track.Title} {
		if v != "" {
			parts = append(parts, v)
		}
	}
	if len(parts) == 0 {
		return state
	}
	return fmt.Sprintf("%-7s  %s", state, strings.Join(parts, " - "))
}

func stateName(s sonos.PlaybackState) string {
	if s == sonos.Paused {
		return "paused"
	}
	return strings.ToLower(string(s))
}

func clock(d time.Duration) string {
	s := int(d.Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
