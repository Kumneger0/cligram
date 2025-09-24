// nolint:revive
package types

import "fmt"

type TelegramError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause,omitempty"`
}

func (e *TelegramError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("telegram error %d: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("telegram error %d: %s", e.Code, e.Message)
}

func (e *TelegramError) Unwrap() error {
	return e.Cause
}

const (
	ErrorCodeAuthFailed        = 1001
	ErrorCodeSendFailed        = 1002
	ErrorCodeGetMessagesFailed = 1003
	ErrorCodeDeleteFailed      = 1004
	ErrorCodeEditFailed        = 1005
	ErrorCodeForwardFailed     = 1006
	ErrorCodeUserNotFound      = 1007
	ErrorCodeInvalidPeer       = 1008
	ErrorCodeSessionFailed     = 1009
	ErrorCodeUploadFailed      = 1010
)

func NewTelegramError(code int, message string, cause error) *TelegramError {
	return &TelegramError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func NewAuthError(cause error) *TelegramError {
	return NewTelegramError(ErrorCodeAuthFailed, "authentication failed", cause)
}

func NewSendMessageError(cause error) *TelegramError {
	return NewTelegramError(ErrorCodeSendFailed, "failed to send message", cause)
}

func NewGetMessagesError(cause error) *TelegramError {
	return NewTelegramError(ErrorCodeGetMessagesFailed, "failed to get messages", cause)
}

func NewDeleteMessageError(cause error) *TelegramError {
	return NewTelegramError(ErrorCodeDeleteFailed, "failed to delete message", cause)
}

func NewUserNotFoundError(userID int64) *TelegramError {
	return NewTelegramError(ErrorCodeUserNotFound, fmt.Sprintf("user with ID %d not found", userID), nil)
}

func NewInvalidPeerError(peerID string) *TelegramError {
	return NewTelegramError(ErrorCodeInvalidPeer, fmt.Sprintf("invalid peer ID: %s", peerID), nil)
}

func NewEditMessageError(cause error) *TelegramError {
	return NewTelegramError(ErrorCodeEditFailed, "failed to edit message", cause)
}

func NewForwardMessageError(cause error) *TelegramError {
	return NewTelegramError(ErrorCodeForwardFailed, "failed to forward message", cause)
}
