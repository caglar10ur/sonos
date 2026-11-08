package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caglar10ur/sonos/mcp/server/handlers"
	loggingmiddleware "github.com/caglar10ur/sonos/mcp/server/middleware"
	"github.com/caglar10ur/sonos/mcp/server/sonoscontrol"
	"github.com/caglar10ur/sonos/mcp/server/spotify"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	Transport           string
	Port                string
	SpotifyClientID     string
	SpotifyClientSecret string
	SearchTimeout       time.Duration
}

func main() {
	cfg := parseConfig()

	// Initialize structured logging
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create a new MCP server
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "Sonos",
		Version: "1.2.0",
	}, nil)

	// Add logging middleware
	s.AddReceivingMiddleware(loggingmiddleware.LoggingMiddleware())

	sonosController, err := sonoscontrol.NewSonosController()
	if err != nil {
		slog.Error("Creating a sonos controller failed", "error", err)
		os.Exit(1)
	}

	spotifyClient, err := spotify.NewSpotifyClient(ctx, cfg.SpotifyClientID, cfg.SpotifyClientSecret)
	if err != nil {
		slog.Warn("Creating a spotify client failed, spotify tools will be disabled", "error", err)
		spotifyClient = nil
	}

	h := handlers.NewHandlers(sonosController, spotifyClient, cfg.SearchTimeout)
	h.RegisterTools(s)

	// Choose transport based on the command-line argument
	if err := runServer(ctx, s, cfg); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func parseConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.Transport, "transport", "", "Transport type for MCP server (http or stdio)")
	flag.StringVar(&cfg.Port, "port", "8888", "Port for the HTTP server")
	flag.StringVar(&cfg.SpotifyClientID, "spotify-client-id", os.Getenv("SPOTIFY_CLIENT_ID"), "Spotify client ID")
	flag.StringVar(&cfg.SpotifyClientSecret, "spotify-client-secret", os.Getenv("SPOTIFY_CLIENT_SECRET"), "Spotify client secret")
	flag.DurationVar(&cfg.SearchTimeout, "search-timeout", 6*time.Second, "Timeout for Sonos device search and room lookup")

	flag.Parse()
	return cfg
}

func runServer(ctx context.Context, s *mcp.Server, cfg Config) error {
	switch cfg.Transport {
	case "http":
		return runHTTPServer(ctx, s, cfg.Port)
	default:
		return runStdioServer(ctx, s)
	}
}

func runStdioServer(ctx context.Context, s *mcp.Server) error {
	slog.Info("Starting stdio server")
	transport := &mcp.StdioTransport{}
	session, err := s.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	session.Wait()
	return nil
}

func runHTTPServer(ctx context.Context, s *mcp.Server, port string) error {
	server := &http.Server{
		Addr: ":" + port,
		// Reject cross-site requests from browsers, so a web page can't drive
		// the speakers through this server. MCP clients aren't affected.
		Handler: http.NewCrossOriginProtection().Handler(
			mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
				return s
			}, nil),
		),
	}

	errChan := make(chan error, 1)
	go func() {
		slog.Info("Starting HTTP server", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		slog.Info("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
