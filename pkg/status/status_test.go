package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestDupInstance(t *testing.T) {
	assert := assert.New(t)
	var err error
	err = CheckDupeInstance("WORK", "TEST\\GO")
	assert.NoError(err)
	err = CheckDupeInstance("WORK2", "TEST\\GO")
	assert.NoError(err)
	err = CheckDupeInstance("WORK2", "TEST\\GO")
	assert.Error(err)
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
