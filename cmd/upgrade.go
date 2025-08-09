package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	goversion "github.com/hashicorp/go-version"
	pb "github.com/schollz/progressbar/v3"
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

type OsName string

const (
	DARWIN  OsName = "darwin"
	WINDOWS OsName = "windows"
	LINUX   OsName = "linux"
)

type LinuxDisto string

const (
	UBUNTU LinuxDisto = "ubuntu"
	DEBIAN LinuxDisto = "debian"
	ALPINE LinuxDisto = "alpine"
)

const (
	REPOURL = "https://api.github.com/repos/kumneger0/cligram/releases/latest"
)

func upgradeCligram(currentVersion string) *cobra.Command {
	fileExtensions := map[LinuxDisto]string{
		UBUNTU: "deb",
		DEBIAN: "deb",
		ALPINE: "apk",
	}

	return &cobra.Command{
		Use:          "upgrade",
		Short:        "up",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			var osName OsName
			var distroName string
			switch runtime.GOOS {
			case "windows":
				osName = WINDOWS
			case "darwin":
				osName = DARWIN
			case "linux":
				osName = LINUX
				var err error
				distroName, err = getDistroName()
				if err != nil {
					log.Fatal("oops we failed to get the distro you are using")
				}

			}
			if osName != LINUX {
				fmt.Println("upgrade command is only supported in linux")
				return
			}

			if !strings.EqualFold(distroName, string(DEBIAN)) &&
				!strings.EqualFold(distroName, string(UBUNTU)) &&
				!strings.EqualFold(distroName, string(ALPINE)) {
				fmt.Println("upgrade is only supported in debian, ubuntu and alpine linux")
				return
			}

			newVersionInfo := GetNewVersionInfo(currentVersion)
			if !newVersionInfo.IsUpdateAvailable {
				fmt.Println("Already latest version")
				return
			}
			latestRelease := newVersionInfo.LatestRelease

			var assetUrl string
			format := fileExtensions[LinuxDisto(strings.ToLower(distroName))]
			for _, v := range latestRelease.Assets {
				if strings.Contains(v.Name, runtime.GOARCH) && strings.Contains(v.Name, strings.ToLower(string(osName))) && strings.Contains(v.Name, format) {
					assetUrl = v.BrowserDownloadURL
				}
			}
			if assetUrl == "" {
				log.Fatal("No compatible package found for your system")
			}

			fmt.Println("Found latest version", latestRelease.TagName)
			userHomeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatal("failed to get user home dir")
			}
			cacheDir := filepath.Join(userHomeDir, ".cache")
			_, err = os.Stat(cacheDir)
			if os.IsNotExist(err) {
				if mkErr := os.Mkdir(cacheDir, 0o755); mkErr != nil {
					log.Fatal("failed to create cache dir:", mkErr)
				}
			}

			var fileName string

			if strings.EqualFold(distroName, string(DEBIAN)) || strings.EqualFold(distroName, string(UBUNTU)) {
				fileName = "cligram-latest-version.deb"
			}
			if strings.EqualFold(distroName, string(ALPINE)) {
				fileName = "cligram-latest-version.apk"
			}

			if fileName == "" {
				log.Fatal("Failed to upgrade")
			}

			filePathToWriteFile := filepath.Join(cacheDir, fileName)
			_ = os.Remove(filePathToWriteFile)
			err = downloadBinary(assetUrl, filePathToWriteFile)

			if err != nil {
				fmt.Println("Failed To downloand the binary", err.Error())
			}

			if strings.EqualFold(distroName, string(DEBIAN)) || strings.EqualFold(distroName, string(UBUNTU)) {
				installBinary("dpkg", "-i", filePathToWriteFile)
			}
			if strings.EqualFold(distroName, string(ALPINE)) {
				installBinary("apk", "add", "--allow-untrusted", filePathToWriteFile)
			}
		},
	}
}

type NewVersionInfo struct {
	IsUpdateAvailable bool
	LatestRelease     Release
}

func GetNewVersionInfo(installedVersion string) NewVersionInfo {
	if installedVersion == "" {
		return NewVersionInfo{
			IsUpdateAvailable: false,
			LatestRelease:     Release{},
		}
	}
	cleanVersion := strings.Replace(installedVersion, "v", "", 1)
	ver, err := goversion.NewVersion(cleanVersion)
	if err != nil {
		fmt.Println("Invalid installed version:", err)
		return NewVersionInfo{
			IsUpdateAvailable: false,
			LatestRelease:     Release{},
		}
	}
	response, err := http.Get(REPOURL)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal("Failed to Read Response Body")
	}
	var latestRelease Release
	err = json.Unmarshal(data, &latestRelease)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal("Failed to Unmarshal", err)
	}
	latestVersion, err := goversion.NewVersion(strings.Replace(latestRelease.TagName, "v", "", 1))
	if err != nil {
		log.Fatal("Failed to get the latest version")
	}

	if !latestVersion.GreaterThan(ver) {
		return NewVersionInfo{
			IsUpdateAvailable: false,
			LatestRelease:     latestRelease,
		}
	}
	return NewVersionInfo{
		IsUpdateAvailable: true,
		LatestRelease:     latestRelease,
	}
}

func getDistroName() (string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed to open /etc/os-release: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if after, ok := strings.CutPrefix(line, "ID="); ok {
			distroName := after
			return strings.Trim(distroName, `"`), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading /etc/os-release: %w", err)
	}

	return "", fmt.Errorf("distribution name not found in /etc/os-release")
}

func downloadBinary(url string, outputPath string) error {
	resp, err := http.Get(url)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	bar := pb.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)

	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	return err
}

func installBinary(installCommand ...string) {
	fmt.Println("Installing package requires sudo privileges...")
	fmt.Print("Continue? (y/N): ")
	var response string
	if _, scanErr := fmt.Scanln(&response); scanErr != nil {
		fmt.Println("Installation cancelled")
		return
	}
	if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
		fmt.Println("Installation cancelled")
		return
	}
	cmd := exec.Command("sudo", installCommand...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err.Error())
	}
}
