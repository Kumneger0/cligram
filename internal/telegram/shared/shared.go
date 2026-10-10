package shared

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"os/exec"

	"github.com/gotd/td/telegram/downloader"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

type ChannelOrUser interface {
	types.UserInfo | types.ChannelInfo
}

func FormatMessage[T ChannelOrUser](msg *tg.Message, userOrChannel *T, allMessages []tg.MessageClass) *types.FormattedMessage {
	if msg == nil {
		slog.Warn("Format message is becoming nil for some reason")
		return nil
	}

	var sender string
	var FromID *string
	var SenderUserInfo *types.UserInfo
	if msg.Out {
		sender = "You"
	} else {
		if userOrChannel == nil {
			sender = "unknown"
		} else {
			switch v := any(userOrChannel).(type) {
			case *types.ChannelInfo:
				if v != nil {
					sender = v.ChannelTitle
					FromID = &v.ID
				} else {
					sender = "unknown channel"
				}
			case *types.UserInfo:
				if v != nil {
					sender = v.FirstName
					FromID = &v.PeerID
					SenderUserInfo = v
				} else {
					sender = "unknown user"
				}
			default:
				sender = "unknown"
			}
		}
	}

	var reply *types.FormattedMessage
	if replyTo, ok := msg.GetReplyTo(); ok {
		if messageReply, ok := replyTo.(*tg.MessageReplyHeader); ok && len(allMessages) > 0 {
			replyMessage := getRelyMessage(allMessages, messageReply.ReplyToMsgID)
			reply = FormatMessage(replyMessage, userOrChannel, allMessages)
		}
	}

	var mediaAttachment *types.MediaAttachment
	isUnsupportedMessage := false
	if msg.Media != nil {
		switch m := msg.Media.(type) {
		case *tg.MessageMediaWebPage, *tg.MessageMediaEmpty:
			// Handled separately or no attachment
		case *tg.MessageMediaPhoto:
			mediaAttachment = parsePhotoMedia(m)
		case *tg.MessageMediaDocument:
			mediaAttachment = parseDocumentMedia(m)
		default:
			isUnsupportedMessage = true
		}
	}

	var content = msg.Message
	if isUnsupportedMessage {
		content = "This Message is not supported by this Telegram client."
	}

	var view int

	if result, ok := msg.GetViews(); ok {
		view = result
	}

	webPageMedia, _ := msg.Media.(*tg.MessageMediaWebPage)
	return &types.FormattedMessage{
		ID:                   msg.ID,
		Sender:               sender,
		Content:              content,
		IsFromMe:             msg.Out,
		Media:                nil,
		MediaAttachment:      mediaAttachment,
		Date:                 time.Unix(int64(msg.Date), 0),
		IsUnsupportedMessage: isUnsupportedMessage,
		WebPage:              nil,
		Document:             nil,
		FromID:               FromID,
		SenderUserInfo:       SenderUserInfo,
		ReplyTo:              reply,
		Reactions:            &msg.Reactions,
		Views:                view,
		HasWebPagePreview:    webPageMedia != nil,
		MessageMediaWebPage:  webPageMedia,
	}
}

func parsePhotoMedia(media *tg.MessageMediaPhoto) *types.MediaAttachment {
	if media == nil {
		return nil
	}
	photo, ok := media.Photo.(*tg.Photo)
	if !ok || photo == nil {
		return nil
	}

	var bestSize *tg.PhotoSize
	var thumbType string = "y"
	for _, sizeClass := range photo.Sizes {
		if ps, ok := sizeClass.(*tg.PhotoSize); ok {
			if bestSize == nil || ps.Size > bestSize.Size {
				bestSize = ps
				thumbType = ps.Type
			}
		}
	}

	var width, height int
	var fileSize int64
	if bestSize != nil {
		width = bestSize.W
		height = bestSize.H
		fileSize = int64(bestSize.Size)
	}

	inputLoc := &tg.InputPhotoFileLocation{
		ID:            photo.ID,
		AccessHash:    photo.AccessHash,
		FileReference: photo.FileReference,
		ThumbSize:     thumbType,
	}

	return &types.MediaAttachment{
		Type:          types.MediaTypePhoto,
		FileSize:      fileSize,
		Width:         width,
		Height:        height,
		Status:        types.MediaStatusIdle,
		InputLocation: inputLoc,
	}
}

func parseDocumentMedia(media *tg.MessageMediaDocument) *types.MediaAttachment {
	if media == nil {
		return nil
	}
	doc, ok := media.Document.(*tg.Document)
	if !ok || doc == nil {
		return nil
	}

	mType := types.MediaTypeDocument
	var fileName string
	var title, performer string
	var duration int
	var width, height int

	for _, attrClass := range doc.Attributes {
		switch attr := attrClass.(type) {
		case *tg.DocumentAttributeFilename:
			fileName = attr.FileName
		case *tg.DocumentAttributeAudio:
			if attr.Voice {
				mType = types.MediaTypeVoice
			} else {
				mType = types.MediaTypeAudio
			}
			duration = attr.Duration
			title = attr.Title
			performer = attr.Performer
			if fileName == "" && attr.Title != "" {
				if attr.Performer != "" {
					fileName = attr.Performer + " - " + attr.Title
				} else {
					fileName = attr.Title
				}
			}
		case *tg.DocumentAttributeVideo:
			mType = types.MediaTypeVideo
			duration = int(attr.Duration)
			width = attr.W
			height = attr.H
		case *tg.DocumentAttributeImageSize:
			width = attr.W
			height = attr.H
		}
	}

	if fileName == "" {
		fileName = DefaultMediaFileName(mType)
	}

	inputLoc := &tg.InputDocumentFileLocation{
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
	}

	return &types.MediaAttachment{
		Type:          mType,
		FileName:      fileName,
		Title:         title,
		Performer:     performer,
		FileSize:      doc.Size,
		MimeType:      doc.MimeType,
		Duration:      duration,
		Width:         width,
		Height:        height,
		Status:        types.MediaStatusIdle,
		InputLocation: inputLoc,
	}
}

func DefaultMediaFileName(mType types.MediaType) string {
	switch mType {
	case types.MediaTypePhoto:
		return "photo.jpg"
	case types.MediaTypeVideo:
		return "video.mp4"
	case types.MediaTypeAudio:
		return "audio.mp3"
	case types.MediaTypeVoice:
		return "voice_message.ogg"
	default:
		return "document"
	}
}

type GroupSenderResolution struct {
	Sender         string
	FromID         *string
	SenderUserInfo *types.UserInfo
}

func ResolveGroupMessageSender(msg *tg.Message, groupID string, users []tg.UserClass, chats []tg.ChatClass) GroupSenderResolution {
	if msg == nil {
		return GroupSenderResolution{Sender: "unknown"}
	}
	if msg.Out {
		return GroupSenderResolution{Sender: "You"}
	}

	if msg.FromID == nil {
		return GroupSenderResolution{
			Sender: "🛡️ Anonymous Admin",
			FromID: &groupID,
		}
	}

	switch p := msg.FromID.(type) {
	case *tg.PeerUser:
		for _, userClass := range users {
			if u, ok := userClass.(*tg.User); ok && u.ID == p.UserID {
				ui := ConvertTGUserToUserInfo(u)
				name := strings.TrimSpace(ui.FirstName + " " + ui.LastName)
				if name == "" {
					name = ui.Username
				}
				if name == "" {
					name = "User"
				}
				return GroupSenderResolution{
					Sender:         name,
					FromID:         &ui.PeerID,
					SenderUserInfo: ui,
				}
			}
		}
		userIDStr := strconv.FormatInt(p.UserID, 10)
		return GroupSenderResolution{
			Sender: fmt.Sprintf("User %d", p.UserID),
			FromID: &userIDStr,
		}
	case *tg.PeerChannel:
		chIDStr := strconv.FormatInt(p.ChannelID, 10)
		if chIDStr == groupID {
			return GroupSenderResolution{
				Sender: "🛡️ Anonymous Admin",
				FromID: &chIDStr,
			}
		}
		for _, chatClass := range chats {
			if ch, ok := chatClass.(*tg.Channel); ok && ch.ID == p.ChannelID {
				return GroupSenderResolution{
					Sender: "📢 " + ch.Title,
					FromID: &chIDStr,
				}
			}
		}
		return GroupSenderResolution{
			Sender: "📢 Channel",
			FromID: &chIDStr,
		}
	case *tg.PeerChat:
		chatIDStr := strconv.FormatInt(p.ChatID, 10)
		if chatIDStr == groupID {
			return GroupSenderResolution{
				Sender: "🛡️ Anonymous Admin",
				FromID: &chatIDStr,
			}
		}
		for _, chatClass := range chats {
			if ch, ok := chatClass.(*tg.Chat); ok && ch.ID == p.ChatID {
				return GroupSenderResolution{
					Sender: "📢 " + ch.Title,
					FromID: &chatIDStr,
				}
			}
		}
		return GroupSenderResolution{
			Sender: "📢 Group",
			FromID: &chatIDStr,
		}
	default:
		return GroupSenderResolution{
			Sender: "unknown",
		}
	}
}

func FormatGroupMessage(msg *tg.Message, groupID string, users []tg.UserClass, chats []tg.ChatClass, allMessages []tg.MessageClass) *types.FormattedMessage {
	if msg == nil {
		return nil
	}

	resolution := ResolveGroupMessageSender(msg, groupID, users, chats)

	var reply *types.FormattedMessage
	if replyTo, ok := msg.GetReplyTo(); ok {
		if messageReply, ok := replyTo.(*tg.MessageReplyHeader); ok && len(allMessages) > 0 {
			replyMessage := getRelyMessage(allMessages, messageReply.ReplyToMsgID)
			reply = FormatGroupMessage(replyMessage, groupID, users, chats, allMessages)
		}
	}

	var mediaAttachment *types.MediaAttachment
	isUnsupportedMessage := false
	if msg.Media != nil {
		switch m := msg.Media.(type) {
		case *tg.MessageMediaWebPage, *tg.MessageMediaEmpty:
			// Handled separately or no attachment
		case *tg.MessageMediaPhoto:
			mediaAttachment = parsePhotoMedia(m)
		case *tg.MessageMediaDocument:
			mediaAttachment = parseDocumentMedia(m)
		default:
			isUnsupportedMessage = true
		}
	}

	var content = msg.Message
	if isUnsupportedMessage {
		content = "This Message is not supported by this Telegram client."
	}

	var view int
	if result, ok := msg.GetViews(); ok {
		view = result
	}

	webPageMedia, _ := msg.Media.(*tg.MessageMediaWebPage)

	return &types.FormattedMessage{
		ID:                   msg.ID,
		Sender:               resolution.Sender,
		Content:              content,
		IsFromMe:             msg.Out,
		Media:                nil,
		MediaAttachment:      mediaAttachment,
		Date:                 time.Unix(int64(msg.Date), 0),
		IsUnsupportedMessage: isUnsupportedMessage,
		WebPage:              nil,
		Document:             nil,
		FromID:               resolution.FromID,
		SenderUserInfo:       resolution.SenderUserInfo,
		ReplyTo:              reply,
		Reactions:            &msg.Reactions,
		Views:                view,
		HasWebPagePreview:    webPageMedia != nil,
		MessageMediaWebPage:  webPageMedia,
	}
}

func getRelyMessage(allMessages []tg.MessageClass, messageID int) *tg.Message {
	var message *tg.Message
	for _, msg := range allMessages {
		msg, ok := msg.(*tg.Message)
		if !ok {
			continue
		}
		if msg.ID == messageID {
			message = msg
			break
		}
	}
	return message
}

func GetUserInfo(ctx context.Context, client tg.Client, userID int64) (*types.UserInfo, error) {
	inputUser := &tg.InputUser{
		UserID: userID,
	}

	userClasses, err := client.UsersGetUsers(ctx, []tg.InputUserClass{inputUser})
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "users.getUsers failed", err)
	}

	if len(userClasses) == 0 {
		return nil, types.NewUserNotFoundError(userID)
	}

	user, ok := userClasses[0].(*tg.User)

	if !ok {
		return nil, types.NewUserNotFoundError(userID)
	}
	return ConvertTGUserToUserInfo(user), nil
}

func ConvertTGUserToUserInfo(tgUser *tg.User) *types.UserInfo {
	firstName := tgUser.FirstName
	lastName := tgUser.LastName

	if tgUser.Deleted {
		firstName = "Deleted Account"
	}

	userInfo := &types.UserInfo{
		FirstName:           firstName,
		LastName:            lastName,
		Username:            tgUser.Username,
		IsBot:               tgUser.Bot,
		PeerID:              strconv.FormatInt(tgUser.ID, 10),
		AccessHash:          strconv.FormatInt(tgUser.AccessHash, 10),
		IsTyping:            false,
		IsOnline:            false,
		Premium:             tgUser.Premium,
		PhoneCallsPrivate:   false,
		PhoneCallsAvailable: !tgUser.Bot,
	}

	if status := getUserOnlineStatus(tgUser.Status); status != nil {
		userInfo.IsOnline = status.IsOnline
		userInfo.LastSeen = status.LastSeen
	}

	if tgUser.Deleted {
		lastSeen := "Last seen long time ago"
		userInfo.LastSeen = &lastSeen
	}
	return userInfo
}

func getUserOnlineStatus(status tg.UserStatusClass) *userOnlineStatus {
	if status == nil {
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: nil,
		}
	}

	switch s := status.(type) {
	case *tg.UserStatusOnline:
		lastSeen := "online"
		return &userOnlineStatus{
			IsOnline: true,
			LastSeen: &lastSeen,
		}
	case *tg.UserStatusLastMonth:
		lastSeen := "last seen within a month"
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	case *tg.UserStatusRecently:
		lastSeen := "last seen recently"
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	case *tg.UserStatusOffline:
		lastSeen := calculateLastSeenHumanReadable(s.WasOnline)
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	case *tg.UserStatusLastWeek:
		lastSeen := "last seen within a week"
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	default:
		lastSeen := "last seen long time ago"
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	}
}

type userOnlineStatus struct {
	IsOnline bool
	LastSeen *string
}

func calculateLastSeenHumanReadable(wasOnline int) string {
	var s strings.Builder
	lastSeenTime := time.Unix(int64(wasOnline), 0)
	currentTime := time.Now()
	diff := currentTime.Sub(lastSeenTime)
	lastSeenFormattedOnlyTime := lastSeenTime.Format("03:04 PM")

	if diff.Seconds() < float64(60) {
		return "last seen just now"
	}
	if diff.Hours() < float64(24) {
		s.WriteString("last seen at ")
		s.WriteString(lastSeenFormattedOnlyTime)
		return s.String()
	}
	if diff.Hours() < float64(48) {
		s.WriteString("last seen yesterday at ")
		s.WriteString(lastSeenFormattedOnlyTime)
		return s.String()
	}

	lastSeenFormattedWithDate := lastSeenTime.Format("02/01/2006 03:04 PM")
	s.WriteString("last seen on ")
	s.WriteString(lastSeenFormattedWithDate)
	return s.String()
}

type MessageHistoryEntities struct {
	Messages []tg.MessageClass
	Users    []tg.UserClass
	Chats    []tg.ChatClass
}

func GetMessageAndUserClasses(history tg.MessagesMessagesClass) (*MessageHistoryEntities, error) {
	entities := &MessageHistoryEntities{}
	switch h := history.(type) {
	case *tg.MessagesMessagesSlice:
		entities.Messages = h.Messages
		entities.Users = h.Users
		entities.Chats = h.Chats
	case *tg.MessagesChannelMessages:
		entities.Messages = h.Messages
		entities.Users = h.Users
		entities.Chats = h.Chats
	case *tg.MessagesMessages:
		entities.Messages = h.Messages
		entities.Users = h.Users
		entities.Chats = h.Chats
	default:
		return nil, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, fmt.Sprintf("unsupported history type: %T", history), nil)
	}
	return entities, nil
}

func ReadStories(ctx context.Context, client *telegram.Client, peer types.Peer, maxID int) error {
	inputPeer, err := ConvertPeerToInputPeer(peer)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	request := &tg.StoriesReadStoriesRequest{
		Peer:  inputPeer,
		MaxID: maxID,
	}
	_, err = client.API().StoriesReadStories(ctx, request)
	if err != nil {
		slog.Error(err.Error())
	}
	return nil
}

func DownloadStoryMedia(ctx context.Context, client *telegram.Client, story *tg.StoryItem, outDir string, peerID string) (*int, error) {
	switch media := story.Media.(type) {
	case *tg.MessageMediaDocument:
		if doc, ok := media.Document.AsNotEmpty(); ok {
			ext := "bin"
			var thumbSize string
			if len(doc.Thumbs) > 0 {
				var d int
				for _, t := range doc.Thumbs {
					if thumb, ok := t.(*tg.PhotoSize); ok {
						if thumb.Size > d {
							d = thumb.Size
						}
					}
					thumbSize = fmt.Sprintf("%d", d)
				}
			}
			switch doc.MimeType {
			case "video/mp4":
				ext = "mp4"
			case "image/jpeg":
				ext = "jpg"
			}

			filePath := filepath.Join(outDir, fmt.Sprintf("story_%d,%s.%s", story.ID, peerID, ext))
			return &story.ID, saveMediaToFileSystem(ctx, client.API(), filePath, doc.AsInputDocumentFileLocation(thumbSize))
		}

	case *tg.MessageMediaPhoto:
		if ph, ok := media.Photo.AsNotEmpty(); ok {
			filePath := filepath.Join(outDir, fmt.Sprintf("story_%d,%s.%s", story.ID, peerID, "jpg"))
			if fileInfo, err := os.Stat(filePath); err == nil {
				if fileInfo.Size() > 0 {
					err := OpenFileInDefaultApp(filePath)
					if err != nil {
						return nil, err
					}
					return &story.ID, nil
				}
			}

			photoFileLocation := &tg.InputPhotoFileLocation{
				ID:            ph.ID,
				AccessHash:    ph.AccessHash,
				FileReference: ph.FileReference,
				ThumbSize:     "y",
			}
			return &story.ID, saveMediaToFileSystem(ctx, client.API(), filePath, photoFileLocation)
		}

	default:
		return nil, errors.New("No downloadable media in this story")
	}

	return nil, errors.New("i have no idea for some fucking reason we are not able to get the type of story")
}

func DownloadMessageMedia(ctx context.Context, client *tg.Client, peerID string, msgID int, att *types.MediaAttachment) (string, error) {
	if att == nil || att.InputLocation == nil {
		return "", errors.New("no media attachment or input location found")
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil || cacheDir == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr == nil && home != "" {
			cacheDir = filepath.Join(home, ".cache")
		} else {
			cacheDir = os.TempDir()
		}
	}
	mediaDir := filepath.Join(cacheDir, "cligram", "media")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return "", err
	}

	fileName := att.FileName
	if fileName == "" {
		fileName = DefaultMediaFileName(att.Type)
	}

	sanitizedFile := filepath.Base(fileName)
	targetPath := filepath.Join(mediaDir, fmt.Sprintf("%s_%d_%s", peerID, msgID, sanitizedFile))

	err = saveMediaToFileSystem(ctx, client, targetPath, att.InputLocation)
	if err != nil {
		return "", err
	}
	return targetPath, nil
}

func saveMediaToFileSystem(ctx context.Context, client *tg.Client, filePath string, inputFileLocation tg.InputFileLocationClass) error {
	dl := downloader.NewDownloader()
	if fileInfo, err := os.Stat(filePath); err == nil {
		if fileInfo.Size() > 0 {
			err := OpenFileInDefaultApp(filePath)
			if err != nil {
				return err
			}
			return nil
		}
	}

	dBuilder := dl.Download(client, inputFileLocation)
	_, err := dBuilder.ToPath(ctx, filePath)
	if err != nil {
		return err
	}

	err = OpenFileInDefaultApp(filePath)
	if err != nil {
		return err
	}
	return nil
}

func OpenFileInDefaultApp(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

func ConvertPeerToInputPeer(peer types.Peer) (tg.InputPeerClass, error) {
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
		return &tg.InputPeerUser{UserID: peerID, AccessHash: accessHash}, nil
	case types.ChannelChat:
		accessHash, err := strconv.ParseInt(peer.AccessHash, 10, 64)
		if err != nil {
			return nil, types.NewInvalidPeerError(peer.AccessHash)
		}
		return &tg.InputPeerChannel{ChannelID: peerID, AccessHash: accessHash}, nil
	case types.GroupChat:
		if peer.AccessHash == "" {
			return &tg.InputPeerChat{ChatID: peerID}, nil
		}
		accessHash, err := strconv.ParseInt(peer.AccessHash, 10, 64)
		if err != nil {
			return nil, types.NewInvalidPeerError(peer.AccessHash)
		}
		return &tg.InputPeerChannel{ChannelID: peerID, AccessHash: accessHash}, nil
	default:
		return nil, types.NewTelegramError(types.ErrorCodeInvalidPeer, "unsupported chat type", nil)
	}
}
