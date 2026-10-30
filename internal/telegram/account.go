package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram/client"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

type DirInfo struct {
	Name    string
	ModTime int64
}

func GetCligramDir() *string {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error(err.Error())
		return nil
	}
	sessionDir := filepath.Join(userHomeDir, ".cligram")
	return &sessionDir
}

func GetAccountPaths() []DirInfo {
	cligramDir := GetCligramDir()
	if cligramDir == nil {
		return []DirInfo{}
	}
	dirEntry, err := os.ReadDir(*cligramDir)
	if err != nil {
		slog.Error(err.Error())
		return []DirInfo{}
	}

	var dirs []DirInfo
	for _, dir := range dirEntry {
		if dir.IsDir() {
			info, err := dir.Info()
			if err != nil {
				slog.Error(err.Error())
				continue
			}
			dirs = append(dirs, DirInfo{
				Name:    dir.Name(),
				ModTime: info.ModTime().Unix(),
			})
		}
	}

	slices.SortFunc(dirs, func(a, b DirInfo) int {
		if a.ModTime < b.ModTime {
			return -1
		}
		if a.ModTime > b.ModTime {
			return 1
		}
		return 0
	})

	return dirs
}

func GenerateDir() string {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error(err.Error())
		return "account1"
	}
	sessionDir := filepath.Join(userHomeDir, ".cligram")

	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return "account1"
	}

	dirEntry, err := os.ReadDir(sessionDir)
	if err != nil {
		slog.Error(err.Error())
		return "account1"
	}

	re := regexp.MustCompile(`^account(\d+)$`)
	maxID := 0

	for _, dir := range dirEntry {
		if dir.IsDir() {
			matches := re.FindStringSubmatch(dir.Name())
			if len(matches) == 2 {
				id, err := strconv.Atoi(matches[1])
				if err == nil && id > maxID {
					maxID = id
				}
			}
		}
	}
	return fmt.Sprintf("account%d", maxID+1)
}

func SaveAccountMetadata(account string, info types.AccountsOnDeviceInfo) error {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	accDir := filepath.Join(userHomeDir, ".cligram", account)
	if err := os.MkdirAll(accDir, 0700); err != nil {
		return err
	}
	metaPath := filepath.Join(accDir, "account.json")
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, data, 0600)
}

func LoadAccountMetadata(account string) (*types.AccountsOnDeviceInfo, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	metaPath := filepath.Join(userHomeDir, ".cligram", account, "account.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}
	var info types.AccountsOnDeviceInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func GetAccountDirsOnThisDevice(telegramAPIID, telegramAPIHash string) []types.AccountsOnDeviceInfo {
	dirs := GetAccountPaths()
	if len(dirs) == 0 {
		return []types.AccountsOnDeviceInfo{}
	}

	var accountsOnThisDevice []types.AccountsOnDeviceInfo
	var missingDirs []DirInfo

	for _, d := range dirs {
		if info, err := LoadAccountMetadata(d.Name); err == nil && info != nil && info.AccountName != "" {
			info.ModTime = d.ModTime
			info.Path = d.Name
			accountsOnThisDevice = append(accountsOnThisDevice, *info)
		} else {
			missingDirs = append(missingDirs, d)
		}
	}

	if len(missingDirs) > 0 && telegramAPIID != "" && telegramAPIHash != "" {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		originalCligram := client.Cligram
		originalUpdateChannel := client.GetGlobalUpdateChannel()
		defer func() {
			client.Cligram = originalCligram
			client.SetGlobalUpdateChannel(originalUpdateChannel)
		}()

		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, d := range missingDirs {
			wg.Add(1)
			go func(d DirInfo) {
				defer wg.Done()

				clientCtx, clientCancel := context.WithCancel(ctx)
				defer clientCancel()

				cligram, err := NewClient(clientCtx, nil, telegramAPIID, telegramAPIHash, d.Name)
				if err != nil {
					slog.Debug("failed to create client for account discovery", "path", d.Name, "error", err)
					return
				}

				err = cligram.Client.Run(clientCtx, func(cCtx context.Context) error {
					self, err := cligram.Client.Self(cCtx)
					if err != nil {
						return err
					}

					accInfo := types.AccountsOnDeviceInfo{
						AccountName: self.FirstName,
						PhoneNumber: self.Phone,
						Path:        d.Name,
						ModTime:     d.ModTime,
					}

					_ = SaveAccountMetadata(d.Name, accInfo)

					mu.Lock()
					accountsOnThisDevice = append(accountsOnThisDevice, accInfo)
					mu.Unlock()

					clientCancel()
					return nil
				})

				if err != nil {
					slog.Debug("client run finished for account discovery", "path", d.Name, "error", err)
				}
			}(d)
		}
		wg.Wait()
	}

	slices.SortFunc(accountsOnThisDevice, func(a, b types.AccountsOnDeviceInfo) int {
		if a.ModTime < b.ModTime {
			return -1
		}
		if a.ModTime > b.ModTime {
			return 1
		}
		return 0
	})

	return accountsOnThisDevice
}

func GetAccountsCmd(telegramAPIID, telegramAPIHash string) tea.Cmd {
	return func() tea.Msg {
		accs := GetAccountDirsOnThisDevice(telegramAPIID, telegramAPIHash)
		return types.AccountsLoadedMsg{
			Accounts: accs,
			Err:      nil,
		}
	}
}
