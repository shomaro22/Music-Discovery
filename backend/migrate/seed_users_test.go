package migrate

import (
	"strings"
	"testing"
)

func TestSeedDevelopmentUsersForEnvironmentRejectsProduction(t *testing.T) {
	err := SeedDevelopmentUsersForEnvironment(nil, " production ")
	if err == nil || !strings.Contains(err.Error(), "disabled in production") {
		t.Fatalf("err=%v", err)
	}
}
