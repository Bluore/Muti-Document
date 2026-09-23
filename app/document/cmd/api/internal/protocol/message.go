package protocol

import "github.com/Bluore/mutidoc/pkg/ot"

type ClientMessage struct {
	Edit *ClientEditMsg `json:"edit,omitempty"`
}

type ClientInitMessage struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

type ClientEditMsg struct {
	Reversion uint32      `json:"reversion"`
	Operator  ot.Operator `json:"operator"`
}

type ServerMessage struct {
	Edit *ServerEditMsg `json:"edit"`
	Init *ServerInitMsg `json:"init"`
}

type ServerEditMsg struct {
	Reversion uint32        `json:"reversion"`
	Operator  []ot.Operator `json:"operator"`
}

type ServerInitMsg struct {
	Reversion uint32 `json:"reversion"`
	Content   string `json:"content"`
	UserID    string `json:"user_id"`
	Users     []User `json:"users"`
}
