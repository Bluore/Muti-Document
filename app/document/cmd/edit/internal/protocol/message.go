package protocol

import (
	"github.com/Bluore/mutidoc/pkg/ot"
)

type ClientMessage struct {
	Edit     *ClientEditMsg     `json:"edit,omitempty"`
	Cursor   *ot.Cursor         `json:"cursor,omitempty"`
	SetTitle *ClientSetTitleMsg `json:"set_title,omitempty"`
}

type ClientInitMessage struct {
	Token string `json:"token"`
}

type ClientEditMsg struct {
	Reversion uint32      `json:"reversion"`
	Operator  ot.Operator `json:"operator"`
}

type ClientSetTitleMsg struct {
	Title string `json:"title"`
}

type ServerMessage struct {
	Edit     *ServerEditMsg     `json:"edit"`
	Init     *ServerInitMsg     `json:"init"`
	User     *ServerUserMsg     `json:"user"`
	Cursor   *[]ot.Cursor       `json:"cursor"`
	SetTitle *ServerSetTitleMsg `json:"set_title"`
}

type ServerEditMsg struct {
	Reversion uint32        `json:"reversion"`
	Operator  []ot.Operator `json:"operator"`
}

type ServerInitMsg struct {
	Reversion   uint32      `json:"reversion"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	UserID      string      `json:"user_id"`
	Users       []User      `json:"users"`
	Cursor      []ot.Cursor `json:"cursor"`
	CursorColor string      `json:"cursor_color"`
}

type ServerUserMsg struct {
	JoinUsers []User `json:"join_users"`
	QuitUsers []User `json:"quit_users"`
}

type ServerSetTitleMsg struct {
	UserID string `json:"user_id"`
	Title  string `json:"title"`
}
