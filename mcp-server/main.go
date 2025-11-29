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
	loggingmiddleware "github.com/caglar10ur/sonos/mcp-server/middleware"
	"github.com/caglar10ur/sonos/mcp-server/sonoscontrol"
	"github.com/caglar10ur/sonos/mcp-server/spotify"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	transport := flag.String("transport", "", "Transport type for MCP server (http or stdio)")
	port := flag.String("port", "8888", "Port for the HTTP server")
	spotifyClientID := flag.String("spotify-client-id", os.Getenv("SPOTIFY_CLIENT_ID"), "Spotify client ID")
	spotifyClientSecret := flag.String("spotify-client-secret", os.Getenv("SPOTIFY_CLIENT_SECRET"), "Spotify client secret")
	searchTimeout := flag.Duration("search-timeout", 3*time.Second, "Timeout for Sonos device search")

	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create a new MCP server
	s := mcp.NewServer(&mcp.Implementation{Name: "Sonos", Version: "1.2.0"}, nil)

	// Add logging middleware
	s.AddReceivingMiddleware(loggingmiddleware.LoggingMiddleware())

	sonosController, err := sonoscontrol.NewSonosController()
	if err != nil {
		log.Fatalf("Creating a sonos controller failed: %v", err)
	}

	spotifyClient, err := spotify.NewSpotifyClient(ctx, *spotifyClientID, *spotifyClientSecret)
	if err != nil {
		log.Printf("Creating a spotify client failed: %v, spotify tools will be disabled", err)
		spotifyClient = nil
	}

	h := handlers.NewHandlers(sonosController, spotifyClient, *searchTimeout)

	registerTools(s, h, spotifyClient)

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

		<-ctx.Done()
		log.Println("Shutting down HTTP server...")

		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
		log.Println("HTTP server exited gracefully.")
	default:
		// Start the stdio server. This allows communication over standard input/output.
		transport := &mcp.StdioTransport{}
		session, err := s.Connect(ctx, transport, nil)
		if err != nil {
			log.Fatalf("Stdio Server error: %v", err)
		}
		session.Wait()
	}
}

func registerTools(s *mcp.Server, h *handlers.Handlers, spotifyClient *spotify.SpotifyClient) {
	// 1. Generic Tools (No specific params or 'any')
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_sonos_devices",
		Description: "List all Sonos devices on the network",
	}, h.ListSonosDevicesHandler)

	// 2. Tools using RoomNameParams
	roomTools := []struct {
		Name    string
		Desc    string
		Handler func(context.Context, *mcp.CallToolRequest, handlers.RoomNameParams) (*mcp.CallToolResult, any, error)
	}{
		{"get_now_playing", "Get the currently playing track on a Sonos device", h.GetNowPlayingHandler},
		{"play", "Start playback on a Sonos device", h.PlayHandler},
		{"stop", "Stop playback on a Sonos device", h.StopHandler},
		{"pause", "Pause playback on a Sonos device", h.PauseHandler},
		{"next", "Play the next track on a Sonos device", h.NextHandler},
		{"previous", "Play the previous track on a Sonos device", h.PreviousHandler},
		{"get_volume", "Get the current volume of a Sonos device", h.GetVolumeHandler},
		{"list_queue", "List the songs in the device's queue", h.ListQueueHandler},
		{"get_position_info", "Get the position information of the currently playing song", h.GetPositionInfoHandler},
		{"mute", "Mute a Sonos device", h.MuteHandler},
		{"unmute", "Unmute a Sonos device", h.UnmuteHandler},
		{"get_mute_status", "Get the mute status of a Sonos device", h.GetMuteStatusHandler},
		{"get_audio_input_attributes", "Get the name and icon of the audio input", h.GetAudioInputAttributesHandler},
		{"get_line_in_level", "Get the current left and right line-in levels", h.GetLineInLevelHandler},
		{"get_zone_info", "Get detailed information about a Sonos device", h.GetZoneInfoHandler},
		{"get_uuid", "Get the UUID of a Sonos device", h.GetUUIDHandler},
		{"switch_to_line_in", "Switch playback to the line-in input", h.SwitchToLineInHandler},
		{"switch_to_queue", "Switch playback to the queue", h.SwitchToQueueHandler},
		{"list_sonos_groups", "List all Sonos zone groups on the network", h.ListGroupsHandler},
		{"get_zone_group_attributes", "Get the zone group attributes for a given room", h.GetZoneGroupAttributesHandler},
		{"get_group_volume", "Get the current volume of a Sonos group", h.GetGroupVolumeHandler},
		{"get_media_info", "Get the current media information on a Sonos device", h.GetMediaInfoHandler},
	}

	for _, t := range roomTools {
		mcp.AddTool(s, &mcp.Tool{Name: t.Name, Description: t.Desc}, t.Handler)
	}

	// 3. Tools with specific params (Explicit registration)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_volume",
		Description: "Set the volume of a Sonos device",
	}, h.SetVolumeHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_line_in_level",
		Description: "Set the left and right line-in levels",
	}, h.SetLineInLevelHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "select_audio",
		Description: "Select an audio input by its ObjectID",
	}, h.SelectAudioHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_group_member",
		Description: "Add a member to a Sonos group",
	}, h.AddGroupMemberHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "remove_group_member",
		Description: "Remove a member from a Sonos group",
	}, h.RemoveGroupMemberHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_group_volume",
		Description: "Set the volume of a Sonos group",
	}, h.SetGroupVolumeHandler)

	if spotifyClient != nil {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "search_spotify",
			Description: "Search for a track, album, or artist on Spotify and returns its URI",
		}, h.SearchSpotifyHandler)

		mcp.AddTool(s, &mcp.Tool{
			Name:        "play_spotify_uri",
			Description: "Play a Spotify URI on a Sonos device",
		}, h.PlaySpotifyURIHandler)
	}
}
