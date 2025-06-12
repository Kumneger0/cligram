package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	JsProcess *os.Process
	RpcClient *JsonRpcClient
)

type JsonRpcClient struct {
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Cmd    *exec.Cmd
	NextID int
	Mu     sync.Mutex
}

type JsonRpcRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type BaseJsonRpcResponse struct {
	JsonRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Result  []interface{} `json:"result,omitempty"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
}

type UserGroupsMsg struct {
	Err error
	//groupds and channels share same propertiy so we don't have to redefine its types use this instead
	Response *UserChannelResponse
}

func (c *JsonRpcClient) GetUserGroups() tea.Cmd {
	userGroupsRpcResponse, err := c.Call("getUserChats", []string{"group"})
	return func() tea.Msg {
		if err != nil {
			return UserGroupsMsg{Err: err}
		}
		var response UserChannelResponse
		if err := json.Unmarshal(userGroupsRpcResponse, &response); err != nil {
			return UserGroupsMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userGroupsRpcResponse), err)}
		}
		if response.Error != nil {
			return UserGroupsMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return UserGroupsMsg{Err: nil, Response: &response}
	}
}

func writeLosToFIle(file *os.File, content []byte) error {
	_, err := file.Write(content)
	return err
}

type PeerInfo struct {
	AccessHash string `json:"accessHash"`
	PeerID     string `json:"peerId"`
}

func (c *JsonRpcClient) Call(method string, params interface{}) ([]byte, error) {
	c.Mu.Lock()
	id := c.NextID
	c.NextID++
	c.Mu.Unlock()

	request := JsonRpcRequest{
		JsonRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	requestPayloadBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var requestBuffer bytes.Buffer
	requestBuffer.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(requestPayloadBytes)))
	requestBuffer.WriteString("\r\n")
	requestBuffer.Write(requestPayloadBytes)

	_, err = c.Stdin.Write(requestBuffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to write request to stdin: %w", err)
	}

	jsonPayloadBytes, err := ReadStdOut(c)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	cwd, _ := os.Getwd()
	file, _ := os.Create(filepath.Join(cwd, "logs.json"))

	//TODO:don't forget to remove this is just for debuging purpose
	writeLosToFIle(file, jsonPayloadBytes)

	return jsonPayloadBytes, nil
}

func ReadStdOut(rpcClient *JsonRpcClient) ([]byte, error) {
	reader := bufio.NewReader((rpcClient.Stdout))
	var contentLength int = -1
	for {
		lineBytes, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF && contentLength != -1 {
				return nil, fmt.Errorf("EOF while expecting JSON content of length %d", contentLength)
			}
			return nil, fmt.Errorf("failed to read response header line: %w", err)
		}
		line := string(lineBytes)

		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headerName := strings.TrimSpace(parts[0])
			headerValue := strings.TrimSpace(parts[1])
			if strings.EqualFold(headerName, "Content-Length") {
				cl, err := strconv.Atoi(headerValue)
				if err != nil {
					return nil, fmt.Errorf("invalid Content-Length value '%s': %w", headerValue, err)
				}
				contentLength = cl
			}
		}
	}

	if contentLength == -1 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("Content-Length is 0, no JSON payload expected/read")
	}

	jsonPayloadBytes := make([]byte, contentLength)
	n, err := io.ReadFull(reader, jsonPayloadBytes)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("EOF while reading JSON payload: expected %d bytes, got %d", contentLength, n)
		}
		return nil, fmt.Errorf("failed to read JSON payload (expected %d bytes, read %d): %w", contentLength, n, err)
	}
	if n != contentLength {
		return nil, fmt.Errorf("short read for JSON payload: expected %d bytes, got %d", contentLength, n)
	}
	return jsonPayloadBytes, nil
}

type UserInfo struct {
	FirstName   string  `json:"firstName"`
	IsBot       bool    `json:"isBot"`
	PeerID      string  `json:"peerId"`
	AccessHash  string  `json:"accessHash"`
	UnreadCount int     `json:"unreadCount"`
	LastSeen    *string `json:"lastSeen"`
	IsOnline    bool    `json:"isOnline"`
}

type ChannelAndGroupInfo struct {
	ChannelTitle      string  `json:"title"`
	Username          *string `json:"username"`
	ChannelID         string  `json:"channelId"`
	AccessHash        string  `json:"accessHash"`
	IsCreator         bool    `json:"isCreator"`
	IsBroadcast       bool    `json:"isBroadcast"`
	ParticipantsCount *int    `json:"participantsCount"`
	UnreadCount       int     `json:"unreadCount"`
}

func (u UserInfo) Title() string {
	return u.FirstName
}

func (u UserInfo) FilterValue() string {
	return u.FirstName
}

func (c ChannelAndGroupInfo) FilterValue() string {
	return c.ChannelTitle
}

func (c ChannelAndGroupInfo) Title() string {
	return c.ChannelTitle
}
