package message

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"

	"github.com/kumneger0/cligram/internal/telegram/types"
)

type Sender struct {
	client APIClient
}

type APIClient interface {
	GetAPI() *tg.Client
	Context() context.Context
}

func NewSender(client APIClient) types.MessageSender {
	return &Sender{
		client: client,
	}
}

func (s *Sender) SendText(ctx context.Context, peer types.Peer, text string, replyTo *int) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := convertPeerToInputPeer(peer)
		if err != nil {
			return types.SendMessageMsg{
				Response: nil,
				Err:      types.NewSendMessageError(err),
			}
		}

		var replyToClass tg.InputReplyToClass
		if replyTo != nil {
			replyToClass = &tg.InputReplyToMessage{
				ReplyToMsgID: *replyTo,
			}
		}

		request := tg.MessagesSendMessageRequest{
			Peer:     inputPeer,
			ReplyTo:  replyToClass,
			Message:  text,
			RandomID: rand.Int63(),
		}

		_, err = s.client.GetAPI().MessagesSendMessage(ctx, &request)
		if err != nil {
			return types.SendMessageMsg{
				Response: nil,
				Err:      types.NewSendMessageError(err),
			}
		}

		return types.SendMessageMsg{
			Response: &types.SendMessageResponse{},
			Err:      nil,
		}
	}
}

func (s *Sender) SendMedia(ctx context.Context, peer types.Peer, filePath string, caption string, replyTo *int) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := convertPeerToInputPeer(peer)
		if err != nil {
			return types.SendMessageMsg{
				Response: nil,
				Err:      types.NewSendMessageError(err),
			}
		}

		messageID, err := s.sendMediaFile(ctx, filePath, caption, inputPeer, replyTo)
		if err != nil {
			return types.SendMessageMsg{
				Response: nil,
				Err:      types.NewSendMessageError(err),
			}
		}

		return types.SendMessageMsg{
			Response: &types.SendMessageResponse{
				MessageID: messageID,
			},
			Err: nil,
		}
	}
}

func (s *Sender) sendMediaFile(ctx context.Context, path string, caption string, peer tg.InputPeerClass, replyTo *int) (*int, error) {
	_, err := detectFileType(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("file not found: " + path)
		}
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	filename := filepath.Base(path)

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := fileInfo.Size()

	upload := uploader.NewUpload(filename, file, size)
	uploaderClient := uploader.NewUploader(s.client.GetAPI())

	fileUpload, err := uploaderClient.Upload(ctx, upload)
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeUploadFailed, "failed to upload file", err)
	}

	media := &tg.InputMediaUploadedDocument{
		File: fileUpload,
	}

	var replyToClass tg.InputReplyToClass
	if replyTo != nil {
		replyToClass = &tg.InputReplyToMessage{
			ReplyToMsgID: *replyTo,
		}
	}

	request := tg.MessagesSendMediaRequest{
		Peer:     peer,
		Media:    media,
		Message:  caption,
		RandomID: rand.Int63(),
		ReplyTo:  replyToClass,
	}

	sendMediaUpdateClass, err := s.client.GetAPI().MessagesSendMedia(ctx, &request)
	if err != nil {
		return nil, err
	}

	var id *int
	switch u := sendMediaUpdateClass.(type) {
	case *tg.Updates:
		for _, up := range u.Updates {
			switch x := up.(type) {
			case *tg.UpdateMessageID:
				i := x.ID
				id = &i
			case *tg.UpdateNewMessage:
				if m, ok := x.Message.(*tg.Message); ok {
					i := m.ID
					id = &i
				}
			case *tg.UpdateNewChannelMessage:
				if m, ok := x.Message.(*tg.Message); ok {
					i := m.ID
					id = &i
				}
			}
		}
	case *tg.UpdatesCombined:
		for _, up := range u.Updates {
			if x, ok := up.(*tg.UpdateMessageID); ok {
				i := x.ID
				id = &i
			}
		}
	case *tg.UpdateShortSentMessage:
		i := u.ID
		id = &i
	}
	return id, nil
}

func convertPeerToInputPeer(peer types.Peer) (tg.InputPeerClass, error) {
	peerID, err := strconv.ParseInt(peer.ID, 10, 64)
	if err != nil {
		return nil, types.NewInvalidPeerError(peer.ID)
	}
	switch peer.ChatType {
	case types.UserChat, types.BotChat:
		accessHash, err := strconv.ParseInt(peer.AccessHash, 10, 64)
		if err != nil {
			return nil, types.NewInvalidPeerError(peer.AccessHash)
		}
		return &tg.InputPeerUser{
			UserID:     peerID,
			AccessHash: accessHash,
		}, nil
	case types.ChannelChat:
		accessHash, err := strconv.ParseInt(peer.AccessHash, 10, 64)
		if err != nil {
			return nil, types.NewInvalidPeerError(peer.AccessHash)
		}
		return &tg.InputPeerChannel{
			ChannelID:  peerID,
			AccessHash: accessHash,
		}, nil
	case types.GroupChat:
		return &tg.InputPeerChat{
			ChatID: peerID,
		}, nil
	default:
		return nil, types.NewTelegramError(types.ErrorCodeInvalidPeer, "unsupported chat type", nil)
	}
}

func detectFileType(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return "", err
	}

	mimeType := http.DetectContentType(buf[:n])
	return mimeType, nil
}
