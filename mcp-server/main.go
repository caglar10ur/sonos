package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caglar10ur/sonos/mcp-server/handlers"
	"github.com/caglar10ur/sonos/mcp-server/sonoscontrol"
	"github.com/caglar10ur/sonos/mcp-server/spotify"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	flag.Parse()
}

func main() {
	transport := flag.String("transport", "", "Transport type for MCP server (http or stdio)")
	port := flag.String("port", "8888", "Port for the HTTP server")

	// Create a new MCP server
	s := mcp.NewServer(&mcp.Implementation{Name: "Sonos", Version: "1.1.0"}, nil)

	sonosController, err := sonoscontrol.NewSonosController()
	if err != nil {
		log.Fatalf("Creating a sonos controller failed: %v", err)
	}

	spotifyClient, err := spotify.NewSpotifyClient(context.Background())
	if err != nil {
		log.Fatalf("Creating a spotify client failed: %v", err)
	}

	h := handlers.NewHandlers(sonosController, spotifyClient)

	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "list_sonos_devices",
			Description: "List all Sonos devices on the network",
		}, h.ListSonosDevicesHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_now_playing",
		Description: "Get the currently playing track on a Sonos device",
	}, h.GetNowPlayingHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "play",
		Description: "Start playback on a Sonos device",
	}, h.PlayHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "stop",
		Description: "Stop playback on a Sonos device",
	}, h.StopHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pause",
		Description: "Pause playback on a Sonos device",
	}, h.PauseHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "next",
		Description: "Play the next track on a Sonos device",
	}, h.NextHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "previous",
		Description: "Play the previous track on a Sonos device",
	}, h.PreviousHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_volume",
		Description: "Get the current volume of a Sonos device",
	}, h.GetVolumeHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_volume",
		Description: "Set the volume of a Sonos device",
	}, h.SetVolumeHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_queue",
		Description: "List the songs in the device's queue",
	}, h.ListQueueHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_position_info",
		Description: "Get the position information of the currently playing song",
	}, h.GetPositionInfoHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mute",
		Description: "Mute a Sonos device",
	}, h.MuteHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "unmute",
		Description: "Unmute a Sonos device",
	}, h.UnmuteHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_mute_status",
		Description: "Get the mute status of a Sonos device",
	}, h.GetMuteStatusHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_audio_input_attributes",
		Description: "Get the name and icon of the audio input",
	}, h.GetAudioInputAttributesHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_line_in_level",
		Description: "Get the current left and right line-in levels",
	}, h.GetLineInLevelHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_line_in_level",
		Description: "Set the left and right line-in levels",
	}, h.SetLineInLevelHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "select_audio",
		Description: "Select an audio input by its ObjectID",
	}, h.SelectAudioHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_zone_info",
		Description: "Get detailed information about a Sonos device",
	}, h.GetZoneInfoHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_uuid",
		Description: "Get the UUID of a Sonos device",
	}, h.GetUUIDHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "switch_to_line_in",
		Description: "Switch playback to the line-in input",
	}, h.SwitchToLineInHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "switch_to_queue",
		Description: "Switch playback to the queue",
	}, h.SwitchToQueueHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_sonos_groups",
		Description: "List all Sonos zone groups on the network",
	}, h.ListGroupsHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_zone_group_attributes",
		Description: "Get the zone group attributes for a given room",
	}, h.GetZoneGroupAttributesHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_group_member",
		Description: "Add a member to a Sonos group",
	}, h.AddGroupMemberHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "remove_group_member",
		Description: "Remove a member from a Sonos group",
	}, h.RemoveGroupMemberHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_group_volume",
		Description: "Get the current volume of a Sonos group",
	}, h.GetGroupVolumeHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_group_volume",
		Description: "Set the volume of a Sonos group",
	}, h.SetGroupVolumeHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_media_info",
		Description: "Get the current media information on a Sonos device",
	}, h.GetMediaInfoHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_spotify",
		Description: "Search for a track, album, or artist on Spotify and returns its URI",
	}, h.SearchSpotifyHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "play_spotify_uri",
		Description: "Play a Spotify URI on a Sonos device",
	}, h.PlaySpotifyURIHandler)

	// Choose transport based on the command-line argument
	switch *transport {
	case "http":
		server := &http.Server{
			Addr: ":" + *port,
			Handler: mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
				return s
			}, nil),
		}

		go func() {
			log.Printf("Starting HTTP server on port %s", *port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP Server error: %v", err)
			}
		}()

		// Wait for interrupt signal to gracefully shut down the server
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutting down HTTP server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
		log.Println("HTTP server exited gracefully.")
	default:
		// Start the stdio server. This allows communication over standard input/output.
		transport := &mcp.StdioTransport{}
		session, err := s.Connect(context.Background(), transport, nil)
		if err != nil {
			log.Fatalf("Stdio Server error: %v", err)
		}
		session.Wait()
	}
}
