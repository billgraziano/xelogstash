package seq

import (
	"math"
	"sync"
	"time"

	"github.com/mattheath/base62"
)

// Bit size of the components
const (
	BitLenTime     = 32
	BitLenSequence = 32
)

var seq struct {
	mux      *sync.Mutex
	sequence int
	ts       int
}

var epoch time.Time
var encoder *base62.Encoding

func init() {
	seq.mux = new(sync.Mutex)
	epoch = time.Date(2018, time.June, 8, 0, 0, 0, 0, time.UTC)
	seq.ts = getSeconds()
	encoder = base62.NewStdEncoding().Option(base62.Padding(12))
}

// Get returns the next sequence number
func Get() string {
	seq.mux.Lock()
	defer seq.mux.Unlock()

	ts := getSeconds()
	if ts > seq.ts {
		seq.ts = ts
		seq.sequence = 0
	}

	x := int64(seq.ts<<32 | seq.sequence)

	s := encoder.EncodeInt64(x)
	seq.sequence++
	return s
}

func getSeconds() int {
	d := time.Since(epoch)
	s := d.Seconds()
	if s > float64(math.MaxInt32) {
		return math.MaxInt32
	}
	return int(s)
}
