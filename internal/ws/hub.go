package ws

import (
	"encoding/json"
	"errors"
	"github.com/umtdemr/wb-backend/internal/data"
)

const (
	CmdJoin   = "join"
	CmdCursor = "cursor" // collaborator cursors
)

// Server to client events
const (
	StcUserJoined = "USER_JOINED"
	StcUserLeft   = "USER_LEFT"
	StcCursor     = "CURSOR" // on client's cursor update
)

const (
	ErrCodeCmdNotFound = 1000
	ErrCodeIdMustSet   = 1001
	ErrCodeAuth        = 1002
	ErrCodeField       = 1003 // error for validations
	ErrCodeUnknown     = 1004
)

var (
	ErrCmdNotFound = errors.New("command not found")
	ErrIdMustSet   = errors.New("id must set")
	ErrAuth        = errors.New("not authorized")
)

// envelope wraps JSON
type envelope map[string]any

type FieldError struct {
	errors map[string]string
}

func (f FieldError) Error() string {
	return "field error"
}

// Hub maintains the set of active clients
type Hub struct {
	boards map[string]map[*Client]bool

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	models data.Models
}

func NewHub(models data.Models) *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		boards:     make(map[string]map[*Client]bool),
		models:     models,
	}
}

// incomingMessageReq represents the data that ws clients should send
type incomingMessageReq struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	Id   string          `json:"id"`
}

type messageResponse struct {
	ReplyTo string   `json:"reply_to,omitempty"`
	Event   string   `json:"event,omitempty"`
	Data    envelope `json:"data"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type FieldErrorResponse struct {
	ErrorResponse
	Fields map[string]string `json:"fields"`
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if _, exists := h.boards[client.boardId]; !exists {
				h.boards[client.boardId] = make(map[*Client]bool)
			}
			h.boards[client.boardId][client] = true
		case client := <-h.unregister:
			if board, exits := h.boards[client.boardId]; exits {
				if _, ok := board[client]; ok {
					delete(board, client)
					close(client.send)

					// clean empty boards
					if len(board) == 0 {
						delete(h.boards, client.boardId)
					}
				}
			}
		}
	}
}

// GetAllClientsInBoard collects all the clients in a board
func (h *Hub) GetAllClientsInBoard(boardSlugId string) []*Client {
	boardClients, exists := h.boards[boardSlugId]
	if !exists {
		return nil
	}

	clients := make([]*Client, 0, len(boardClients))
	for client := range boardClients {
		clients = append(clients, client)
	}

	return clients
}

type Cursor struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type CursorWithUser struct {
	*Cursor
	UserId   int64  `json:"user_id"`
	UserName string `json:"user_name"`
}
