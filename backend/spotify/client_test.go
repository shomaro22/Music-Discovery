package spotify

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientSearchArtistsUsesClientCredentialsAndCachesToken(t *testing.T) {
	var tokenCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/token":
			atomic.AddInt32(&tokenCalls, 1)
			want := "Basic " + base64.StdEncoding.EncodeToString([]byte("client:secret"))
			if r.Header.Get("Authorization") != want {
				t.Fatalf("Authorization = %q, want %q", r.Header.Get("Authorization"), want)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
		case "/v1/search":
			if r.Header.Get("Authorization") != "Bearer token" {
				t.Fatalf("bearer token was not sent")
			}
			if r.URL.Query().Get("type") != "artist" || r.URL.Query().Get("q") != "Muse" {
				t.Fatalf("unexpected search query: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"artists":{"items":[{"id":"artist-id","name":"Muse","genres":["rock"],"images":[],"popularity":80,"external_urls":{"spotify":"https://open.spotify.com/artist/artist-id"}}],"total":1}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	value, err := NewClient(Config{
		ClientID: "client", ClientSecret: "secret",
		AccountsURL: server.URL, APIURL: server.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	for i := 0; i < 2; i++ {
		result, err := value.SearchArtists(context.Background(), "Muse", 10)
		if err != nil {
			t.Fatalf("SearchArtists: %v", err)
		}
		if len(result.Items) != 1 || result.Items[0].ID != "artist-id" {
			t.Fatalf("unexpected result: %#v", result)
		}
	}
	if got := atomic.LoadInt32(&tokenCalls); got != 1 {
		t.Fatalf("token calls = %d, want 1", got)
	}
}

func TestNewClientWithoutCredentialsReturnsUnavailableClient(t *testing.T) {
	value, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := value.SearchArtists(context.Background(), "Muse", 10); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("SearchArtists error = %v, want ErrUnavailable", err)
	}
	if _, err := value.SearchTracks(context.Background(), "Muse", 10); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("SearchTracks error = %v, want ErrUnavailable", err)
	}
}

func TestClientSearchTracksAddsConfiguredMarket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
		case "/v1/search":
			if got := r.URL.Query().Get("market"); got != "JP" {
				t.Fatalf("market=%q want=JP", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":0}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	value, err := NewClient(Config{
		ClientID: "client", ClientSecret: "secret",
		AccountsURL: server.URL, APIURL: server.URL + "/v1", Market: "jp",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := value.SearchTracks(context.Background(), `genre:"pop"`, 10); err != nil {
		t.Fatalf("SearchTracks: %v", err)
	}
}
