package entity

// SpotifyCandidate is an Artist/Song pair imported from Spotify.
// TagSlugs contains system Tag slugs inferred from the Spotify genre search.
// It always contains the source Genre Tag and may contain conservative
// Mood/Element/Atmosphere mappings used by the recommendation contact rule.
type SpotifyCandidate struct {
	Artist   Artist
	Song     Song
	TagSlugs []string
}
