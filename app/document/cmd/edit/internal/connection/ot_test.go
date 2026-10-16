package connection

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/protocol"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/ot"
)

// newTestOt 创建一个不接持久化的文档会话
func newTestOt() *DocumentClient {
	return NewOt(model.Documents{}, nil, nil, nil, nil)
}

func cursorAt(offset uint64) ot.Cursor {
	var c ot.Cursor
	c.Anchor.Offset = offset
	c.Anchor.Assoc = 1
	c.Head.Offset = offset
	c.Head.Assoc = 1
	return c
}

func namedCursor(userID string, offset uint64) ot.Cursor {
	c := cursorAt(offset)
	c.UserID = userID
	return c
}

func currentCursorMap(dc *DocumentClient) map[string]uint64 {
	m := make(map[string]uint64, len(dc.Cursors))
	for _, c := range dc.Cursors {
		m[c.UserID] = c.Anchor.Offset
	}
	return m
}

func TestApplyEditTransformsCursors(t *testing.T) {
	dc := newTestOt()
	dc.Content = []byte("abcde")

	dc.SetCursor(namedCursor("u1", 2))
	dc.SetCursor(namedCursor("u2", 4))
	// Drop the logs produced by the SetCursor calls so the assertions below see
	// only the cursor changes caused by the edits that follow.
	dc.CursorLogs = nil

	// insert "X" at offset 2: [Retain 2, Insert "X", Retain 3]
	ins := ot.Operator{Ops: []ot.OperatorAtomic{
		ot.NewRetain(2),
		ot.NewInsert("X"),
		ot.NewRetain(3),
	}}
	dc.applyEdit(&ins)

	if got := string(dc.Content); got != "abXcde" {
		t.Fatalf("content = %q, want %q", got, "abXcde")
	}

	got := currentCursorMap(dc)
	if got["u1"] != 3 || got["u2"] != 5 {
		t.Fatalf("cursors after insert = %v, want u1=3 u2=5", got)
	}

	logs, newVersion := dc.GetHistoryCursor(0)
	if newVersion != 1 || len(logs) != 1 {
		t.Fatalf("got %d logs at version %d, want 1 at 1", len(logs), newVersion)
	}
	if logs[0][0].UserID != "u1" || logs[0][0].Anchor.Offset != 3 {
		t.Fatalf("log[0][0] = %+v, want u1 at 3", logs[0][0])
	}
	if logs[0][1].UserID != "u2" || logs[0][1].Anchor.Offset != 5 {
		t.Fatalf("log[0][1] = %+v, want u2 at 5", logs[0][1])
	}

	// delete "a": [Delete 1, Retain 5]
	del := ot.Operator{Ops: []ot.OperatorAtomic{
		ot.NewDelete(1),
		ot.NewRetain(5),
	}}
	dc.applyEdit(&del)

	if got := string(dc.Content); got != "bXcde" {
		t.Fatalf("content = %q, want %q", got, "bXcde")
	}

	got = currentCursorMap(dc)
	if got["u1"] != 2 || got["u2"] != 4 {
		t.Fatalf("cursors after delete = %v, want u1=2 u2=4", got)
	}

	logs, newVersion = dc.GetHistoryCursor(1)
	if newVersion != 2 || len(logs) != 1 {
		t.Fatalf("got %d logs at version %d, want 1 at 2", len(logs), newVersion)
	}
	if logs[0][0].Anchor.Offset != 2 || logs[0][1].Anchor.Offset != 4 {
		t.Fatalf("second log = %+v, want offsets 2 and 4", logs[0])
	}
}

func TestGetCursorUpToDate(t *testing.T) {
	dc := newTestOt()

	logs, newVersion := dc.GetHistoryCursor(0)
	if len(logs) != 0 || newVersion != 0 {
		t.Fatalf("expected no logs, got %v at %d", logs, newVersion)
	}

	dc.SetCursor(namedCursor("u1", 1))

	logs, newVersion = dc.GetHistoryCursor(1)
	if len(logs) != 0 || newVersion != 1 {
		t.Fatalf("expected caller to be up to date, got %v at %d", logs, newVersion)
	}
}

func TestSetCursorRebasesToCurrent(t *testing.T) {
	dc := newTestOt()
	dc.Content = []byte("abcde")

	// insert "X" at offset 2 -> reversion 1, content "abXcde"
	ins := ot.Operator{Ops: []ot.OperatorAtomic{
		ot.NewRetain(2),
		ot.NewInsert("X"),
		ot.NewRetain(3),
	}}
	dc.applyEdit(&ins)

	// A cursor computed at reversion 0 must be rebased through the insert.
	dc.SetCursor(namedCursor("u1", 2))

	got := currentCursorMap(dc)
	if got["u1"] != 3 {
		t.Fatalf("cursors after rebase = %v, want u1=3", got)
	}

	cursor := dc.Cursors["u1"]
	if cursor.UserID != "u1" {
		t.Fatalf("cursor user id = %q, want u1", cursor.UserID)
	}
	if cursor.Reversion != 1 {
		t.Fatalf("cursor reversion = %d, want 1", cursor.Reversion)
	}
}

func TestSetCursorRecordsLog(t *testing.T) {
	dc := newTestOt()

	dc.SetCursor(namedCursor("u1", 1))

	logs, newVersion := dc.GetHistoryCursor(0)
	if newVersion != 1 || len(logs) != 1 {
		t.Fatalf("got %d logs at version %d, want 1 at 1", len(logs), newVersion)
	}
	if logs[0][0].UserID != "u1" || logs[0][0].Anchor.Offset != 1 {
		t.Fatalf("log[0][0] = %+v, want u1 at 1", logs[0][0])
	}
}

func TestSetCursorAheadReversionDoesNotPanic(t *testing.T) {
	dc := newTestOt()
	dc.Content = []byte("abcde")

	c := namedCursor("u1", 2)
	c.Reversion = 99
	dc.SetCursor(c) // must not panic

	if dc.Cursors["u1"].Anchor.Offset != 2 {
		t.Fatalf("offset = %d, want 2", dc.Cursors["u1"].Anchor.Offset)
	}
}

func TestSetTitleBroadcasts(t *testing.T) {
	dc := newTestOt()

	out := make(chan protocol.ServerMessage, 1)
	dc.Subscribers["u1"] = out

	dc.SetTitle("u1", "hello title")

	select {
	case msg := <-out:
		if msg.SetTitle == nil {
			t.Fatalf("expected set_title, got %+v", msg)
		}
		if msg.SetTitle.UserID != "u1" || msg.SetTitle.Title != "hello title" {
			t.Fatalf("set_title = %+v", msg.SetTitle)
		}
	default:
		t.Fatal("expected a broadcast title message")
	}

	// Setting the same title again must not broadcast.
	dc.SetTitle("u1", "hello title")
	select {
	case msg := <-out:
		t.Fatalf("unexpected second title broadcast: %+v", msg)
	default:
	}
}

func TestGetInitMessageSnapshot(t *testing.T) {
	dc := newTestOt()
	dc.Content = []byte("abcde")
	dc.Title = "doc"
	dc.Users["u1"] = protocol.User{ID: "u1", Name: "Alice", CursorColor: "#fff"}
	dc.SetCursor(namedCursor("u1", 2))

	msg, cursorVersion := dc.GetInitMessage()

	if msg.Reversion != 0 {
		t.Fatalf("reversion = %d, want 0", msg.Reversion)
	}
	if msg.Title != "doc" {
		t.Fatalf("title = %q, want doc", msg.Title)
	}
	if msg.Content != "abcde" {
		t.Fatalf("content = %q, want abcde", msg.Content)
	}
	if len(msg.Users) != 1 || msg.Users[0].ID != "u1" {
		t.Fatalf("users = %+v, want [u1]", msg.Users)
	}
	if len(msg.Cursor) != 1 || msg.Cursor[0].UserID != "u1" {
		t.Fatalf("cursors = %+v, want [u1]", msg.Cursor)
	}
	if cursorVersion != uint32(len(dc.CursorLogs)) || cursorVersion != 1 {
		t.Fatalf("cursorVersion = %d, want 1", cursorVersion)
	}
}

// TestDocumentClientConcurrentAccess hammers the client from several goroutines
// to make sure the lock order (ContentMu -> UserMu -> CursorMu) cannot deadlock.
// It is most useful under -race.
func TestDocumentClientConcurrentAccess(t *testing.T) {
	dc := newTestOt()
	const workers = 8

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, _ = dc.GetInitMessage()
		}
	}()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			userID := fmt.Sprintf("u%d", id)
			out := make(chan protocol.ServerMessage, 64)
			if !dc.ConnectUser(userID, protocol.User{ID: userID}, out) {
				return
			}
			defer dc.DisconnectUser(userID)

			for i := 0; i < 20; i++ {
				oper := ot.Operator{
					Ops:    []ot.OperatorAtomic{ot.NewInsert("x")},
					UserID: userID,
				}
				dc.applyEdit(&oper)
				dc.SetCursor(namedCursor(userID, uint64(i%5)))
				dc.SetTitle(userID, fmt.Sprintf("title-%d", i))
			}
		}(w)
	}

	wg.Wait()
}

func TestOperatorEncodeDecode(t *testing.T) {
	oper := &ot.Operator{Ops: []ot.OperatorAtomic{
		ot.NewRetain(2),
		ot.NewInsert("X"),
		ot.NewDelete(3),
	}}

	raw, err := encodeOperator(oper)
	if err != nil {
		t.Fatalf("encode operator: %v", err)
	}
	if got, want := string(raw), `[2,"X",-3]`; got != want {
		t.Fatalf("encoded operator = %s, want %s", got, want)
	}

	decoded, err := decodeOperator(raw)
	if err != nil {
		t.Fatalf("decode operator: %v", err)
	}

	base := []byte("abcde")
	if got, want := string(decoded.Apply(base)), string(oper.Apply(base)); got != want {
		t.Fatalf("decoded apply = %q, want %q", got, want)
	}
}

func TestComposeOperators(t *testing.T) {
	// "abc" -> "aXbc" -> "aXc"
	op1 := ot.Operator{Ops: []ot.OperatorAtomic{
		ot.NewRetain(1),
		ot.NewInsert("X"),
		ot.NewRetain(2),
	}}
	op2 := ot.Operator{Ops: []ot.OperatorAtomic{
		ot.NewRetain(2),
		ot.NewDelete(1),
		ot.NewRetain(1),
	}}

	composed, err := composeOperators([]ot.Operator{op1, op2})
	if err != nil {
		t.Fatalf("compose operators: %v", err)
	}
	if composed == nil {
		t.Fatal("compose operators returned nil")
	}

	base := []byte("abc")
	want := op2.Apply(op1.Apply(base))
	if got := composed.Apply(base); string(got) != string(want) {
		t.Fatalf("composed apply = %q, want %q", got, want)
	}
}

func TestComposeOperatorsEmpty(t *testing.T) {
	composed, err := composeOperators(nil)
	if err != nil {
		t.Fatalf("compose operators: %v", err)
	}
	if composed != nil {
		t.Fatalf("compose operators = %+v, want nil", composed)
	}
}
