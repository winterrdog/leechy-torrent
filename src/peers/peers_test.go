package peers

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	type TestState struct {
		input  []byte
		output []Peer
		fails  bool
	}

	var tests = map[string]TestState{
		"correctly parses peers": {
			input: []byte{127, 0, 0, 1, 0x0, 0x50, 1, 1, 1, 1, 0x01, 0xbb},
			output: []Peer{
				{IP: net.IP{127, 0, 0, 1}, Port: 80},
				{IP: net.IP{1, 1, 1, 1}, Port: 443},
			},
			fails: false,
		},
		"not enough bytes in peers": {
			input:  []byte{127, 0, 0, 1, 0x00},
			output: nil,
			fails:  true,
		},
	}

	for _, test := range tests {
		peers, err := Unmarshal(test.input)
		if test.fails {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		assert.Equal(t, test.output, peers)
	}
}

func TestString(t *testing.T) {
	type TestState struct {
		input  Peer
		output string
	}

	var tests = [...]TestState{
		{
			input:  Peer{IP: net.IP{127, 0, 0, 1}, Port: 8080},
			output: "127.0.0.1:8080",
		},
	}

	for _, test := range tests {
		var s = test.input.String()
		assert.Equal(t, test.output, s)
	}
}
