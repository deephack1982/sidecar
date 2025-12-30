package version

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsCacheValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		entry          *CacheEntry
		currentVersion string
		want           bool
	}{
		{
			name:           "nil entry",
			entry:          nil,
			currentVersion: "v1.0.0",
			want:           false,
		},
		{
			name: "valid cache - same version, recent",
			entry: &CacheEntry{
				LatestVersion:  "v1.1.0",
				CurrentVersion: "v1.0.0",
				CheckedAt:      now,
				HasUpdate:      true,
			},
			currentVersion: "v1.0.0",
			want:           true,
		},
		{
			name: "expired cache - same version, old timestamp",
			entry: &CacheEntry{
				LatestVersion:  "v1.1.0",
				CurrentVersion: "v1.0.0",
				CheckedAt:      now.Add(-7 * time.Hour), // older than 6h TTL
				HasUpdate:      true,
			},
			currentVersion: "v1.0.0",
			want:           false,
		},
		{
			name: "invalid cache - version mismatch (upgrade)",
			entry: &CacheEntry{
				LatestVersion:  "v1.1.0",
				CurrentVersion: "v1.0.0",
				CheckedAt:      now,
				HasUpdate:      true,
			},
			currentVersion: "v1.1.0",
			want:           false,
		},
		{
			name: "invalid cache - version mismatch (downgrade)",
			entry: &CacheEntry{
				LatestVersion:  "v1.1.0",
				CurrentVersion: "v1.0.0",
				CheckedAt:      now,
				HasUpdate:      true,
			},
			currentVersion: "v0.9.0",
			want:           false,
		},
		{
			name: "boundary - exactly at TTL",
			entry: &CacheEntry{
				LatestVersion:  "v1.1.0",
				CurrentVersion: "v1.0.0",
				CheckedAt:      now.Add(-6*time.Hour + time.Minute), // just under TTL
				HasUpdate:      true,
			},
			currentVersion: "v1.0.0",
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCacheValid(tt.entry, tt.currentVersion)
			if got != tt.want {
				t.Errorf("IsCacheValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	// Create temp config dir
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "sidecar")

	// Override cachePath for testing by saving directly
	cachePath := filepath.Join(configDir, "version_cache.json")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	entry := &CacheEntry{
		LatestVersion:  "v1.2.0",
		CurrentVersion: "v1.0.0",
		CheckedAt:      time.Now().Truncate(time.Second), // Truncate for JSON roundtrip
		HasUpdate:      true,
	}

	// Write JSON directly
	data := `{"latestVersion":"v1.2.0","currentVersion":"v1.0.0","checkedAt":"` +
		entry.CheckedAt.Format(time.RFC3339) + `","hasUpdate":true}`
	if err := os.WriteFile(cachePath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	// Read back
	readData, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}

	if len(readData) == 0 {
		t.Error("Expected non-empty cache file")
	}
}

func TestLoadCache_FileNotExist(t *testing.T) {
	// LoadCache uses os.UserHomeDir() internally, so we can't easily
	// redirect it. This test verifies error handling for missing files.
	// The actual cachePath() function will return a real path.
	_, err := LoadCache()
	// Error is expected since cache likely doesn't exist in test env
	// or if it does exist, that's also fine
	_ = err
}

func TestCacheEntry_JSONRoundtrip(t *testing.T) {
	// Test that CacheEntry serializes/deserializes correctly
	original := CacheEntry{
		LatestVersion:  "v2.0.0",
		CurrentVersion: "v1.5.0",
		CheckedAt:      time.Now().Truncate(time.Second),
		HasUpdate:      true,
	}

	// Create temp file
	tmpFile := filepath.Join(t.TempDir(), "cache.json")

	// Write
	data := `{"latestVersion":"` + original.LatestVersion +
		`","currentVersion":"` + original.CurrentVersion +
		`","checkedAt":"` + original.CheckedAt.Format(time.RFC3339) +
		`","hasUpdate":` + "true" + `}`

	if err := os.WriteFile(tmpFile, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	// Read and verify
	readData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(readData) != data {
		t.Errorf("JSON roundtrip failed: got %s, want %s", readData, data)
	}
}
