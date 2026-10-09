package shared

import (
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

func TestFormatMessage_PhotoMedia(t *testing.T) {
	msg := &tg.Message{
		ID:      101,
		Message: "Check out this photo",
		Date:    int(time.Now().Unix()),
		Media: &tg.MessageMediaPhoto{
			Photo: &tg.Photo{
				ID:            9991,
				AccessHash:    8882,
				FileReference: []byte("ref123"),
				Sizes: []tg.PhotoSizeClass{
					&tg.PhotoSize{
						Type: "s",
						W:    100,
						H:    100,
						Size: 1024,
					},
					&tg.PhotoSize{
						Type: "y",
						W:    1920,
						H:    1080,
						Size: 204800,
					},
				},
			},
		},
	}

	user := &types.UserInfo{FirstName: "Alice", PeerID: "123"}
	formatted := FormatMessage(msg, user, nil)

	if formatted == nil {
		t.Fatalf("expected formatted message, got nil")
	}
	if formatted.IsUnsupportedMessage {
		t.Errorf("expected photo message to be supported, got IsUnsupportedMessage=true")
	}
	if formatted.Content != "Check out this photo" {
		t.Errorf("expected caption 'Check out this photo', got %q", formatted.Content)
	}
	if formatted.MediaAttachment == nil {
		t.Fatalf("expected MediaAttachment to be populated, got nil")
	}
	if formatted.MediaAttachment.Type != types.MediaTypePhoto {
		t.Errorf("expected MediaTypePhoto, got %v", formatted.MediaAttachment.Type)
	}
	if formatted.MediaAttachment.Width != 1920 || formatted.MediaAttachment.Height != 1080 {
		t.Errorf("expected dimensions 1920x1080, got %dx%d", formatted.MediaAttachment.Width, formatted.MediaAttachment.Height)
	}
	if formatted.MediaAttachment.FileSize != 204800 {
		t.Errorf("expected FileSize 204800, got %d", formatted.MediaAttachment.FileSize)
	}
}

func TestFormatMessage_DocumentMedia(t *testing.T) {
	msg := &tg.Message{
		ID:      102,
		Message: "Here is the report",
		Date:    int(time.Now().Unix()),
		Media: &tg.MessageMediaDocument{
			Document: &tg.Document{
				ID:            5551,
				AccessHash:    6662,
				FileReference: []byte("refdoc"),
				MimeType:      "application/pdf",
				Size:          1048576, // 1 MB
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeFilename{
						FileName: "report.pdf",
					},
				},
			},
		},
	}

	user := &types.UserInfo{FirstName: "Bob", PeerID: "456"}
	formatted := FormatMessage(msg, user, nil)

	if formatted == nil {
		t.Fatalf("expected formatted message, got nil")
	}
	if formatted.IsUnsupportedMessage {
		t.Errorf("expected document message to be supported, got IsUnsupportedMessage=true")
	}
	if formatted.Content != "Here is the report" {
		t.Errorf("expected caption 'Here is the report', got %q", formatted.Content)
	}
	if formatted.MediaAttachment == nil {
		t.Fatalf("expected MediaAttachment to be populated, got nil")
	}
	if formatted.MediaAttachment.Type != types.MediaTypeDocument {
		t.Errorf("expected MediaTypeDocument, got %v", formatted.MediaAttachment.Type)
	}
	if formatted.MediaAttachment.FileName != "report.pdf" {
		t.Errorf("expected FileName 'report.pdf', got %q", formatted.MediaAttachment.FileName)
	}
	if formatted.MediaAttachment.FileSize != 1048576 {
		t.Errorf("expected FileSize 1048576, got %d", formatted.MediaAttachment.FileSize)
	}
}

func TestFormatMessage_AudioAndVideoMedia(t *testing.T) {
	audioMsg := &tg.Message{
		ID:   103,
		Date: int(time.Now().Unix()),
		Media: &tg.MessageMediaDocument{
			Document: &tg.Document{
				ID:            7771,
				AccessHash:    8881,
				FileReference: []byte("refaudio"),
				MimeType:      "audio/ogg",
				Size:          50000,
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeAudio{
						Voice:    true,
						Duration: 15,
					},
				},
			},
		},
	}

	user := &types.UserInfo{FirstName: "Charlie", PeerID: "789"}
	formattedAudio := FormatMessage(audioMsg, user, nil)

	if formattedAudio.MediaAttachment == nil {
		t.Fatalf("expected MediaAttachment for voice, got nil")
	}
	if formattedAudio.MediaAttachment.Type != types.MediaTypeVoice {
		t.Errorf("expected MediaTypeVoice, got %v", formattedAudio.MediaAttachment.Type)
	}
	if formattedAudio.MediaAttachment.Duration != 15 {
		t.Errorf("expected Duration 15, got %d", formattedAudio.MediaAttachment.Duration)
	}

	videoMsg := &tg.Message{
		ID:   104,
		Date: int(time.Now().Unix()),
		Media: &tg.MessageMediaDocument{
			Document: &tg.Document{
				ID:            9991,
				AccessHash:    1111,
				FileReference: []byte("refvideo"),
				MimeType:      "video/mp4",
				Size:          5242880,
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeVideo{
						Duration: 60,
						W:        1280,
						H:        720,
					},
					&tg.DocumentAttributeFilename{
						FileName: "clip.mp4",
					},
				},
			},
		},
	}

	formattedVideo := FormatMessage(videoMsg, user, nil)
	if formattedVideo.MediaAttachment == nil {
		t.Fatalf("expected MediaAttachment for video, got nil")
	}
	if formattedVideo.MediaAttachment.Type != types.MediaTypeVideo {
		t.Errorf("expected MediaTypeVideo, got %v", formattedVideo.MediaAttachment.Type)
	}
	if formattedVideo.MediaAttachment.Duration != 60 {
		t.Errorf("expected Duration 60, got %d", formattedVideo.MediaAttachment.Duration)
	}
	if formattedVideo.MediaAttachment.Width != 1280 || formattedVideo.MediaAttachment.Height != 720 {
		t.Errorf("expected 1280x720, got %dx%d", formattedVideo.MediaAttachment.Width, formattedVideo.MediaAttachment.Height)
	}
}

func TestFormatMessage_MusicAudioWithTitleAndPerformer(t *testing.T) {
	musicMsg := &tg.Message{
		ID:   105,
		Date: int(time.Now().Unix()),
		Media: &tg.MessageMediaDocument{
			Document: &tg.Document{
				ID:            8881,
				AccessHash:    9991,
				FileReference: []byte("refmusic"),
				MimeType:      "audio/mpeg",
				Size:          7000000,
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeAudio{
						Voice:     false,
						Duration:  215,
						Title:     "Bohemian Rhapsody",
						Performer: "Queen",
					},
					&tg.DocumentAttributeFilename{
						FileName: "track1.mp3",
					},
				},
			},
		},
	}

	user := &types.UserInfo{FirstName: "Dave", PeerID: "999"}
	formattedMusic := FormatMessage(musicMsg, user, nil)

	if formattedMusic.MediaAttachment == nil {
		t.Fatalf("expected MediaAttachment for music, got nil")
	}
	if formattedMusic.MediaAttachment.Type != types.MediaTypeAudio {
		t.Errorf("expected MediaTypeAudio, got %v", formattedMusic.MediaAttachment.Type)
	}
	if formattedMusic.MediaAttachment.Title != "Bohemian Rhapsody" {
		t.Errorf("expected Title 'Bohemian Rhapsody', got %q", formattedMusic.MediaAttachment.Title)
	}
	if formattedMusic.MediaAttachment.Performer != "Queen" {
		t.Errorf("expected Performer 'Queen', got %q", formattedMusic.MediaAttachment.Performer)
	}
	if formattedMusic.MediaAttachment.Duration != 215 {
		t.Errorf("expected Duration 215, got %d", formattedMusic.MediaAttachment.Duration)
	}
}
