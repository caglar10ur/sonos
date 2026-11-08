package didl

import (
	"strings"
	"testing"
)

func TestFirst(t *testing.T) {
	if got := First(nil); got != "" {
		t.Errorf("First(nil) = %q", got)
	}
	if got := First([]Text{{Value: "a"}, {Value: "b"}}); got != "a" {
		t.Errorf("First = %q, want a", got)
	}
}

func TestParseItem(t *testing.T) {
	l, err := Parse(`<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">
<item id="Q:0/1" parentID="Q:0" restricted="true"><dc:title>Karma Police</dc:title><dc:creator>Radiohead</dc:creator><upnp:album>OK Computer</upnp:album></item></DIDL-Lite>`)
	if err != nil {
		t.Fatal(err)
	}
	it := l.Item[0]
	if it.ID != "Q:0/1" || !it.Restricted || First(it.Title) != "Karma Police" || First(it.Creator) != "Radiohead" || First(it.Album) != "OK Computer" {
		t.Errorf("item = %+v", it)
	}
}

func TestEmptyDocuments(t *testing.T) {
	for _, n := range []int{-1, 0} {
		if got := EmptyDocuments(n); got != "" {
			t.Errorf("EmptyDocuments(%d) = %q, want empty", n, got)
		}
	}
	if got := EmptyDocuments(2); strings.Count(got, "<DIDL-Lite") != 2 {
		t.Errorf("EmptyDocuments(2) = %q", got)
	}
}
