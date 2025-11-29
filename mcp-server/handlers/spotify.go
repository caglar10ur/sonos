package handlers

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"

	"github.com/caglar10ur/sonos"
	"github.com/caglar10ur/sonos/didl"
	"github.com/caglar10ur/sonos/mcp-server/spotify"
	avt "github.com/caglar10ur/sonos/services/AVTransport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	spot "github.com/zmb3/spotify/v2"
)

func (h *Handlers) SearchSpotifyHandler(ctx context.Context, req *mcp.CallToolRequest, params SearchSpotifyParams) (*mcp.CallToolResult, any, error) {
	var searchRequestType spot.SearchType
	switch params.SearchType {
	case "track":
		searchRequestType = spot.SearchTypeTrack
	case "album":
		searchRequestType = spot.SearchTypeAlbum
	case "artist":
		searchRequestType = spot.SearchTypeArtist
	case "playlist":
		searchRequestType = spot.SearchTypePlaylist
	default:
		return handleGenericError(fmt.Errorf("invalid search type: %s", params.SearchType)), nil, nil
	}

	results, err := h.spotifyClient.Search(ctx, params.Query, searchRequestType)
	if err != nil {
		return handleGenericError(err), nil, nil
	}

	var result string

	switch params.SearchType {
	case "track":
		if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
			track := results.Tracks.Tracks[0]
			var albumArtURL string
			if len(track.Album.Images) > 0 {
				albumArtURL = track.Album.Images[0].URL
			}
			trackInfo := spotify.SpotifyTrackInfo{
				URI:      string(track.URI),
				Title:    track.Name,
				Artist:   track.Artists[0].Name,
				Album:    track.Album.Name,
				AlbumArt: albumArtURL,
			}
			jsonBytes, err := json.Marshal(trackInfo)
			if err != nil {
				return handleGenericError(fmt.Errorf("failed to marshal track info: %v", err)), nil, nil
			}
			result = string(jsonBytes)
		}
	case "album":
		if results.Albums != nil && len(results.Albums.Albums) > 0 {
			result = string(results.Albums.Albums[0].URI)
		}
	case "artist":
		if results.Artists != nil && len(results.Artists.Artists) > 0 {
			result = string(results.Artists.Artists[0].URI)
		}
	case "playlist":
		if results.Playlists != nil && len(results.Playlists.Playlists) > 0 {
			result = string(results.Playlists.Playlists[0].URI)
		}
	}

	if result == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "no results found"},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

func (h *Handlers) PlaySpotifyURIHandler(ctx context.Context, req *mcp.CallToolRequest, params PlaySpotifyURIParams) (*mcp.CallToolResult, any, error) {
	var trackInfo spotify.SpotifyTrackInfo
	err := json.Unmarshal([]byte(params.TrackInfoJSON), &trackInfo)
	if err != nil {
		return handleGenericError(fmt.Errorf("failed to unmarshal track info: %v", err)), nil, nil
	}

	return h.withRoom(ctx, params.RoomName, func(zp *sonos.ZonePlayer) (*mcp.CallToolResult, any, error) {
		// Parse Spotify URI
		parts := strings.Split(trackInfo.URI, ":")
		if len(parts) != 3 || parts[0] != "spotify" {
			return handleGenericError(fmt.Errorf("invalid Spotify URI format")), nil, nil
		}

		typeStr := parts[1]
		id := parts[2]

		var resURI string
		var itemid string

		switch typeStr {
		case "track":
			resURI = fmt.Sprintf("x-sonos-spotify:spotify%%3atrack%%3a%s?sid=12&flags=0&sn=2", id)
			itemid = fmt.Sprintf("00032020%s", url.QueryEscape(trackInfo.URI))
		case "album":
			resURI = fmt.Sprintf("x-rincon-cpcontainer:1004206cspotify%%3aalbum%%3a%s?sid=12&flags=0&sn=2", id)
			itemid = fmt.Sprintf("1004206cspotify%%3aalbum%%3a%s", id)
		case "artist":
			// TODO
			resURI = fmt.Sprintf("x-rincon-cpcontainer:1005206cspotify%%3aartist%%3a%s?sid=12&flags=0&sn=2", id)
		case "playlist":
			// TODO
			resURI = fmt.Sprintf("x-rincon-cpcontainer:1006206cspotify:playlist:id?sid=12&flags=0&sn=2", id)
			//magic = "1006206"
		default:
			return handleGenericError(fmt.Errorf("unsupported Spotify URI type: %s", typeStr)), nil, nil
		}

		// Populate the structs with data
		didl := DIDLLite{
			Dc:    "http://purl.org/dc/elements/1.1/",
			Upnp:  "urn:schemas-upnp-org:metadata-1-0/upnp/",
			R:     "urn:schemas-rinconnetworks-com:metadata-1-0/",
			Xmlns: "urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/",
			Item: didl.Item{
				ID:          itemid,
				Restricted:  true,
				Class:       []didl.Class{{Value: "object.item.audioItem.musicTrack"}},
				Title:       []didl.Title{{Value: trackInfo.Title}},
				Creator:     []didl.Creator{{Value: trackInfo.Artist}},
				Album:       []didl.Album{{Value: trackInfo.Album}},
				AlbumArtURI: []didl.AlbumArtURI{{Value: trackInfo.AlbumArt}},
				AlbumArtist: []didl.AlbumArtist{{Value: trackInfo.Artist}},
				Desc: []didl.Desc{{
					ID:        "cdudn",
					NameSpace: "urn:schemas-rinconnetworks-com:metadata-1-0/",
					Value:     "SA_RINCON3079_X_#Svc3079-0-Token",
				}},
			},
		}

		// Marshal the struct to XML
		output, err := xml.Marshal(didl)
		if err != nil {
			return handleGenericError(fmt.Errorf("failed to marshall xml Spotify URI type: %s", typeStr)), nil, nil
		}

		if false {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("this is what I generated: %s", string(output))},
				},
			}, nil, nil
		}
		_, err = zp.AVTransport.AddURIToQueue(&avt.AddURIToQueueArgs{
			InstanceID:                      0,
			EnqueuedURI:                     resURI,
			EnqueuedURIMetaData:             string(output),
			DesiredFirstTrackNumberEnqueued: 1,
			EnqueueAsNext:                   true,
		})

		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}

		err = zp.Play()
		if err != nil {
			return handleError(err, params.RoomName), nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Playing Spotify URI %s in %s", trackInfo.URI, params.RoomName)},
			},
		}, nil, nil
	})
}
