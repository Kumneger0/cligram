package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	ErrNoGUIFilePicker = errors.New("No GUI file picker found.\nPlease install 'zenity' (GNOME) or 'kdialog' (KDE) to attach files.")

	lookPath = exec.LookPath
)

type FilePickedMsg struct {
	Path     string
	Err      error
	Canceled bool
}

func FindGUIFilePicker() (string, []string, error) {
	if path, err := lookPath("zenity"); err == nil && path != "" {
		return "zenity", []string{"--file-selection", "--title=Select File to Attach"}, nil
	}
	if path, err := lookPath("kdialog"); err == nil && path != "" {
		return "kdialog", []string{"--getopenfilename", "--title=Select File to Attach"}, nil
	}
	return "", nil, ErrNoGUIFilePicker
}

func PickFileCmd(picker string, args []string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(picker, args...)
		out, err := cmd.Output()
		if err != nil {
			return FilePickedMsg{Path: "", Err: err, Canceled: true}
		}
		selectedPath := strings.TrimSpace(string(out))
		if selectedPath == "" {
			return FilePickedMsg{Path: "", Canceled: true}
		}
		return FilePickedMsg{Path: selectedPath, Err: nil, Canceled: false}
	}
}

func TriggerFilePicker(m *Model) (Model, tea.Cmd) {
	picker, args, err := FindGUIFilePicker()
	if err != nil {
		m.IsModalVisible = true
		m.ModalContent = GetModalContent(err.Error())
		return *m, func() tea.Msg {
			return OpenModalMsg{
				ModalMode:    ModalModeAlert,
				AlertTitle:   "GUI File Picker Required",
				AlertMessage: err.Error(),
			}
		}
	}

	return *m, PickFileCmd(picker, args)
}

func FormatFileSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func renderAttachmentChip(filePath string, maxBarWidth int) string {
	if filePath == "" {
		return ""
	}
	if filePath == "uploading..." {
		uploadingStyle := lipgloss.NewStyle().
			Foreground(DefaultTheme.WarningColor).
			Bold(true).
			Render("⏳ Uploading attachment...")
		return lipgloss.NewStyle().Padding(0, 1).Render(uploadingStyle)
	}

	baseName := filepath.Base(filePath)
	if len(baseName) > 40 {
		baseName = baseName[:37] + "..."
	}

	var sizeStr string
	if fi, err := os.Stat(filePath); err == nil {
		sizeStr = FormatFileSize(fi.Size())
	}

	iconPart := lipgloss.NewStyle().
		Foreground(DefaultTheme.AccentColor).
		Bold(true).
		Render("📎 " + baseName)

	var sizePart string
	if sizeStr != "" {
		sizePart = lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryText).
			Render(" (" + sizeStr + ")")
	}

	hintPart := lipgloss.NewStyle().
		Foreground(DefaultTheme.SecondaryText).
		Render(" • [Esc] Detach")

	chipContent := iconPart + sizePart + hintPart

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DefaultTheme.AccentColor).
		Background(lipgloss.Color("#1E293B")).
		Padding(0, 1).
		MarginBottom(0).
		Render(chipContent)
}
