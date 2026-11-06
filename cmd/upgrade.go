package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	goversion "github.com/hashicorp/go-version"
	"github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/ui"
	"github.com/spf13/cobra"
)

type Release struct {
	URL             string    `json:"url"`
	AssetsURL       string    `json:"assets_url"`
	UploadURL       string    `json:"upload_url"`
	HTMLURL         string    `json:"html_url"`
	ID              int64     `json:"id"`
	Author          User      `json:"author"`
	NodeID          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Immutable       bool      `json:"immutable"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Assets          []Asset   `json:"assets"`
	TarballURL      string    `json:"tarball_url"`
	ZipballURL      string    `json:"zipball_url"`
	Body            string    `json:"body"`
	MentionsCount   int       `json:"mentions_count"`
}

type User struct {
	Login             string `json:"login"`
	ID                int64  `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	UserViewType      string `json:"user_view_type"`
	SiteAdmin         bool   `json:"site_admin"`
}

type Asset struct {
	URL                string    `json:"url"`
	ID                 int64     `json:"id"`
	NodeID             string    `json:"node_id"`
	Name               string    `json:"name"`
	Label              string    `json:"label"`
	Uploader           User      `json:"uploader"`
	ContentType        string    `json:"content_type"`
	State              string    `json:"state"`
	Size               int64     `json:"size"`
	Digest             string    `json:"digest"`
	DownloadCount      int       `json:"download_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	BrowserDownloadURL string    `json:"browser_download_url"`
}

const (
	REPOURL          = "https://api.github.com/repos/kumneger0/cligram/releases/latest"
	installScriptURL = "https://raw.githubusercontent.com/Kumneger0/cligram/main/install.sh"
)

type UpdateState struct {
	mu            sync.Mutex
	checked       bool
	available     bool
	latestVersion string
}

var GlobalUpdateState UpdateState

func CheckUpdateAsync(currentVersion string) {
	if currentVersion == "" {
		return
	}
	go func() {
		info := GetNewVersionInfo(currentVersion)
		GlobalUpdateState.mu.Lock()
		GlobalUpdateState.checked = true
		GlobalUpdateState.available = info.IsUpdateAvailable
		if info.IsUpdateAvailable {
			GlobalUpdateState.latestVersion = info.LatestRelease.TagName
		}
		GlobalUpdateState.mu.Unlock()

		NotifyNewVersionIfAvailable()
	}()
}

func NotifyNewVersionIfAvailable() {
	GlobalUpdateState.mu.Lock()
	defer GlobalUpdateState.mu.Unlock()

	time.Sleep(3 * time.Second) // Wait for 2 seconds before sending the notification
	if GlobalUpdateState.available && GlobalUpdateState.latestVersion != "" {
		if config.GetIgnoredUpdateVersion() != GlobalUpdateState.latestVersion {
			if Program != nil {
				Program.Send(ui.NewVersionMsg{Version: GlobalUpdateState.latestVersion})
			}
		}
	}
}

func PromptExitUpgradeIfAvailable() {
	GlobalUpdateState.mu.Lock()
	available := GlobalUpdateState.available
	latestVersion := GlobalUpdateState.latestVersion
	GlobalUpdateState.mu.Unlock()

	if !available || latestVersion == "" {
		return
	}

	ignoredVersion := config.GetIgnoredUpdateVersion()
	if ignoredVersion == latestVersion {
		return
	}

	fmt.Printf("\nA new version (%s) of cligram is available. Some features may not work as expected—upgrade is recommended.\n", latestVersion)
	fmt.Print("Would you like to upgrade now? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(response)

	if strings.EqualFold(response, "y") || strings.EqualFold(response, "yes") {
		binDir := GetActiveBinDir()
		fmt.Printf("Upgrading to %s...\n", latestVersion)
		if err := RunUpgrade(binDir); err != nil {
			fmt.Printf("Upgrade failed: %v\n", err)
			return
		}
		_ = config.ClearIgnoredUpdateVersion()
		fmt.Println("cligram upgraded successfully!")
	} else {
		if err := config.SetIgnoredUpdateVersion(latestVersion); err != nil {
			slog.Error("failed to save ignored update version", "error", err)
		} else {
			fmt.Printf("Upgrade skipped. You won't be reminded again for %s. (Run 'cligram upgrade' anytime to update.)\n", latestVersion)
		}
	}
}

func upgradeCligram(currentVersion string) *cobra.Command {
	return &cobra.Command{
		Use:          "upgrade",
		Short:        "upgrade to latest version",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
				return fmt.Errorf("upgrade is only supported on Linux and macOS")
			}

			newVersionInfo := GetNewVersionInfo(currentVersion)
			if !newVersionInfo.IsUpdateAvailable {
				fmt.Println("Already on the latest version")
				return nil
			}

			latestRelease := newVersionInfo.LatestRelease
			fmt.Printf("Found latest version %s (current: %s)\n", latestRelease.TagName, currentVersion)

			binDir := GetActiveBinDir()
			if err := RunUpgrade(binDir); err != nil {
				return fmt.Errorf("upgrade failed: %w", err)
			}

			_ = config.ClearIgnoredUpdateVersion()
			fmt.Println("Successfully upgraded cligram!")
			return nil
		},
	}
}

func GetActiveBinDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(execPath)
}

func RunUpgrade(binDir string) error {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return fmt.Errorf("upgrade is only supported on Linux and macOS")
	}

	if _, err := exec.LookPath("bash"); err != nil {
		return fmt.Errorf("bash is required to run the upgrade script: %w", err)
	}
	if _, err := exec.LookPath("curl"); err != nil {
		return fmt.Errorf("curl is required to download the upgrade script: %w", err)
	}

	cmdStr := fmt.Sprintf("curl -fsSL %s | bash", installScriptURL)
	cmd := exec.Command("bash", "-c", cmdStr)
	if binDir != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("BINDIR=%s", binDir))
	} else {
		cmd.Env = os.Environ()
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

type NewVersionInfo struct {
	IsUpdateAvailable bool
	LatestRelease     Release
}

func IsNewerVersion(currentVersion, latestVersion string) (bool, error) {
	if currentVersion == "" || latestVersion == "" {
		return false, nil
	}
	cleanCurrent := strings.TrimPrefix(currentVersion, "v")
	v1, err := goversion.NewVersion(cleanCurrent)
	if err != nil {
		return false, err
	}
	cleanLatest := strings.TrimPrefix(latestVersion, "v")
	v2, err := goversion.NewVersion(cleanLatest)
	if err != nil {
		return false, err
	}
	return v2.GreaterThan(v1), nil
}

func GetNewVersionInfo(installedVersion string) NewVersionInfo {
	if installedVersion == "" {
		return NewVersionInfo{
			IsUpdateAvailable: false,
			LatestRelease:     Release{},
		}
	}

	response, err := http.Get(REPOURL)
	if err != nil {
		slog.Error("failed to fetch latest release", "url", REPOURL, "error", err)
		return NewVersionInfo{IsUpdateAvailable: false, LatestRelease: Release{}}
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		slog.Error("unexpected status code fetching latest release", "status", response.StatusCode)
		return NewVersionInfo{IsUpdateAvailable: false, LatestRelease: Release{}}
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("failed to read latest release response", "error", err)
		return NewVersionInfo{IsUpdateAvailable: false, LatestRelease: Release{}}
	}
	var latestRelease Release
	err = json.Unmarshal(data, &latestRelease)
	if err != nil {
		slog.Error("failed to decode latest release response", "error", err)
		return NewVersionInfo{IsUpdateAvailable: false, LatestRelease: Release{}}
	}

	newer, err := IsNewerVersion(installedVersion, latestRelease.TagName)
	if err != nil {
		slog.Error("failed to compare versions", "installed", installedVersion, "latest", latestRelease.TagName, "error", err)
		return NewVersionInfo{
			IsUpdateAvailable: false,
			LatestRelease:     latestRelease,
		}
	}

	return NewVersionInfo{
		IsUpdateAvailable: newer,
		LatestRelease:     latestRelease,
	}
}
