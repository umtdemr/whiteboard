package ws

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/umtdemr/wb-backend/internal/data"
	"strconv"
	"strings"
)

const (
	subjectPrefix       = "board."
	excludeClientHeader = "Exclude-Client"
)

// Hub maintains the set of active clients
type Hub struct {
	boards map[string]map[*Client]bool

	// Register requests from the clients.
	register chan *RegistrationRequest

	// Unregister requests from clients.
	unregister chan *Client

	models data.Models

	nc   *nats.Conn
	subs map[string]*nats.Subscription
}

func NewHub(models data.Models, nc *nats.Conn) *Hub {
	return &Hub{
		register:   make(chan *RegistrationRequest),
		unregister: make(chan *Client),
		boards:     make(map[string]map[*Client]bool),
		subs:       make(map[string]*nats.Subscription),
		models:     models,
		nc:         nc,
	}
}

type RegistrationRequest struct {
	Client  *Client
	ReplyTo string
}

func (h *Hub) Run() {
	for {
		select {
		case request := <-h.register:
			client := request.Client

			// register the user
			if _, exists := h.boards[client.boardId]; !exists {
				h.boards[client.boardId] = make(map[*Client]bool)
			}
			h.boards[client.boardId][client] = true

			// subscribe to broadcasts for this board
			h.ensureSubscription(client.boardId)

			otherUsers := h.boards[client.boardId]
			usersMap := make(map[int32]bool) // to avoid duplicated reports
			usersList := make([]*joinResponseUser, 0, len(h.boards[client.boardId]))

			for clientInBoard := range otherUsers {
				if clientInBoard.user.ID == client.user.ID {
					continue
				}

				// if exists, ignore
				if _, ok := usersMap[clientInBoard.user.ID]; ok {
					continue
				}

				usersMap[clientInBoard.user.ID] = true
				usersList = append(usersList, &joinResponseUser{User: clientInBoard.user, Cursor: clientInBoard.cursor})
			}
			resp := messageResponse{
				ReplyTo: request.ReplyTo,
				Data:    envelope{"join": joinResponse{OnlineUsers: usersList}},
			}

			client.sendCompressedData(resp)

			// let other users know about new online user
			client.broadCastMessage(messageResponse{
				Event: EventUserJoined,
				Data:  envelope{"user": client.user},
			})

		case client := <-h.unregister:
			if board, exits := h.boards[client.boardId]; exits {
				if _, ok := board[client]; ok {
					delete(board, client)
					close(client.send)

					// clean empty boards
					if len(board) == 0 {
						delete(h.boards, client.boardId)
					}

					h.cleanupSubscription(client.boardId)
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

func (h *Hub) broadcastToBoard(boardId string, msg []byte, excludeClientId int32) {
	subject := subjectPrefix + boardId

	m := &nats.Msg{
		Subject: subject,
		Data:    msg,
	}

	// add header to exclude specific client for broadcast message
	if excludeClientId != 0 {
		m.Header = nats.Header{
			excludeClientHeader: []string{fmt.Sprintf("%d", excludeClientId)},
		}
	}

	if err := h.nc.PublishMsg(m); err != nil {
		log.Error().Err(err).Msg("Failed to publish message")
	}
}

func (h *Hub) handleNatsMessage(m *nats.Msg) {
	boardId := strings.TrimPrefix(m.Subject, subjectPrefix)
	clients, exists := h.boards[boardId]
	if !exists {
		return
	}

	var excludeClientId int32
	if excludeId := m.Header.Get(excludeClientHeader); excludeId != "" {
		if id, err := strconv.Atoi(excludeId); err == nil {
			excludeClientId = int32(id)
		}
	}

	for client := range clients {
		if client.user.ID == excludeClientId {
			continue
		}
		select {
		case client.send <- m.Data:
		}
	}
}

func (h *Hub) ensureSubscription(boardId string) {
	subject := subjectPrefix + boardId

	if _, exists := h.subs[subject]; !exists {
		sub, err := h.nc.Subscribe(subject, func(m *nats.Msg) {
			h.handleNatsMessage(m)
		})

		if err != nil {
			log.Error().Err(err).Msg("Failed to subscribe to NATS subject")
			return
		}

		h.subs[subject] = sub
	}
}

func (h *Hub) cleanupSubscription(boardID string) {
	subject := subjectPrefix + boardID
	if sub, exists := h.subs[subject]; exists {
		if len(h.boards[boardID]) == 0 {
			_ = sub.Unsubscribe()
			delete(h.subs, subject)
		}
	}
}
