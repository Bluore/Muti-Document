package idgenerate

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

var (
	sequence   int32
	sequenceMu sync.Mutex

	lastGenerateTime int64
)

func NewID(serverID int64) string {
	if serverID >= 1e4 {
		panic("serverID out of range")
	}

	sequenceMu.Lock()
	defer sequenceMu.Unlock()
	cur := time.Now().Unix()
	if cur > lastGenerateTime {
		sequence = 0
		lastGenerateTime = cur
		return fmt.Sprintf("0%014d%03d%04d", cur, serverID, 0)
	} else if cur == lastGenerateTime {
		sequence += 1
		return fmt.Sprintf("0%014d%03d%04d", cur, serverID, sequence)
	}
	panic("InvalidSystemClock")
}

func generateID(cur time.Time, serverID int64, seq int32) string {
	str := strconv.FormatInt(cur.Unix()*10^7+serverID*10^4+int64(seq), 10)
	return fmt.Sprintf("0%21s", str)
}
