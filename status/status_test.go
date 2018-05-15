package status

import "testing"

func TestCheck(t *testing.T) {
	var err error
	err = Check("a", "b", "c", "d")
	if err != nil {
		t.Error("It shouldn't exist")
	}
	err = Check("a", "b", "c", "d")
	if err == nil {
		t.Error("It should exist")
	}
}

func TestCaseSensitive(t *testing.T) {
	var err error
	err = Check("d", "d", "f", "G")
	if err != nil {
		t.Error("It shouldn't exist")
	}
	err = Check("D", "D", "F", "g")
	if err == nil {
		t.Error("It should exist")
	}
}
