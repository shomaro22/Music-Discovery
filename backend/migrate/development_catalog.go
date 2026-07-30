package migrate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"music-discovery/backend/entity"
)

type developmentCatalogSeed struct {
	Artist   entity.Artist
	Song     entity.Song
	GenreTag string
}

var developmentCatalogSeeds = []developmentCatalogSeed{
	{
		Artist:   entity.Artist{SpotifyArtistID: "6rs1KAoQnFalSqSU4LTh8g", Name: "back number", SpotifyURL: "https://open.spotify.com/artist/6rs1KAoQnFalSqSU4LTh8g", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "3QIAwtEEDOrv0g5NKCGrXZ", Name: "花束", AlbumName: "スーパースター", ImageURL: "https://i.scdn.co/image/ab67616d0000b2734951a4f5f9d719dfa4ef3d75", SpotifyURL: "https://open.spotify.com/track/3QIAwtEEDOrv0g5NKCGrXZ", DurationMs: 286053, Popularity: 50, ReleaseDate: time.Date(2011, 10, 26, 0, 0, 0, 0, time.UTC)},
		GenreTag: "genre-pop",
	},
	{
		Artist:   entity.Artist{SpotifyArtistID: "6YqdtpUutxodni6lUD4stM", Name: "WANIMA", SpotifyURL: "https://open.spotify.com/artist/6YqdtpUutxodni6lUD4stM", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "6FHgHDXiGnRfhrSok9xAz3", Name: "ともに", AlbumName: "Everybody!!", ImageURL: "https://i.scdn.co/image/ab67616d0000b273be77ec342dae727a2ce8c7d7", SpotifyURL: "https://open.spotify.com/track/6FHgHDXiGnRfhrSok9xAz3", DurationMs: 207368, Popularity: 50, ReleaseDate: time.Date(2018, 1, 17, 0, 0, 0, 0, time.UTC)},
		GenreTag: "genre-rock",
	},
	{
		Artist:   entity.Artist{SpotifyArtistID: "4Z8W4fKeB5YxbusRsdQVPb", Name: "Radiohead", SpotifyURL: "https://open.spotify.com/artist/4Z8W4fKeB5YxbusRsdQVPb", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "70LcF31zb1H0PyJoS1Sx1r", Name: "Creep", AlbumName: "Pablo Honey", ImageURL: "https://i.scdn.co/image/ab67616d0000b273ec548c00d3ac2f10be73366d", SpotifyURL: "https://open.spotify.com/track/70LcF31zb1H0PyJoS1Sx1r", DurationMs: 238640, Popularity: 50, ReleaseDate: time.Date(1993, 2, 22, 0, 0, 0, 0, time.UTC), Explicit: true},
		GenreTag: "genre-alternative",
	},
	{
		Artist:   entity.Artist{SpotifyArtistID: "3qm84nBOXUEQ2vnTfUTTFC", Name: "Guns N' Roses", SpotifyURL: "https://open.spotify.com/artist/3qm84nBOXUEQ2vnTfUTTFC", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "7snQQk1zcKl8gZ92AnueZW", Name: "Sweet Child O' Mine", AlbumName: "Appetite For Destruction", ImageURL: "https://i.scdn.co/image/ab67616d0000b27321ebf49b3292c3f0f575f0f5", SpotifyURL: "https://open.spotify.com/track/7snQQk1zcKl8gZ92AnueZW", DurationMs: 356066, Popularity: 50, ReleaseDate: time.Date(1987, 7, 21, 0, 0, 0, 0, time.UTC)},
		GenreTag: "genre-metal",
	},
	{
		Artist:   entity.Artist{SpotifyArtistID: "7dGJo4pcD2V6oG8kP0tJRR", Name: "Eminem", SpotifyURL: "https://open.spotify.com/artist/7dGJo4pcD2V6oG8kP0tJRR", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "1v7L65Lzy0j0vdpRjJewt1", Name: "Lose Yourself", AlbumName: "Just Lose It", ImageURL: "https://i.scdn.co/image/ab67616d0000b273b6ef2ebd34efb08cb76f6eec", SpotifyURL: "https://open.spotify.com/track/1v7L65Lzy0j0vdpRjJewt1", DurationMs: 322226, Popularity: 50, ReleaseDate: time.Date(2004, 1, 1, 0, 0, 0, 0, time.UTC), Explicit: true},
		GenreTag: "genre-hip-hop",
	},
	{
		Artist:   entity.Artist{SpotifyArtistID: "05YVYeV4HxYp5rrWalvuE1", Name: "Grover Washington, Jr.", SpotifyURL: "https://open.spotify.com/artist/05YVYeV4HxYp5rrWalvuE1", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "5fdNHVZHbWB1AaXk4RBGVD", Name: "Just the Two of Us", AlbumName: "Winelight", ImageURL: "https://i.scdn.co/image/ab67616d0000b273d8e0e065037b496212e554b7", SpotifyURL: "https://open.spotify.com/track/5fdNHVZHbWB1AaXk4RBGVD", DurationMs: 443773, Popularity: 50, ReleaseDate: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)},
		GenreTag: "genre-jazz",
	},
	{
		Artist:   entity.Artist{SpotifyArtistID: "1vCWHaC5f2uS3yhpwWbIA6", Name: "Avicii", SpotifyURL: "https://open.spotify.com/artist/1vCWHaC5f2uS3yhpwWbIA6", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "0ct6r3EGTcMLPtrXHDvVjc", Name: "The Nights", AlbumName: "The Days / Nights", ImageURL: "https://i.scdn.co/image/ab67616d0000b2730ae4f4d42e4a09f3a29f64ad", SpotifyURL: "https://open.spotify.com/track/0ct6r3EGTcMLPtrXHDvVjc", DurationMs: 176658, Popularity: 50, ReleaseDate: time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)},
		GenreTag: "genre-electronic",
	},
	{
		Artist:   entity.Artist{SpotifyArtistID: "7nzSoJISlVJsn7O0yTeMOB", Name: "Joe Hisaishi", SpotifyURL: "https://open.spotify.com/artist/7nzSoJISlVJsn7O0yTeMOB", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "15X1ksqhe8cY1NlEZMl7X8", Name: "海の見える街", AlbumName: "魔女の宅急便 サントラ音楽集", ImageURL: "https://i.scdn.co/image/ab67616d0000b273559ab67b95bf2a8c19fd151e", SpotifyURL: "https://open.spotify.com/track/15X1ksqhe8cY1NlEZMl7X8", DurationMs: 180333, Popularity: 50, ReleaseDate: time.Date(2004, 9, 29, 0, 0, 0, 0, time.UTC)},
		GenreTag: "genre-classical",
	},
	{
		Artist:   entity.Artist{SpotifyArtistID: "0blbVefuxOGltDBa00dspv", Name: "LiSA", SpotifyURL: "https://open.spotify.com/artist/0blbVefuxOGltDBa00dspv", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "0qMip0B2D4ePEjBJvAtYre", Name: "紅蓮華", AlbumName: "LEO-NiNE", ImageURL: "https://i.scdn.co/image/ab67616d0000b27364c8cf7bc530a05dd10e8efe", SpotifyURL: "https://open.spotify.com/track/0qMip0B2D4ePEjBJvAtYre", DurationMs: 237586, Popularity: 50, ReleaseDate: time.Date(2020, 10, 14, 0, 0, 0, 0, time.UTC)},
		GenreTag: "genre-anime",
	},
	{
		Artist:   entity.Artist{SpotifyArtistID: "3Nrfpe0tUJi4K4DXYWgMUX", Name: "BTS", SpotifyURL: "https://open.spotify.com/artist/3Nrfpe0tUJi4K4DXYWgMUX", Popularity: 50},
		Song:     entity.Song{SpotifyTrackID: "5QDLhrAOJJdNAmCTJ8xMyW", Name: "Dynamite", AlbumName: "BE", ImageURL: "https://i.scdn.co/image/ab67616d0000b273c07d5d2fdc02ae252fcd07e5", SpotifyURL: "https://open.spotify.com/track/5QDLhrAOJJdNAmCTJ8xMyW", DurationMs: 199053, Popularity: 50, ReleaseDate: time.Date(2020, 11, 20, 0, 0, 0, 0, time.UTC)},
		GenreTag: "genre-k-pop",
	},
}

var developmentMoodTags = []string{"mood-uplifting", "mood-calm", "mood-focus", "mood-sad", "mood-extraordinary"}
var developmentElementTags = []string{"element-lyrics", "element-vocal", "element-melody", "element-worldview", "element-rhythm"}
var developmentAtmosphereTags = []string{"atmosphere-bright", "atmosphere-night", "atmosphere-calm", "atmosphere-deep", "atmosphere-cinematic", "atmosphere-refreshing"}

func SeedDevelopmentCatalog(db *gorm.DB) error {
	return SeedDevelopmentCatalogForEnvironment(db, "")
}

// SeedDevelopmentCatalogForEnvironment creates deterministic test content.
// Production is rejected even when the startup flag is accidentally enabled.
func SeedDevelopmentCatalogForEnvironment(db *gorm.DB, appEnv string) error {
	if strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		return errors.New("development catalog seed is disabled in production")
	}
	if db == nil {
		return errors.New("database is required")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		tagIDs, err := developmentTagIDs(tx)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		for index, seed := range developmentCatalogSeeds {
			artist, err := upsertDevelopmentArtist(tx, seed.Artist, now)
			if err != nil {
				return err
			}
			song, err := upsertDevelopmentSong(tx, artist.ID, seed.Song, now)
			if err != nil {
				return err
			}
			for _, slug := range developmentSongTagSlugs(index, seed.GenreTag) {
				tagID, ok := tagIDs[slug]
				if !ok {
					return fmt.Errorf("development tag %s is missing", slug)
				}
				link := entity.SongTag{
					ID:        uuid.New(),
					SongID:    song.ID,
					TagID:     tagID,
					Weight:    1,
					CreatedAt: now,
					UpdatedAt: now,
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "song_id"}, {Name: "tag_id"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"weight", "updated_at",
					}),
				}).Create(&link).Error; err != nil {
					return fmt.Errorf("link development song %s to tag %s: %w", song.SpotifyTrackID, slug, err)
				}
			}
		}
		return nil
	})
}

func developmentTagIDs(tx *gorm.DB) (map[string]uuid.UUID, error) {
	seeds := systemTagSeeds()
	slugs := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		slugs = append(slugs, seed.Slug)
	}
	var tags []entity.Tag
	if err := tx.Where("slug IN ?", slugs).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("load development tags: %w", err)
	}
	if len(tags) != len(slugs) {
		return nil, fmt.Errorf("development tags are incomplete: got %d, want %d", len(tags), len(slugs))
	}
	result := make(map[string]uuid.UUID, len(tags))
	for _, tag := range tags {
		result[tag.Slug] = tag.ID
	}
	return result, nil
}

func developmentSongTagSlugs(index int, genre string) []string {
	result := []string{genre}
	for _, offset := range []int{0, 2, 4} {
		result = append(result, developmentMoodTags[(index+offset)%len(developmentMoodTags)])
		result = append(result, developmentElementTags[(index+offset)%len(developmentElementTags)])
		result = append(result, developmentAtmosphereTags[(index+offset)%len(developmentAtmosphereTags)])
	}
	return result
}

func upsertDevelopmentArtist(tx *gorm.DB, value entity.Artist, now time.Time) (*entity.Artist, error) {
	value.CreatedAt = now
	value.UpdatedAt = now
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "spotify_artist_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": value.Name, "image_url": value.ImageURL, "spotify_url": value.SpotifyURL,
			"popularity": value.Popularity, "deleted_at": nil, "updated_at": now,
		}),
	}).Create(&value).Error; err != nil {
		return nil, fmt.Errorf("upsert development artist %s: %w", value.SpotifyArtistID, err)
	}
	if err := tx.Unscoped().Where("spotify_artist_id = ?", value.SpotifyArtistID).First(&value).Error; err != nil {
		return nil, fmt.Errorf("reload development artist %s: %w", value.SpotifyArtistID, err)
	}
	return &value, nil
}

func upsertDevelopmentSong(tx *gorm.DB, artistID uuid.UUID, value entity.Song, now time.Time) (*entity.Song, error) {
	value.ArtistID = artistID
	value.CreatedAt = now
	value.UpdatedAt = now
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "spotify_track_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"artist_id": artistID, "name": value.Name, "album_name": value.AlbumName,
			"image_url": value.ImageURL, "spotify_url": value.SpotifyURL, "preview_url": value.PreviewURL,
			"duration_ms": value.DurationMs, "popularity": value.Popularity, "release_date": value.ReleaseDate,
			"explicit": value.Explicit, "deleted_at": nil, "updated_at": now,
		}),
	}).Create(&value).Error; err != nil {
		return nil, fmt.Errorf("upsert development song %s: %w", value.SpotifyTrackID, err)
	}
	if err := tx.Unscoped().Where("spotify_track_id = ?", value.SpotifyTrackID).First(&value).Error; err != nil {
		return nil, fmt.Errorf("reload development song %s: %w", value.SpotifyTrackID, err)
	}
	return &value, nil
}
