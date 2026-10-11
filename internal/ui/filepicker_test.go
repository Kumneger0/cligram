package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFindGUIFilePicker_ZenityPriority(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		if file == "zenity" {
			return "/usr/bin/zenity", nil
		}
		if file == "kdialog" {
			return "/usr/bin/kdialog", nil
		}
		return "", errors.New("not found")
	}

	picker, args, err := FindGUIFilePicker()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if picker != "zenity" {
		t.Errorf("expected picker 'zenity', got %q", picker)
	}
	if len(args) == 0 || args[0] != "--file-selection" {
		t.Errorf("expected zenity --file-selection flag, got %v", args)
	}
}

func TestFindGUIFilePicker_KdialogFallback(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		if file == "zenity" {
			return "", errors.New("not found")
		}
		if file == "kdialog" {
			return "/usr/bin/kdialog", nil
		}
		return "", errors.New("not found")
	}

	picker, args, err := FindGUIFilePicker()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if picker != "kdialog" {
		t.Errorf("expected picker 'kdialog', got %q", picker)
	}
	if len(args) == 0 || args[0] != "--getopenfilename" {
		t.Errorf("expected kdialog --getopenfilename flag, got %v", args)
	}
}

func TestFindGUIFilePicker_NeitherFound(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	_, _, err := FindGUIFilePicker()
	if err == nil {
		t.Fatalf("expected ErrNoGUIFilePicker, got nil")
	}
	if !strings.Contains(err.Error(), "zenity") || !strings.Contains(err.Error(), "kdialog") {
		t.Errorf("expected error message mentioning zenity and kdialog, got %v", err)
	}
}

func TestTriggerFilePicker_NoPicker_ShowsActionableModal(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	var m Model
	updated, cmd := TriggerFilePicker(&m)
	if cmd != nil {
		t.Errorf("expected nil cmd when no picker found, got %v", cmd)
	}
	if !updated.IsModalVisible {
		t.Errorf("expected IsModalVisible to be true")
	}
	if !strings.Contains(updated.ModalContent, "zenity") || !strings.Contains(updated.ModalContent, "kdialog") {
		t.Errorf("expected modal content to guide user on zenity/kdialog, got %q", updated.ModalContent)
	}
}

func TestTriggerFilePicker_PickerFound_ReturnsCmd(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		if file == "zenity" {
			return "/usr/bin/zenity", nil
		}
		return "", errors.New("not found")
	}

	var m Model
	updated, cmd := TriggerFilePicker(&m)
	if cmd == nil {
		t.Fatalf("expected non-nil tea.Cmd when picker is found")
	}
	if updated.IsModalVisible {
		t.Errorf("expected IsModalVisible to remain false")
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{2560, "2.5 KB"},
		{1048576, "1.0 MB"},
		{10485760, "10.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		actual := FormatFileSize(tt.bytes)
		if actual != tt.expected {
			t.Errorf("FormatFileSize(%d) = %q, expected %q", tt.bytes, actual, tt.expected)
		}
	}
}

func TestRenderAttachmentChip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_preview_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := strings.Repeat("hello world ", 100)
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	chip := renderAttachmentChip(tmpFile.Name(), 80)
	if !strings.Contains(chip, "📎") {
		t.Errorf("expected chip to contain paperclip icon 📎, got:\n%s", chip)
	}
	base := filepath.Base(tmpFile.Name())
	if !strings.Contains(chip, base) {
		t.Errorf("expected chip to contain basename %q, got:\n%s", base, chip)
	}
	if !strings.Contains(chip, "[Esc] Detach") {
		t.Errorf("expected chip to contain [Esc] Detach hint, got:\n%s", chip)
	}

	// Test uploading... state
	uploadingChip := renderAttachmentChip("uploading...", 80)
	if !strings.Contains(uploadingChip, "Uploading attachment") {
		t.Errorf("expected uploading chip, got %q", uploadingChip)
	}

	// Test empty path
	if emptyChip := renderAttachmentChip("", 80); emptyChip != "" {
		t.Errorf("expected empty string for empty path, got %q", emptyChip)
	}
}

func TestPrepareInputView_WithAttachmentChip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "staged_file_*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	_ = tmpFile.Close()

	ti := textinput.New()
	ti.SetValue("Here is a screenshot")

	m := Model{
		Input:        ti,
		SelectedFile: tmpFile.Name(),
		FocusedOn:    Input,
	}

	view := prepareInputView(&m, layoutDimensions{mainWidth: 80, inputHeight: 3})
	if !strings.Contains(view, "📎") || !strings.Contains(view, "[Esc] Detach") {
		t.Errorf("expected input view to render attachment chip with [Esc] Detach, got:\n%s", view)
	}
}

func TestUpdate_FilePickedMsg(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "chosen_file_*.jpg")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	_ = tmpFile.Close()

	m := Model{
		Input:     textinput.New(),
		FocusedOn: Main,
	}

	// Success message
	updated, _ := m.Update(FilePickedMsg{Path: tmpFile.Name(), Canceled: false})
	updatedModel := updated.(Model)
	if updatedModel.SelectedFile != tmpFile.Name() {
		t.Errorf("expected SelectedFile to be %q, got %q", tmpFile.Name(), updatedModel.SelectedFile)
	}
	if updatedModel.FocusedOn != Input {
		t.Errorf("expected FocusedOn to switch to Input, got %v", updatedModel.FocusedOn)
	}

	// Canceled message does not overwrite
	updatedCancelled, _ := updatedModel.Update(FilePickedMsg{Path: "", Canceled: true})
	updatedCancelledModel := updatedCancelled.(Model)
	if updatedCancelledModel.SelectedFile != tmpFile.Name() {
		t.Errorf("expected SelectedFile to remain unchanged on cancellation, got %q", updatedCancelledModel.SelectedFile)
	}
}

func TestUpdate_EscKey_DetachesFileWithoutClearingDraft(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "draft_attach_*.pdf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	_ = tmpFile.Close()

	ti := textinput.New()
	ti.SetValue("Important draft message")

	m := Model{
		Input:        ti,
		SelectedFile: tmpFile.Name(),
		FocusedOn:    Input,
	}

	// Press Esc while file is staged
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(escMsg)
	updatedModel := updated.(Model)

	// 1. File must be detached
	if updatedModel.SelectedFile != "" {
		t.Errorf("expected SelectedFile to be cleared upon pressing Esc, got %q", updatedModel.SelectedFile)
	}

	// 2. Drafted text must NOT be cleared
	if updatedModel.Input.Value() != "Important draft message" {
		t.Errorf("expected input text to be preserved, got %q", updatedModel.Input.Value())
	}
}

func TestUpdate_Shortcuts_TriggerFilePicker(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		return "", errors.New("no picker installed")
	}

	m := Model{
		Input:     textinput.New(),
		FocusedOn: Input,
	}

	// Test Ctrl+A
	ctrlAMsg := tea.KeyMsg{Type: tea.KeyCtrlA}
	updatedA, _ := m.Update(ctrlAMsg)
	updatedModelA := updatedA.(Model)
	if !updatedModelA.IsModalVisible || !strings.Contains(updatedModelA.ModalContent, "zenity") {
		t.Errorf("expected Ctrl+A to trigger file picker and show actionable modal when missing")
	}

	// Test Ctrl+O
	ctrlOMsg := tea.KeyMsg{Type: tea.KeyCtrlO}
	updatedO, _ := m.Update(ctrlOMsg)
	updatedModelO := updatedO.(Model)
	if !updatedModelO.IsModalVisible || !strings.Contains(updatedModelO.ModalContent, "zenity") {
		t.Errorf("expected Ctrl+O to trigger file picker and show actionable modal when missing")
	}
}
