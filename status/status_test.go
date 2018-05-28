package status

import "testing"

func TestCheckDupe(t *testing.T) {
	var err error
	err = CheckDupe("a", "b", "c", "d")
	if err != nil {
		t.Error("It shouldn't exist")
	}
	err = CheckDupe("a", "b", "c", "d")
	if err == nil {
		t.Error("It should exist")
	}
}

func TestCaseSensitive(t *testing.T) {
	var err error
	err = CheckDupe("d", "d", "f", "G")
	if err != nil {
		t.Error("It shouldn't exist")
	}
	err = CheckDupe("D", "D", "F", "g")
	if err == nil {
		t.Error("It should exist")
	}
}
