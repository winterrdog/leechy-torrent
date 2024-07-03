package bitfield

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasPiece(t *testing.T) {
	bf := Bitfield{0b01010100, 0b01010100}
	outputs := []bool{false, true, false, true, false, true, false, false, false, true, false, true, false, true, false, false, false, false, false, false}
	for i := 0; i < len(outputs); i++ {
		assert.Equal(t, outputs[i], bf.HasPiece(i))
	}

	// check if the 4th piece is present
	bf = Bitfield{0b00001000}
	assert.Equal(t, true, bf.HasPiece(4))
	assert.Equal(t, false, bf.HasPiece(5))
	assert.Equal(t, false, bf.HasPiece(8))
}

func TestSetPiece(t *testing.T) {
	var tests = []struct {
		input  Bitfield
		index  int
		output Bitfield
	}{
		{
			input:  Bitfield{0b01010100, 0b01010100},
			index:  4, //          v (set)
			output: Bitfield{0b01011100, 0b01010100},
		},
		{
			input:  Bitfield{0b01010100, 0b01010100},
			index:  9, //                   v (noop)
			output: Bitfield{0b01010100, 0b01010100},
		},
		{
			input:  Bitfield{0b01010100, 0b01010100},
			index:  15, //                        v (set)
			output: Bitfield{0b01010100, 0b01010101},
		},
		{
			input:  Bitfield{0b01010100, 0b01010100},
			index:  19, //                            v (noop)
			output: Bitfield{0b01010100, 0b01010100},
		},
	}
	for _, test := range tests {
		inputBitfield := test.input
		inputBitfield.SetPiece(test.index)
		assert.Equal(t, test.output, inputBitfield)
	}
}
