package ot

import "encoding/json"

// Cursor describes a caret or selection inside a document.
//
// Offset is a byte offset into the document text and therefore uses the same
// unit as Operator (see Insert.len). A selection has two endpoints: Anchor is
// where the selection started and Head is where it currently ends. For a plain
// caret both offsets are equal.
//
// Assoc ("association") decides what happens to an endpoint when another user
// inserts text exactly at its offset:
//
//	assoc <  0: the endpoint sticks to the character before the insertion, so
//	            the inserted text appears after it and the offset is kept.
//	assoc >= 0: the endpoint sticks to the character after the insertion, so
//	            the inserted text appears before it and the offset is moved
//	            past the inserted text.
type Cursor struct {
	UserID    string `json:"user_id"`
	Reversion uint32 `json:"reversion"`
	Anchor    struct {
		Offset uint64 `json:"offset"`
		Assoc  int8   `json:"assoc"`
	} `json:"anchor"`
	Head struct {
		Offset uint64 `json:"offset"`
		Assoc  int8   `json:"assoc"`
	} `json:"head"`
}

func (c *Cursor) UnmarshalJSON(data []byte) error {
	var raw struct {
		UserID    string          `json:"user_id"`
		Reversion uint32          `json:"reversion"`
		Anchor    json.RawMessage `json:"anchor"`
		Head      json.RawMessage `json:"head"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var (
		anchorOffset uint64
		anchorAssoc  int8
		headOffset   uint64
		headAssoc    int8
	)

	if err := unmarshalCursorPosition(raw.Anchor, &anchorOffset, &anchorAssoc); err != nil {
		return err
	}
	if err := unmarshalCursorPosition(raw.Head, &headOffset, &headAssoc); err != nil {
		return err
	}

	c.UserID = raw.UserID
	c.Reversion = raw.Reversion
	c.Anchor.Offset = anchorOffset
	c.Anchor.Assoc = anchorAssoc
	c.Head.Offset = headOffset
	c.Head.Assoc = headAssoc

	return nil
}

// unmarshalCursorPosition decodes a single selection endpoint. The offset is
// decoded as a signed integer so that negative values coming from a client do
// not fail on the unsigned target; they are clamped to zero instead.
func unmarshalCursorPosition(raw json.RawMessage, offset *uint64, assoc *int8) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var pos struct {
		Offset int64 `json:"offset"`
		Assoc  int8  `json:"assoc"`
	}
	if err := json.Unmarshal(raw, &pos); err != nil {
		return err
	}

	if pos.Offset < 0 {
		pos.Offset = 0
	}
	*offset = uint64(pos.Offset)
	*assoc = pos.Assoc

	return nil
}

// ApplyTransform moves the cursor through an operation so that it keeps pointing at
// the same logical position in the new document. It is used to rebase a remote
// cursor over the operations that have been applied in the meantime.
//
// The cursor is updated in place and both endpoints are transformed
// independently, so a backwards selection (head before anchor) is handled too.
func (c *Cursor) ApplyTransform(oper *Operator) {
	// idx is the number of characters already emitted into the new document,
	// i.e. the position at which the current operator takes effect.
	var idx int64

	it := newOperatorIterator(oper)
	for {
		if it.get() == nil {
			break
		}

		if ins, ok := it.getInsert(); ok {
			n := int64(ins.len())
			c.transformInsert(idx, n)
			idx += n
			it.next()
			continue
		}

		if ret, ok := it.getRetain(); ok {
			// Retained text keeps its offsets; just skip past it.
			idx += int64(ret.N)
			it.next()
			continue
		}

		if del, ok := it.getDelete(); ok {
			// Deleted text is not emitted, so idx stays where it is.
			c.transformDelete(idx, int64(del.N))
			it.next()
			continue
		}
	}
}

func (c *Cursor) transformInsert(idx, n int64) {
	c.Anchor.Offset = transformInsertOffset(c.Anchor.Offset, c.Anchor.Assoc, idx, n)
	c.Head.Offset = transformInsertOffset(c.Head.Offset, c.Head.Assoc, idx, n)
}

func (c *Cursor) transformDelete(idx, n int64) {
	c.Anchor.Offset = transformDeleteOffset(c.Anchor.Offset, idx, n)
	c.Head.Offset = transformDeleteOffset(c.Head.Offset, idx, n)
}

// transformInsertOffset shifts an offset that lies at or after an insertion of
// length n made at position idx. An offset exactly at idx only moves when the
// cursor is associated to the right (assoc >= 0).
func transformInsertOffset(offset uint64, assoc int8, idx, n int64) uint64 {
	pos := int64(offset)
	if pos > idx || (pos == idx && assoc >= 0) {
		pos += n
	}
	return uint64(pos)
}

// transformDeleteOffset collapses offsets that fall inside a deletion of length
// n made at position idx and shifts offsets that lie after it.
func transformDeleteOffset(offset uint64, idx, n int64) uint64 {
	pos := int64(offset)
	switch {
	case pos <= idx:
		// Before (or at the start of) the deletion: nothing changes.
		return offset
	case pos >= idx+n:
		// After the deletion: move back by the deleted length.
		return uint64(pos - n)
	default:
		// Inside the deletion: collapse to its start.
		return uint64(idx)
	}
}
