package entity

// SpotifyCandidate is an Artist/Song pair imported from Spotify.
// Imported candidates intentionally have no Tag relations. They remain outside
// recommendation selection until a content administrator reviews and tags them.
type SpotifyCandidate struct {
	Artist Artist
	Song   Song
}
