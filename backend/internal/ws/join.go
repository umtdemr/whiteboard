package ws

import (
	"github.com/umtdemr/wb-backend/internal/data"
	"github.com/umtdemr/wb-backend/internal/token"
	"github.com/umtdemr/wb-backend/internal/validator"
)

// joinMessage is a request type for the board join requests
type joinMessage struct {
	BoardSlugId   string `json:"board_slug_id"`
	UserAuthToken string `json:"user_auth_token"`
}

type joinResponseUser struct {
	User   *data.User `json:"user"`
	Cursor *Cursor    `json:"cursor"`
}

type joinResponse struct {
	OnlineUsers []*joinResponseUser `json:"online_users"`
}

func (m *joinMessage) Handle(replyTo string, client *Client) error {
	user, err := client.hub.models.User.GetForToken(token.ScopeAuthentication, m.UserAuthToken)
	if err != nil || user == nil {
		client.sendErrorAuthResponse(replyTo)
		return nil
	}

	// set user for the client
	client.user = user
	client.boardId = m.BoardSlugId

	close(client.joined)

	client.hub.register <- &RegistrationRequest{
		Client:  client,
		ReplyTo: replyTo,
	}

	return nil
}

func (m *joinMessage) Validate(v *validator.Validator) {
	v.Check(len(m.BoardSlugId) == 12, "board_slug_id", "must be 12 bytes long")
	v.Check(len(m.UserAuthToken) > 5, "user_auth_token", "required")
}
