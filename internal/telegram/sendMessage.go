package telegram

import (
	"errors"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	uploader "github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

type SendMessageResult struct {
	Result SendMessageResponseType `json:"result,omitempty"`
}

type SendMessageResponseType struct {
	MessageID *int `json:"messageId,omitempty"`
}

type SendMessageMsg struct {
	Response *SendMessageResult
	Err      error
}

func (c *CligramClient) SendMessage(pInfo PeerInfo,
	msg string,
	isReplay bool,
	replyToMessageID string,
	cType ChatType, isFile bool,
	filePath string) tea.Cmd {
	return func() tea.Msg {
		paramsFixed := make([]any, 7)
		paramsFixed[0] = pInfo
		paramsFixed[1] = msg
		paramsFixed[2] = isReplay
		paramsFixed[3] = replyToMessageID
		paramsFixed[4] = cType
		paramsFixed[5] = isFile
		paramsFixed[6] = filePath

		var peer tg.InputPeerClass

		PeerID, err := strconv.ParseInt(pInfo.PeerID, 10, 64)
		if err != nil {
			return SendMessageMsg{Response: nil, Err: err}
		}

		accessHash, err := strconv.ParseInt(pInfo.AccessHash, 10, 64)
		if err != nil {
			return SendMessageMsg{Response: nil, Err: err}
		}
		switch cType {
		case UserChat, ChatType(Bot):
			peer = &tg.InputPeerUser{
				UserID:     PeerID,
				AccessHash: accessHash,
			}
		case ChannelChat, GroupChat:
			peer = &tg.InputPeerChannel{
				ChannelID:  PeerID,
				AccessHash: accessHash,
			}
		default:
			return SendMessageMsg{Response: nil, Err: errors.New("unsupported chat type")}
		}

		var replyTo tg.InputReplyToClass

		if isReplay {
			replayMSGID, err := strconv.ParseInt(replyToMessageID, 10, 0)
			if err != nil {
				return SendMessageMsg{Response: nil, Err: err}
			}
			replyTo = &tg.InputReplyToMessage{
				ReplyToMsgID: int(replayMSGID),
			}
		}

		if isFile {
			messageID, err := c.sendMedia(filePath, msg, peer, replyToMessageID)
			if err != nil {
				return SendMessageMsg{Response: nil, Err: err}
			}
			return SendMessageMsg{Response: &SendMessageResult{Result: SendMessageResponseType{MessageID: messageID}}, Err: nil}
		}

		request := tg.MessagesSendMessageRequest{
			Peer:     peer,
			ReplyTo:  replyTo,
			Message:  msg,
			RandomID: rand.Int63(),
		}

		_, err = c.Client.API().MessagesSendMessage(c.ctx, &request)

		if err != nil {
			return SendMessageMsg{Response: nil, Err: err}
		}
		return SendMessageMsg{Response: nil, Err: nil}
	}
}

func (c *CligramClient) sendMedia(path string, caption string, peer tg.InputPeerClass, replyToMessageID string) (*int, error) {
	_, err := detectFileType(path)

	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("wrong file path")
		}
		return nil, err
	}

	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	splited := strings.Split(path, "/")
	filename := splited[len(splited)-1]

	defer file.Close()
	fileInfo, _ := file.Stat()
	size := fileInfo.Size()

	upload := uploader.NewUpload(filename, file, size)
	up := uploader.NewUploader(c.Client.API())

	fileUpload, err := up.Upload(c.ctx, upload)

	if err != nil {
		return nil, err
	}

	media := &tg.InputMediaUploadedDocument{
		File: fileUpload,
	}

	var replyTo tg.InputReplyToClass

	if replyToMessageID != "" {
		replayMSGID, err := strconv.ParseInt(replyToMessageID, 10, 0)
		if err != nil {
			return nil, err
		}
		replyTo = &tg.InputReplyToMessage{
			ReplyToMsgID: int(replayMSGID),
		}
	}

	request := tg.MessagesSendMediaRequest{
		Peer:     peer,
		Media:    media,
		Message:  caption,
		RandomID: rand.Int63(),
		ReplyTo:  replyTo,
	}

	sendMedaiUpdateClass, err := c.Client.API().MessagesSendMedia(c.ctx, &request)
	if err != nil {
		return nil, err
	}
	updates := sendMedaiUpdateClass.(*tg.Updates).Updates[0]

	updateMessag := updates.(*tg.UpdateMessageID)

	return &updateMessag.ID, nil
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
