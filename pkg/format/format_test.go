package format

import (
	"fmt"
	"testing"
	"time"
)

func TestRoundDuration(t *testing.T) {
	//var d time.Duration
	s := time.Now()
	d := s.Add(time.Hour * 3).Sub(s)
	// if err != nil {
	// 	t.Error(err)
	// }
	if d < time.Second*10 {
		t.Error("failed")
	}
	fmt.Printf("1 hour: %v  ==>  %v\r\n", d, RoundDuration(d, 1*time.Hour))

}
