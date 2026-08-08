package migrate

import (
	"strings"
	"testing"
)

func TestSystemTagSeedsFollowQuestionChoices(t *testing.T) {
	values := systemTagSeeds()
	if len(values) != 26 {
		t.Fatalf("tag seed count = %d, want 26", len(values))
	}

	names := map[string]bool{}
	slugs := map[string]bool{}
	categoryCounts := map[string]int{}
	for _, value := range values {
		if names[value.Name] {
			t.Fatalf("duplicated tag name: %s", value.Name)
		}
		if slugs[value.Slug] {
			t.Fatalf("duplicated tag slug: %s", value.Slug)
		}
		if strings.TrimSpace(value.Description) == "" {
			t.Fatalf("description is empty: %s", value.Slug)
		}
		names[value.Name] = true
		slugs[value.Slug] = true
		categoryCounts[value.Category]++
	}

	expected := map[string]int{"mood": 5, "feature": 5, "scene": 6, "genre": 10}
	for category, count := range expected {
		if categoryCounts[category] != count {
			t.Fatalf("%s count = %d, want %d", category, categoryCounts[category], count)
		}
	}
}

func TestCuratedSubgenreTagSeedsAreIndependentFromProfileTags(t *testing.T) {
	profileTags := systemTagSeeds()
	allTags := allTagSeeds()

	if len(profileTags) != 26 {
		t.Fatalf("profile tag seed count = %d, want 26", len(profileTags))
	}
	if len(allTags) != 36 {
		t.Fatalf("all tag seed count = %d, want 36", len(allTags))
	}

	names := map[string]bool{}
	slugs := map[string]bool{}
	for _, value := range allTags {
		if names[value.Name] {
			t.Fatalf("duplicated tag name: %s", value.Name)
		}
		if slugs[value.Slug] {
			t.Fatalf("duplicated tag slug: %s", value.Slug)
		}
		names[value.Name] = true
		slugs[value.Slug] = true
	}
}

func TestDevelopmentCatalogTagDistributionCoversEveryProfileTag(t *testing.T) {
	usage := map[string]int{}
	for index, seed := range developmentCatalogSeeds {
		seen := map[string]bool{}
		for _, slug := range developmentSongTagSlugs(index, seed.GenreTag) {
			if seen[slug] {
				t.Fatalf("song %d has duplicated tag %s", index, slug)
			}
			seen[slug] = true
			usage[slug]++
		}
		if len(seen) != 10 {
			t.Fatalf("song %d tag count = %d, want 10", index, len(seen))
		}
	}

	for _, seed := range systemTagSeeds() {
		if usage[seed.Slug] == 0 {
			t.Fatalf("tag is not assigned to development catalog: %s", seed.Slug)
		}
	}
}

func TestDevelopmentCatalogSeedRejectsProduction(t *testing.T) {
	err := SeedDevelopmentCatalogForEnvironment(nil, " production ")
	if err == nil || !strings.Contains(err.Error(), "disabled in production") {
		t.Fatalf("err = %v", err)
	}
}
