package sonos

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/caglar10ur/sonos/didl"
)

// PlaybackState is what a group's playback is doing.
type PlaybackState string

// The playback states a player reports. Other values are passed through
// as they are.
const (
	Stopped       PlaybackState = "STOPPED"
	Playing       PlaybackState = "PLAYING"
	Paused        PlaybackState = "PAUSED_PLAYBACK"
	Transitioning PlaybackState = "TRANSITIONING"
)

// NowPlaying is the state of a group's playback.
type NowPlaying struct {
	State PlaybackState
	// Track is what is loaded, nil if nothing is.
	Track *Track
	// Position is how far into Track playback was at time At; see
	// PositionAt.
	Position time.Duration
	At       time.Time
	// QueuePosition is Track's 1-based position in the queue, or 0 when the
	// group isn't playing from its queue.
	QueuePosition int
}

// PositionAt returns how far into the track playback is at time t: Position
// plus the time since At while playing, but not past the track's end.
func (np *NowPlaying) PositionAt(t time.Time) time.Duration {
	if np.State != Playing || np.At.IsZero() || t.Before(np.At) {
		return np.Position
	}
	pos := np.Position + t.Sub(np.At)
	if np.Track != nil && np.Track.Duration > 0 {
		pos = min(pos, np.Track.Duration)
	}
	return pos
}

// Track describes a track or a stream. Fields the player doesn't report
// are empty, e.g. Duration for a radio stream.
type Track struct {
	Title  string
	Artist string
	Album  string
	// AlbumArtURI is an absolute URL, usually served by the player itself.
	AlbumArtURI string
	// URI is what the player plays, e.g. an x-sonos-spotify: or http: URI.
	URI      string
	Duration time.Duration
}

// trackFrom builds a Track from a track's URI, DIDL-Lite metadata and
// duration as AVTransport reports them. Relative album art URLs are
// resolved against base, the player's location.
func trackFrom(uri, metadata, duration string, base *url.URL) *Track {
	if uri == "" && (metadata == "" || metadata == "NOT_IMPLEMENTED") {
		return nil
	}
	t := &Track{URI: uri, Duration: parseDuration(duration)}
	if meta, err := didl.Parse(metadata); err == nil && len(meta.Item) > 0 {
		item := meta.Item[0]
		t.Title = didl.First(item.Title)
		t.Artist = didl.First(item.Creator)
		t.Album = didl.First(item.Album)
		t.AlbumArtURI = resolve(base, didl.First(item.AlbumArtURI))
	}
	return t
}

func resolve(base *url.URL, ref string) string {
	if ref == "" || base == nil {
		return ref
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return base.ResolveReference(u).String()
}

// parseDuration parses a UPnP duration such as "0:04:58" or "1:02:03.500",
// returning 0 for values like "NOT_IMPLEMENTED" or "".
func parseDuration(s string) time.Duration {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	sec, err3 := strconv.ParseFloat(parts[2], 64)
	if err1 != nil || err2 != nil || err3 != nil || h < 0 || m < 0 || sec < 0 {
		return 0
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec*float64(time.Second))
}

// ParsePosition parses a position in a track as people write it: a clock
// time such as "1:30" or "1:02:03", or a duration such as "90s".
func ParsePosition(s string) (time.Duration, error) {
	if d, err := time.ParseDuration(s); err == nil && d >= 0 {
		return d, nil
	}
	invalid := fmt.Errorf("sonos: invalid position %q: want e.g. 1:30 or 90s", s)
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, invalid
	}
	var d time.Duration
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0, invalid
		}
		d = d*60 + time.Duration(n)
	}
	return d * time.Second, nil
}

// formatDuration formats d as H:MM:SS, the form Seek takes, dropping
// fractions of a second.
func formatDuration(d time.Duration) string {
	s := int(d / time.Second)
	return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
}
