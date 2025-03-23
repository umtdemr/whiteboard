package ws

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
)

const (
	CmdJoin   = "join"
	CmdCursor = "cursor" // collaborator cursors
)

// Server to client events
const (
	EventUserJoined = "USER_JOINED"
	EventUserLeft   = "USER_LEFT"
	EventCursor     = "CURSOR" // on client's cursor update
)

type ErrorCode int

const (
	ErrCodeCmdNotFound ErrorCode = 1000 + iota
	ErrCodeIdMustSet
	ErrCodeAuth
	ErrCodeField // error for validations
	ErrCodeUnknown
	ErrCodeUnknownMessageType
	ErrCodeUnknownCompressionMethod
	ErrCodeJsonDecoding
)

type WsError struct {
	Code    ErrorCode
	Message string
}

func (we *WsError) Error() string {
	return fmt.Sprintf("%d: %s", we.Code, we.Message)
}

func (we *WsError) toResponse() ErrorResponse {
	return ErrorResponse{
		Code:    we.Code,
		Message: we.Message,
	}
}

var (
	ErrCmdNotFound              = &WsError{ErrCodeCmdNotFound, "command not found"}
	ErrIdMustSet                = &WsError{ErrCodeIdMustSet, "id must set"}
	ErrAuth                     = &WsError{ErrCodeAuth, "not authorized"}
	ErrUnknownMessageType       = &WsError{ErrCodeUnknownMessageType, "only binary messages are allowed"}
	ErrUnknownCompressionMethod = &WsError{ErrCodeUnknownCompressionMethod, "unknown compression method"}
)

// envelope wraps JSON
type envelope map[string]any

type FieldError struct {
	errors map[string]string
}

func (f FieldError) Error() string {
	return fmt.Sprintf("%d: %s", ErrCodeField, "field error")
}

type ErrorResponse struct {
	Message string    `json:"message"`
	Code    ErrorCode `json:"code"`
}

type FieldErrorResponse struct {
	ErrorResponse
	Fields map[string]string `json:"fields"`
}

func compressData(message any) ([]byte, error) {
	decoded, err := json.Marshal(message)

	if err != nil {
		return nil, err
	}

	b := bytes.NewBuffer(nil)
	msgWriter := zlib.NewWriter(b)
	_, err = msgWriter.Write(decoded)
	msgWriter.Close()

	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
