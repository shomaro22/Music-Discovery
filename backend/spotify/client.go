package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrUnavailable = errors.New("spotify unavailable")

type Config struct {
	ClientID     string
	ClientSecret string
	AccountsURL  string
	APIURL       string
	Timeout      time.Duration
}

type Image struct {
	URL string `json:"url"`
}

type Artist struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Genres       []string          `json:"genres"`
	Images       []Image           `json:"images"`
	Popularity   int               `json:"popularity"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type Album struct {
	Name        string  `json:"name"`
	ReleaseDate string  `json:"release_date"`
	Images      []Image `json:"images"`
}

type TrackArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Track struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Artists      []TrackArtist     `json:"artists"`
	Album        Album             `json:"album"`
	DurationMS   int               `json:"duration_ms"`
	Explicit     bool              `json:"explicit"`
	Popularity   int               `json:"popularity"`
	PreviewURL   string            `json:"preview_url"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type SearchArtistsResult struct {
	Items []Artist `json:"items"`
	Total int      `json:"total"`
}

type SearchTracksResult struct {
	Items []Track `json:"items"`
	Total int     `json:"total"`
}

type IClient interface {
	SearchArtists(context.Context, string, int) (SearchArtistsResult, error)
	SearchTracks(context.Context, string, int) (SearchTracksResult, error)
}

type unavailableClient struct{}

func (unavailableClient) SearchArtists(context.Context, string, int) (SearchArtistsResult, error) {
	return SearchArtistsResult{}, ErrUnavailable
}

func (unavailableClient) SearchTracks(context.Context, string, int) (SearchTracksResult, error) {
	return SearchTracksResult{}, ErrUnavailable
}

type client struct {
	config    Config
	http      *http.Client
	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func NewClient(config Config) (IClient, error) {
	if strings.TrimSpace(config.ClientID) == "" || strings.TrimSpace(config.ClientSecret) == "" {
		return unavailableClient{}, nil
	}
	if config.AccountsURL == "" {
		config.AccountsURL = "https://accounts.spotify.com"
	}
	if config.APIURL == "" {
		config.APIURL = "https://api.spotify.com/v1"
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	return &client{config: config, http: &http.Client{Timeout: config.Timeout}}, nil
}

func (c *client) SearchArtists(ctx context.Context, query string, limit int) (SearchArtistsResult, error) {
	var response struct {
		Artists SearchArtistsResult `json:"artists"`
	}
	if err := c.search(ctx, query, "artist", limit, &response); err != nil {
		return SearchArtistsResult{}, err
	}
	return response.Artists, nil
}

func (c *client) SearchTracks(ctx context.Context, query string, limit int) (SearchTracksResult, error) {
	var response struct {
		Tracks SearchTracksResult `json:"tracks"`
	}
	if err := c.search(ctx, query, "track", limit, &response); err != nil {
		return SearchTracksResult{}, err
	}
	return response.Tracks, nil
}

func (c *client) search(ctx context.Context, query, kind string, limit int, target interface{}) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("empty query")
	}
	if limit < 1 || limit > 10 {
		return fmt.Errorf("invalid limit")
	}
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	endpoint, _ := url.Parse(strings.TrimRight(c.config.APIURL, "/") + "/search")
	values := endpoint.Query()
	values.Set("q", query)
	values.Set("type", kind)
	values.Set("limit", strconv.Itoa(limit))
	endpoint.RawQuery = values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("spotify search: %w", ErrUnavailable)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		c.invalidateToken()
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("spotify search status %d: %w", res.StatusCode, ErrUnavailable)
	}
	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		return fmt.Errorf("spotify search decode: %w", ErrUnavailable)
	}
	return nil
}

func (c *client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expiresAt) {
		return c.token, nil
	}
	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.config.AccountsURL, "/")+"/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	credential := base64.StdEncoding.EncodeToString([]byte(c.config.ClientID + ":" + c.config.ClientSecret))
	req.Header.Set("Authorization", "Basic "+credential)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("spotify token: %w", ErrUnavailable)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("spotify token status %d: %w", res.StatusCode, ErrUnavailable)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil || payload.AccessToken == "" {
		return "", fmt.Errorf("spotify token decode: %w", ErrUnavailable)
	}
	c.token = payload.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(payload.ExpiresIn)*time.Second - 30*time.Second)
	return c.token, nil
}

func (c *client) invalidateToken() {
	c.mu.Lock()
	c.token = ""
	c.expiresAt = time.Time{}
	c.mu.Unlock()
}
