package ot

import (
	"encoding/json"
	"reflect"
	"testing"
)

func newCursor(anchorOffset uint64, anchorAssoc int8, headOffset uint64, headAssoc int8) Cursor {
	var c Cursor
	c.Anchor.Offset = anchorOffset
	c.Anchor.Assoc = anchorAssoc
	c.Head.Offset = headOffset
	c.Head.Assoc = headAssoc
	return c
}

func assertCursor(t *testing.T, name string, got, want Cursor) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got anchor=%d/%d head=%d/%d, want anchor=%d/%d head=%d/%d",
			name,
			got.Anchor.Offset, got.Anchor.Assoc, got.Head.Offset, got.Head.Assoc,
			want.Anchor.Offset, want.Anchor.Assoc, want.Head.Offset, want.Head.Assoc)
	}
}

func TestCursorTransformRetain(t *testing.T) {
	oper := Operator{Ops: []OperatorAtomic{NewRetain(5)}}

	got := newCursor(3, 1, 3, 1)
	got.ApplyTransform(&oper)

	assertCursor(t, "retain keeps offsets", got, newCursor(3, 1, 3, 1))
}

func TestCursorTransformEmptyOperator(t *testing.T) {
	oper := Operator{}

	got := newCursor(4, -1, 7, 1)
	got.ApplyTransform(&oper)

	assertCursor(t, "empty operator keeps offsets", got, newCursor(4, -1, 7, 1))
}

func TestCursorTransformInsert(t *testing.T) {
	// insert at the very beginning of the document
	atStart := Operator{Ops: []OperatorAtomic{NewInsert("abc")}}

	// insert after 3 retained characters
	inMiddle := Operator{Ops: []OperatorAtomic{NewRetain(3), NewInsert("X")}}

	cases := []struct {
		name string
		oper Operator
		in   Cursor
		want Cursor
	}{
		{"at start keeps with left assoc", atStart, newCursor(0, -1, 0, -1), newCursor(0, -1, 0, -1)},
		{"at start shifts with right assoc", atStart, newCursor(0, 1, 0, 1), newCursor(3, 1, 3, 1)},
		{"at start shifts with neutral assoc", atStart, newCursor(0, 0, 0, 0), newCursor(3, 0, 3, 0)},
		{"after insertion shifts", atStart, newCursor(5, 1, 5, 1), newCursor(8, 1, 8, 1)},
		{"before insertion keeps", inMiddle, newCursor(2, 1, 2, 1), newCursor(2, 1, 2, 1)},
		{"at insertion left assoc keeps", inMiddle, newCursor(3, -1, 3, -1), newCursor(3, -1, 3, -1)},
		{"at insertion right assoc shifts", inMiddle, newCursor(3, 1, 3, 1), newCursor(4, 1, 4, 1)},
		{"after insertion shifts", inMiddle, newCursor(5, 1, 5, 1), newCursor(6, 1, 6, 1)},
	}

	for _, tc := range cases {
		got := tc.in
		got.ApplyTransform(&tc.oper)
		assertCursor(t, tc.name, got, tc.want)
	}
}

func TestCursorTransformDelete(t *testing.T) {
	// doc length 7: retain 2, delete 3 (region [2,5)), retain 2
	oper := Operator{Ops: []OperatorAtomic{
		NewRetain(2),
		NewDelete(3),
		NewRetain(2),
	}}

	cases := []struct {
		name string
		in   Cursor
		want Cursor
	}{
		{"before deletion keeps", newCursor(1, 1, 1, 1), newCursor(1, 1, 1, 1)},
		{"at deletion start keeps", newCursor(2, 1, 2, 1), newCursor(2, 1, 2, 1)},
		{"inside deletion collapses", newCursor(3, 1, 3, 1), newCursor(2, 1, 2, 1)},
		{"inside deletion (end) collapses", newCursor(4, 1, 4, 1), newCursor(2, 1, 2, 1)},
		{"at deletion end collapses", newCursor(5, 1, 5, 1), newCursor(2, 1, 2, 1)},
		{"after deletion shifts left", newCursor(6, 1, 6, 1), newCursor(3, 1, 3, 1)},
	}

	for _, tc := range cases {
		got := tc.in
		got.ApplyTransform(&oper)
		assertCursor(t, tc.name, got, tc.want)
	}
}

func TestCursorTransformMixed(t *testing.T) {
	// old: d0 d1 d2 d3 d4 d5 d6
	// new: d0 d1 X Y d2 d3 d6
	oper := Operator{Ops: []OperatorAtomic{
		NewRetain(2),
		NewInsert("XY"),
		NewRetain(2),
		NewDelete(2),
		NewRetain(1),
	}}

	cases := []struct {
		name string
		in   Cursor
		want Cursor
	}{
		{"insert at cursor, left assoc", newCursor(2, -1, 2, -1), newCursor(2, -1, 2, -1)},
		{"insert at cursor, right assoc", newCursor(2, 1, 2, 1), newCursor(4, 1, 4, 1)},
		{"cursor inside retained region", newCursor(3, 1, 3, 1), newCursor(5, 1, 5, 1)},
		{"cursor right before deletion", newCursor(4, 1, 4, 1), newCursor(6, 1, 6, 1)},
		{"cursor at deletion end", newCursor(6, 1, 6, 1), newCursor(6, 1, 6, 1)},
		{"selection collapses", newCursor(4, 1, 6, 1), newCursor(6, 1, 6, 1)},
		{"forward selection", newCursor(2, 1, 6, 1), newCursor(4, 1, 6, 1)},
	}

	for _, tc := range cases {
		got := tc.in
		got.ApplyTransform(&oper)
		assertCursor(t, tc.name, got, tc.want)
	}
}

func TestCursorTransformBackwardSelection(t *testing.T) {
	// cursor with different assoc on each endpoint: head before anchor.
	oper := Operator{Ops: []OperatorAtomic{NewRetain(2), NewInsert("X")}}

	got := newCursor(4, 1, 2, -1) // anchor after insertion point, head exactly at it
	got.ApplyTransform(&oper)

	assertCursor(t, "endpoints transformed independently", got, newCursor(5, 1, 2, -1))
}

func TestCursorUnmarshalJSON(t *testing.T) {
	t.Run("full", func(t *testing.T) {
		data := []byte(`{"user_id":"u1","reversion":3,"anchor":{"offset":5,"assoc":-1},"head":{"offset":8,"assoc":1}}`)
		var c Cursor
		if err := json.Unmarshal(data, &c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := newCursor(5, -1, 8, 1)
		want.UserID = "u1"
		want.Reversion = 3
		if c != want {
			t.Fatalf("got %+v, want %+v", c, want)
		}
	})

	t.Run("missing endpoints", func(t *testing.T) {
		var c Cursor
		if err := json.Unmarshal([]byte(`{"anchor":{"offset":2,"assoc":-1}}`), &c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Anchor.Offset != 2 || c.Anchor.Assoc != -1 || c.Head.Offset != 0 {
			t.Fatalf("unexpected cursor: %+v", c)
		}
	})

	t.Run("null endpoints", func(t *testing.T) {
		var c Cursor
		if err := json.Unmarshal([]byte(`{"anchor":null,"head":null}`), &c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c != (Cursor{}) {
			t.Fatalf("unexpected cursor: %+v", c)
		}
	})

	t.Run("negative offset is clamped", func(t *testing.T) {
		var c Cursor
		if err := json.Unmarshal([]byte(`{"anchor":{"offset":-4,"assoc":-1}}`), &c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Anchor.Offset != 0 {
			t.Fatalf("got offset %d, want 0", c.Anchor.Offset)
		}
	})

	t.Run("empty object", func(t *testing.T) {
		var c Cursor
		if err := json.Unmarshal([]byte(`{}`), &c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c != (Cursor{}) {
			t.Fatalf("unexpected cursor: %+v", c)
		}
	})

	t.Run("invalid endpoint", func(t *testing.T) {
		var c Cursor
		if err := json.Unmarshal([]byte(`{"anchor":{"offset":"nope"}}`), &c); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestCursorJSONRoundTrip(t *testing.T) {
	want := newCursor(1, -1, 4, 1)
	want.UserID = "user"
	want.Reversion = 9

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Cursor
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, want)
	}
}
