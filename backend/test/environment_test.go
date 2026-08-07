package testutil

import "testing"

func TestValidateTestDSNRequiresExactDatabaseName(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		ok   bool
	}{
		{name: "dedicated test database", dsn: "postgres://postgres:postgres@localhost:5432/music_discovery_test?sslmode=disable", ok: true},
		{name: "test only in username", dsn: "postgres://test_user:postgres@localhost:5432/music_discovery?sslmode=disable", ok: false},
		{name: "test only in password", dsn: "postgres://postgres:test_password@localhost:5432/music_discovery?sslmode=disable", ok: false},
		{name: "similar database name", dsn: "postgres://postgres:postgres@localhost:5432/music_discovery_test_backup?sslmode=disable", ok: false},
		{name: "keyword DSN", dsn: "host=localhost user=postgres password=postgres dbname=music_discovery_test sslmode=disable", ok: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTestDSN(tt.dsn)
			if tt.ok && err != nil {
				t.Fatalf("validateTestDSN() error=%v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("validateTestDSN() unexpectedly accepted unsafe DSN")
			}
		})
	}
}
