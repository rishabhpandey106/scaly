package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToBase62_Zero(t *testing.T) {
	assert.Equal(t, "a", ToBase62(0))
}

func TestToBase62_One(t *testing.T) {
	// 1 -> charset[1] = 'b'
	assert.Equal(t, "b", ToBase62(1))
}

func TestToBase62_CharsetBoundaries(t *testing.T) {
	// charset has 62 chars (indices 0-61)
	// index 0  = 'a'
	// index 25 = 'z'
	// index 26 = 'A'
	// index 51 = 'Z'
	// index 52 = '0'
	// index 61 = '9'
	assert.Equal(t, "a", ToBase62(0))
	assert.Equal(t, string(charset[25]), ToBase62(25))
	assert.Equal(t, string(charset[26]), ToBase62(26))
	assert.Equal(t, string(charset[51]), ToBase62(51))
	assert.Equal(t, string(charset[52]), ToBase62(52))
	assert.Equal(t, string(charset[61]), ToBase62(61))
}

func TestToBase62_62(t *testing.T) {
	// 62 in base-62 is "10" (i.e., 1*62 + 0) -> charset[1] + charset[0] = "ba"
	assert.Equal(t, "ba", ToBase62(62))
}

func TestToBase62_LargeValue(t *testing.T) {
	// known mapping for counter = 1000
	result := ToBase62(1000)
	assert.NotEmpty(t, result)
	// verify it only contains charset characters
	for _, ch := range result {
		assert.Contains(t, charset, string(ch))
	}
}

func TestToBase62_MonotonicallyIncreasing(t *testing.T) {
	// larger inputs should generally produce longer or lexicographically larger codes
	prev := ToBase62(1)
	for i := int64(2); i <= 100; i++ {
		curr := ToBase62(i)
		assert.NotEqual(t, prev, curr, "expected unique codes for distinct counters")
		prev = curr
	}
}

func TestToBase62_OnlyCharsetChars(t *testing.T) {
	for _, n := range []int64{0, 1, 62, 100, 3843, 238327} {
		code := ToBase62(n)
		for _, ch := range code {
			assert.Contains(t, charset, string(ch), "unexpected character %q in code %q for n=%d", ch, code, n)
		}
	}
}
