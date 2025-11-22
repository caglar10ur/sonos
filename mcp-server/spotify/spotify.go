package spotify

import (
	"context"
	"os"

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

type SpotifyClient struct {
	client *spotify.Client
}

func NewSpotifyClient(ctx context.Context) (*SpotifyClient, error) {
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
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
