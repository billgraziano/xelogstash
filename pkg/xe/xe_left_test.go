package xe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLeft(t *testing.T) {
	assert := assert.New(t)
	type testCase struct {
		got  string
		n    int
		want string
	}
	tests := []testCase{
		{"😢✅👌❤", 5, "😢✅👌❤"},
		{"😢✅👌❤", 4, "😢✅👌❤"},
		{"😢✅👌❤", 3, "😢✅👌..."},
		{"😢✅👌❤", 2, "😢✅..."},
		{"😢✅👌❤", 1, "😢..."},
		{"ăabcdef", 3, "ăab..."},
		{"ăabcdef", 200, "ăabcdef"},
		{"ăabcdef", 0, ""},
		{"ăabcdef", -1, "ăabcdef"},
		{"abcdef", 3, "abc..."},
		{"abcdef", 200, "abcdef"},
		{"ăabcdef", 7, "ăabcdef"},
		{"H㐀〾▓朗퐭텟şüöžåйкл¤", 2, "H㐀..."},
		{"H㐀〾▓朗퐭텟şüöžåйкл¤", 15, "H㐀〾▓朗퐭텟şüöžåйкл..."},
		{"H㐀〾▓朗퐭텟şüöžåйкл¤", 16, "H㐀〾▓朗퐭텟şüöžåйкл¤"},
		{"H㐀〾▓朗퐭텟şüöžåйкл¤", 17, "H㐀〾▓朗퐭텟şüöžåйкл¤"},
		{"日本\x80語", 2, "日本..."},        //x80 is an illegal UTF8 encoding
		{"日本\x80語", 3, "日本\x80..."},    //x80 is an illegal UTF8 encoding
		{"日本\x80語", 4, "日本\x80語"},      //x80 is an illegal UTF8 encoding
		{"日本\x80語", 5, "日本\x80語"},      //x80 is an illegal UTF8 encoding
		{"日本\x80語㐀〾", 4, "日本\x80語..."}, //x80 is an illegal UTF8 encoding
		{"日本\x80語㐀〾", 40, "日本\x80語㐀〾"}, //x80 is an illegal UTF8 encoding
	}

	for _, tc := range tests {
		result := left(tc.got, tc.n, "...")
		assert.Equal(tc.want, result)
	}
}
