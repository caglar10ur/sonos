// Command sonosctl controls the Sonos players on the local network.
//
//	sonosctl ls
//	sonosctl play Kitchen https://example.com/stream.mp3
//	sonosctl volume Kitchen +5
//	sonosctl watch Kitchen
//
// Run sonosctl help for every command.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"

	"github.com/caglar10ur/sonos"
)

// command is a sonosctl subcommand.
type command struct {
	name  string
	args  string // usage after the name, e.g. "ROOM [URI]"
	help  string
	nargs [2]int // min and max number of arguments
	run   func(ctx context.Context, c *cli, args []string) error
}

var commands = []command{
	{"ls", "", "List rooms, their groups and what they play", [2]int{0, 0}, cmdList},
	{"status", "ROOM", "Show what a room's group plays, its play mode and volumes", [2]int{1, 1}, cmdStatus},

	{"play", "ROOM [URI]", "Resume playback, or play a URI", [2]int{1, 2}, cmdPlay},
	{"pause", "ROOM", "Pause playback", [2]int{1, 1}, groupAction((*sonos.Group).Pause)},
	{"stop", "ROOM", "Stop playback", [2]int{1, 1}, groupAction((*sonos.Group).Stop)},
	{"next", "ROOM", "Skip to the next track", [2]int{1, 1}, groupAction((*sonos.Group).Next)},
	{"prev", "ROOM", "Go back to the previous track", [2]int{1, 1}, groupAction((*sonos.Group).Previous)},
	{"seek", "ROOM POSITION", "Move within the track: 1:30, 90s, or +30s/-30s from where it is", [2]int{2, 2}, cmdSeek},
	{"shuffle", "ROOM on|off", "Turn shuffle on or off", [2]int{2, 2}, cmdShuffle},
	{"repeat", "ROOM off|all|one", "Repeat nothing, the queue or the track", [2]int{2, 2}, cmdRepeat},
	{"linein", "ROOM [FROM]", "Play the line-in of FROM (default: ROOM itself)", [2]int{1, 2}, cmdLineIn},

	{"volume", "ROOM [N|+N|-N]", "Show or set a room's volume, 0 to 100", [2]int{1, 2}, cmdVolume},
	{"group-volume", "ROOM [N|+N|-N]", "Show or set the volume of a room's group", [2]int{1, 2}, cmdGroupVolume},
	{"mute", "ROOM", "Mute a room", [2]int{1, 1}, roomAction((*sonos.Room).Mute)},
	{"unmute", "ROOM", "Unmute a room", [2]int{1, 1}, roomAction((*sonos.Room).Unmute)},

	{"queue", "ROOM [OP]", "List the queue of a room's group, or edit it; OP is add URI [POS], rm POS, clear or play POS", [2]int{1, 4}, cmdQueue},

	{"join", "ROOM TARGET", "Make ROOM play with TARGET's group", [2]int{2, 2}, cmdJoin},
	{"leave", "ROOM", "Take ROOM out of its group", [2]int{1, 1}, roomAction((*sonos.Room).Leave)},

	{"watch", "ROOM", "Print a line whenever what ROOM plays, or how loud, changes", [2]int{1, 1}, cmdWatch},
	{"events", "ROOM", "Print every raw event of ROOM's player", [2]int{1, 1}, cmdEvents},
	{"announce", "ROOM URI [VOLUME]", "Play URI, e.g. a doorbell, then go back to what was playing", [2]int{2, 3}, cmdAnnounce},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(argv []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sonosctl", flag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 10*time.Second, "how long to look for players")
	asJSON := fs.Bool("json", false, "print JSON, for ls, status, volume and queue")
	verbose := fs.Bool("v", false, "log debug messages, e.g. subscription renewals")
	fs.Usage = func() { usage(stderr, fs) }
	if err := fs.Parse(argv); err != nil {
		return 2
	}
	if fs.NArg() == 0 || fs.Arg(0) == "help" {
		usage(stdout, fs)
		return 0
	}

	i := slices.IndexFunc(commands, func(c command) bool { return c.name == fs.Arg(0) })
	if i < 0 {
		fmt.Fprintf(stderr, "sonosctl: unknown command %q; run sonosctl help\n", fs.Arg(0))
		return 2
	}
	cmd, args := commands[i], fs.Args()[1:]
	if len(args) < cmd.nargs[0] || len(args) > cmd.nargs[1] {
		fmt.Fprintf(stderr, "usage: sonosctl %s %s\n", cmd.name, cmd.args)
		return 2
	}

	level := slog.LevelWarn
	if *verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	s, err := sonos.New(sonos.WithLogger(logger))
	if err != nil {
		fmt.Fprintln(stderr, "sonosctl:", err)
		return 1
	}
	defer s.Close()

	c := &cli{sonos: s, timeout: *timeout, json: *asJSON, out: stdout, errOut: stderr}
	if err := cmd.run(ctx, c, args); err != nil && ctx.Err() == nil {
		fmt.Fprintln(stderr, "sonosctl:", err)
		return 1
	}
	return 0
}

func usage(w io.Writer, fs *flag.FlagSet) {
	fmt.Fprintln(w, "usage: sonosctl [flags] COMMAND [arguments]")
	fmt.Fprintln(w, "\nCommands:")
	width := 0
	for _, c := range commands {
		width = max(width, len(c.name)+1+len(c.args))
	}
	for _, c := range commands {
		fmt.Fprintf(w, "  %-*s  %s\n", width, strings.TrimSpace(c.name+" "+c.args), c.help)
	}
	fmt.Fprintln(w, "\nROOM is a room's name, in any case, e.g. kitchen or \"living room\".")
	fmt.Fprintln(w, "\nFlags:")
	fs.SetOutput(w)
	fs.PrintDefaults()
}

// cli is what commands share.
type cli struct {
	sonos   *sonos.Sonos
	timeout time.Duration
	json    bool
	out     io.Writer
	errOut  io.Writer
}

// household finds the players, giving up after the -timeout.
func (c *cli) household(ctx context.Context) (*sonos.Household, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	h, err := c.sonos.Household(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("%w (see the README if your players are on the network)", err)
	}
	return h, err
}

// room returns the room with the given name, ignoring case.
func (c *cli) room(ctx context.Context, name string) (*sonos.Room, error) {
	h, err := c.household(ctx)
	if err != nil {
		return nil, err
	}
	return findRoom(h, name)
}

func findRoom(h *sonos.Household, name string) (*sonos.Room, error) {
	var names []string
	for _, r := range h.Rooms {
		if strings.EqualFold(r.Name, name) {
			return r, nil
		}
		names = append(names, r.Name)
	}
	return nil, fmt.Errorf("%w: %q; the rooms are %s", sonos.ErrRoomNotFound, name, strings.Join(names, ", "))
}

func (c *cli) printf(format string, args ...any) {
	fmt.Fprintf(c.out, format, args...)
}

// printJSON prints v as indented JSON.
func (c *cli) printJSON(v any) error {
	enc := json.NewEncoder(c.out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// roomAction is a command that runs fn on a room.
func roomAction(fn func(*sonos.Room, context.Context) error) func(context.Context, *cli, []string) error {
	return func(ctx context.Context, c *cli, args []string) error {
		r, err := c.room(ctx, args[0])
		if err != nil {
			return err
		}
		return fn(r, ctx)
	}
}

// groupAction is a command that runs fn on a room's group.
func groupAction(fn func(*sonos.Group, context.Context) error) func(context.Context, *cli, []string) error {
	return func(ctx context.Context, c *cli, args []string) error {
		r, err := c.room(ctx, args[0])
		if err != nil {
			return err
		}
		return fn(r.Group(), ctx)
	}
}
