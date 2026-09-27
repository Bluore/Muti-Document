package utils

import "math/rand"

// cursorColors is the palette handed out to newly connected users so their
// remote cursors can be told apart on the client.
var cursorColors = []string{
	"#E57373", "#F06292", "#BA68C8", "#9575CD",
	"#7986CB", "#64B5F6", "#4FC3F7", "#4DB6AC",
	"#81C784", "#FFB74D", "#FF8A65", "#A1887F",
}

// RandomCursorColor returns a random color from the shared cursor palette.
func RandomCursorColor() string {
	return cursorColors[rand.Intn(len(cursorColors))]
}
