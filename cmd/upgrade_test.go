package cmd

import (
	"testing"

	"github.com/kumneger0/cligram/internal/config"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current  string
		latest   string
		expected bool
		wantErr  bool
	}{
		{"v2.7.0", "v2.8.0", true, false},
		{"2.7.0", "2.8.0", true, false},
		{"v2.8.0", "v2.8.0", false, false},
		{"v2.8.1", "v2.8.0", false, false},
		{"v2.7.0", "v2.7.1", true, false},
		{"v2.7.0-SNAPSHOT", "v2.7.0", true, false},
		{"", "v2.8.0", false, false},
		{"v2.8.0", "", false, false},
		{"invalid", "v2.8.0", false, true},
		{"v2.8.0", "invalid", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			got, err := IsNewerVersion(tt.current, tt.latest)
			if (err != nil) != tt.wantErr {
				t.Fatalf("IsNewerVersion(%q, %q) error = %v, wantErr %v", tt.current, tt.latest, err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("IsNewerVersion(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.expected)
			}
		})
	}
}

func TestGetActiveBinDir(t *testing.T) {
	dir := GetActiveBinDir()
	if dir == "" {
		t.Errorf("expected non-empty active bin dir")
	}
}

func TestUpdateSuppression(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Set ignored version
	if err := config.SetIgnoredUpdateVersion("v2.8.0"); err != nil {
		t.Fatalf("SetIgnoredUpdateVersion failed: %v", err)
	}

	GlobalUpdateState.mu.Lock()
	GlobalUpdateState.available = true
	GlobalUpdateState.latestVersion = "v2.8.0"
	GlobalUpdateState.mu.Unlock()

	// NotifyNewVersionIfAvailable should safely do nothing because version is ignored
	NotifyNewVersionIfAvailable()

	// Clear ignored version
	if err := config.ClearIgnoredUpdateVersion(); err != nil {
		t.Fatalf("ClearIgnoredUpdateVersion failed: %v", err)
	}

	if v := config.GetIgnoredUpdateVersion(); v != "" {
		t.Errorf("expected empty ignored version, got %q", v)
	}
}
