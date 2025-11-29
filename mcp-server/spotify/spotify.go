package spotify

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

type SpotifyTrackInfo struct {
	URI      string `json:"uri"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	AlbumArt string `json:"album_art_url"`
}

type SpotifyClientInterface interface {
	Search(ctx context.Context, query string, searchType spotify.SearchType) (*spotify.SearchResult, error)
}

type SpotifyClient struct {
	client *spotify.Client
}

func NewSpotifyClient(ctx context.Context, clientID, clientSecret string) (*SpotifyClient, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Spotify client ID and secret must be provided.")
	}
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		return nil, err
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotify.New(httpClient)

	return &SpotifyClient{
		client: client,
	}, nil
}

func (sc *SpotifyClient) Search(ctx context.Context, query string, searchType spotify.SearchType) (*spotify.SearchResult, error) {
	return sc.client.Search(ctx, query, searchType)
}
