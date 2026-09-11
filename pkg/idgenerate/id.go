package idgenerate

import (
	"fmt"
	"sync/atomic"
	"time"
)

var (
	sequence atomic.Int32

	lastGenerateTime time.Time
)

func init() {
	sequence.Store(0)
	lastGenerateTime = time.Now()
}

func NewID(serverID int64) string {
	if serverID >= 1e4 {
		panic("serverID out of range")
	}

	cur := time.Now()
	if cur.After(lastGenerateTime) {
		sequence.Store(0)
		lastGenerateTime = cur
		return fmt.Sprintf("0%014d%03d%04d", cur.Unix(), serverID, 0)
	} else if cur.Equal(lastGenerateTime) {
		seq := sequence.Add(1)
		return fmt.Sprintf("0%014d%03d%04d", cur.Unix(), serverID, seq)
	}
	panic("InvalidSystemClock")
}
