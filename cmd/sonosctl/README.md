# sonosctl

Controls the Sonos players on your network from the command line, using
this repository's library. Its source is also the library's largest example.

```sh
go install github.com/caglar10ur/sonos/cmd/sonosctl@latest
```

```console
$ sonosctl ls
ROOM            MODEL            SPEAKERS  GROUP           PLAYING
Kitchen         Sonos Era 300    1         Kitchen         paused   Aerosmith - I Don't Want to Miss a Thing
Living Room     Sonos Arc Ultra  4         Living Room     stopped
$ sonosctl play kitchen
$ sonosctl volume kitchen +5
Kitchen: 31
$ sonosctl watch kitchen
15:04:05  Kitchen       playing  Aerosmith - I Don't Want to Miss a Thing  0:20/4:58  volume 31
```

Run `sonosctl help` for every command. In short:

| | |
|---|---|
| Rooms | `ls`, `status ROOM` |
| Playback | `play ROOM [URI]`, `pause`, `stop`, `next`, `prev`, `seek ROOM 1:30\|+30s`, `shuffle ROOM on\|off`, `repeat ROOM off\|all\|one`, `linein ROOM [FROM]` |
| Volume | `volume ROOM [N\|+N\|-N]`, `group-volume ROOM [...]`, `mute`, `unmute` |
| Queue | `queue ROOM`, `queue ROOM add URI [POS]`, `rm POS`, `clear`, `play POS` |
| Grouping | `join ROOM TARGET`, `leave ROOM` |
| Events | `watch ROOM`, `events ROOM`, `announce ROOM URI [VOLUME]` |

Room names are matched ignoring case. Playback, the queue and play mode act
on the room's group, so `sonosctl pause kitchen` pauses every room playing
with the kitchen. `-json` prints `ls`, `status`, `volume` and `queue` as
JSON, `-timeout` sets how long to look for players, and `-v` logs debug
messages such as subscription renewals.

`announce` saves what the group plays and how loud each room is, plays the
URI, waits for it to end and puts everything back, also after Ctrl-C.

The package documentation has shorter, compile-checked examples:
`go doc -all github.com/caglar10ur/sonos` or pkg.go.dev.

## When nothing is found, or no events arrive

Discovery uses SSDP multicast (239.255.255.250:1900), and events are
delivered by the players connecting back to your machine. Both are easy to
block without noticing:

- **macOS Local Network permission.** Programs need it to send multicast or
  reach local addresses. Without it, sends fail with "no route to host" and
  nothing is found. Allow your terminal app under System Settings → Privacy &
  Security → Local Network. A tmux server started before the permission was
  granted keeps running without it; start a new one (for example
  `tmux -L new`) or run outside tmux.
- **Application firewalls** such as Little Snitch. `go run` builds a new
  binary in a different place every time, so a rule for one build doesn't
  match the next; `go install` sonosctl and allow that binary. Some firewalls
  ask again whenever the binary changes.
- **Players on another subnet or VLAN.** SSDP multicast doesn't cross
  routers. Relay it between the networks (for example with a multicast relay
  on the router) and allow the players' unicast replies back.
- **Events.** Players connect back to the address and port the library
  listens on (a random port by default; pick one with `sonos.WithListenAddr`).
  A firewall between them and your machine must allow that. A subscription
  that succeeds but never delivers an event usually means this connection is
  blocked.
