package protocol

type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CursorColor string `json:"cursor_color"`
}
