package ws

type Cursor struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type CursorWithUser struct {
	*Cursor
	UserId   int64  `json:"user_id"`
	UserName string `json:"user_name"`
}

// cursorMessage is a request type to handle collaborator cursors
type cursorMessage struct {
	X float64 `json:"x"`
	Y float64 `json:"Y"`
}

func (m *cursorMessage) Handle(_ string, client *Client) error {
	if client.cursor == nil {
		client.cursor = &Cursor{}
	}
	// save cursor
	client.cursor.X = m.X
	client.cursor.Y = m.Y

	// send new cursor data to other users in same board
	client.broadCastMessage(messageResponse{
		Event: EventCursor,
		Data: envelope{"cursor": CursorWithUser{
			UserName: client.user.FullName,
			UserId:   int64(client.user.ID),
			Cursor:   client.cursor,
		}},
	})

	return nil
}
