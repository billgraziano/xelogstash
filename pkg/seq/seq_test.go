package seq

import (
	"fmt"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	x := Get()
	if x == "" {
		t.Error("get returned no value")
	}
	fmt.Println("x:", x)
	fmt.Println(seq)
}

func Test2Get(t *testing.T) {
	x := Get()
	if x == "" {
		t.Error("get returned empty")
	}
	fmt.Println("x:", x)
	fmt.Println(seq)

	time.Sleep(2 * time.Second)
	y := Get()
	fmt.Println("y:", y)
	fmt.Println(seq)
}
