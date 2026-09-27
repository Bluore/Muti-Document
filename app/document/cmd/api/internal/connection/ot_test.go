package connection

import (
	"testing"

	"github.com/Bluore/mutidoc/pkg/ot"
)

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
	dc := NewOt()
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
	dc := NewOt()

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
	dc := NewOt()
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
	dc := NewOt()

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
	dc := NewOt()
	dc.Content = []byte("abcde")

	c := namedCursor("u1", 2)
	c.Reversion = 99
	dc.SetCursor(c) // must not panic

	if dc.Cursors["u1"].Anchor.Offset != 2 {
		t.Fatalf("offset = %d, want 2", dc.Cursors["u1"].Anchor.Offset)
	}
}
