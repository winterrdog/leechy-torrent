package message

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatRequestMsg(t *testing.T) {
	msg := FormatRequestMsg(4, 567, 4321)
	expected := &Message{
		Id: MsgRequest,
		Payload: []byte{
			0x0, 0x0, 0x0, 0x4, // index
			0x0, 0x0, 0x02, 0x37, // begin
			0x0, 0x0, 0x10, 0xe1, // length
		},
	}

	assert.Equal(t, expected, msg)
}

func TestFormatHave(t *testing.T) {
	msg := FormatHave(123)
	expected := &Message{
		Id:      MsgHave,
		Payload: []byte{0x0, 0x0, 0x0, 0x7b},
	}

	assert.Equal(t, expected, msg)
}

func TestParsePiece(t *testing.T) {
	t.Run("Invalid Message ID", func(t *testing.T) {
		msg := &Message{Id: MsgChoke}
		_, err := ParsePiece(0, nil, msg)
		assert.EqualError(t, err, "expected 'piece' message, got choke with ID as 0")
	})

	t.Run("Invalid Payload Length", func(t *testing.T) {
		msg := &Message{Id: MsgPiece, Payload: []byte{0x0, 0x0, 0x0, 0x0}}
		_, err := ParsePiece(0, nil, msg)
		assert.EqualError(t, err, "expected payload length of at least 8, got 4")
	})

	t.Run("Valid Message", func(t *testing.T) {
		msg := &Message{
			Id: MsgPiece,
			Payload: []byte{
				0x0, 0x0, 0x0, 0x0, // index
				0x0, 0x0, 0x0, 0x0, // begin
				0x0, 0x0, 0x0, 0x0, // data
			},
		}
		n, err := ParsePiece(0, make([]byte, 10), msg)
		assert.NoError(t, err)
		assert.Equal(t, 4, n)
	})

	t.Run("Invalid Piece Index", func(t *testing.T) {
		msg := &Message{
			Id: MsgPiece,
			Payload: []byte{
				0x0, 0x0, 0x0, 0x1, // index is 1 but expected 0
				0x0, 0x0, 0x0, 0x0, // begin
				0x0, 0x0, 0x0, 0x0, // data
			},
		}
		_, err := ParsePiece(0, nil, msg)
		assert.EqualError(t, err, "expected piece index 0, got 1 instead")
	})

	t.Run("Payload Too Short", func(t *testing.T) {
		msg := &Message{
			Id:      MsgPiece,
			Payload: []byte{0x0, 0x0, 0x0, 0x0},
		}
		_, err := ParsePiece(0, nil, msg)
		assert.EqualError(t, err, "expected payload length of at least 8, got 4")
	})

	t.Run("Begin Offset Beyond Buffer Size", func(t *testing.T) {
		msg := &Message{
			Id: MsgPiece,
			Payload: []byte{
				0x0, 0x0, 0x0, 0x0, // index
				0x0, 0x0, 0x0, 0x1, // begin is 1 but buffer size is 0
				0x0, 0x0, 0x0, 0x0, // data
			},
		}
		_, err := ParsePiece(0, nil, msg)
		assert.EqualError(t, err, "begin offset 1 is way beyond buffer size 0")
	})

	t.Run("Offset Ok but Data Too Large", func(t *testing.T) {
		msg := &Message{
			Id: MsgPiece,
			Payload: []byte{
				0x00, 0x00, 0x00, 0x04, // Index is 6, not 4
				0x00, 0x00, 0x00, 0x02, // Begin is ok
				// Block is 10 long but begin=2; too long for input buffer
				0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x0a, 0x0b, 0x0c, 0x0d,
			},
		}
		_, err := ParsePiece(4, make([]byte, 5), msg)
		assert.EqualError(t, err, "data too large[10] for offset 2 with buffer size 5")
	})
}

func TestString(t *testing.T) {
	/*
		for each message type, test that the string representation and payload length are as expected
	*/
	type TestState struct {
		msg      *Message
		expected string
	}
	tests := []TestState{
		{
			msg:      &Message{Id: MsgChoke, Payload: nil},
			expected: "choke [0]",
		},
		{
			msg:      &Message{Id: MsgUnchoke, Payload: nil},
			expected: "unchoke [0]",
		},
		{
			msg:      &Message{Id: MsgInterested, Payload: nil},
			expected: "interested [0]",
		},
		{
			msg:      &Message{Id: MsgNotInterested, Payload: nil},
			expected: "not-interested [0]",
		},
		{
			msg:      &Message{Id: MsgHave, Payload: []byte{0x0, 0x0, 0x0, 0x1}},
			expected: "have [4]",
		},
		{
			msg:      &Message{Id: MsgBitfield, Payload: []byte{0x0, 0x0, 0x0, 0x1, 0x0}},
			expected: "bitfield [5]",
		},
		{
			msg:      &Message{Id: MsgRequest, Payload: []byte{0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x3}},
			expected: "request [12]",
		},
		{
			msg:      &Message{Id: MsgPiece, Payload: []byte{0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x3}},
			expected: "piece [12]",
		},
		{
			msg:      &Message{Id: MsgCancel, Payload: []byte{0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x3}},
			expected: "cancel [12]",
		},
		{
			msg:      &Message{Id: 0x0f, Payload: []byte{0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x3}},
			expected: "unknown id: 15 [12]",
		},
		{
			msg:      nil,
			expected: "keep-alive",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.msg.String())
	}
}

func TestSerialize(t *testing.T) {
	t.Run("Keep-Alive Message", func(t *testing.T) {
		msg := (*Message)(nil)
		expected := []byte{0x0, 0x0, 0x0, 0x0}
		assert.Equal(t, expected, msg.Serialize())
	})

	t.Run("Valid Message", func(t *testing.T) {
		msg := &Message{
			Id:      MsgRequest,
			Payload: []byte{0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x3},
		}
		expected := []byte{
			0x0, 0x0, 0x0, 0xd, // length prefix i.e. 1 + 4 + 4 + 4 = 13
			0x6,                // ID
			0x0, 0x0, 0x0, 0x1, // index
			0x0, 0x0, 0x0, 0x2, // begin
			0x0, 0x0, 0x0, 0x3, // length
		}
		assert.Equal(t, expected, msg.Serialize())
	})
}

func TestRead(t *testing.T) {
	t.Run("Keep-Alive Message", func(t *testing.T) {
		data := []byte{0x0, 0x0, 0x0, 0x0}
		reader := bytes.NewReader(data)

		msg, err := Read(reader)
		assert.NoError(t, err)
		assert.Equal(t, (*Message)(nil), msg)
	})

	t.Run("Valid Message", func(t *testing.T) {
		data := []byte{
			0x0, 0x0, 0x0, 0xd, // length prefix i.e. 1 + 4 + 4 + 4 = 13
			0x6,                // ID
			0x0, 0x0, 0x0, 0x1, // index
			0x0, 0x0, 0x0, 0x2, // begin
			0x0, 0x0, 0x0, 0x3, // length
		}
		reader := bytes.NewReader(data)

		msg, err := Read(reader)
		assert.NoError(t, err)
		assert.Equal(t, MsgRequest, msg.Id)
		assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x3}, msg.Payload)
	})

	t.Run("Length Prefix Too Short", func(t *testing.T) {
		data := []byte{0x0, 0x0, 0x0}
		reader := bytes.NewReader(data)

		msg, err := Read(reader)
		assert.Error(t, err)
		assert.Nil(t, msg)
	})

	t.Run("Buffer Too Short for Length Prefix", func(t *testing.T) {
		data := []byte{0x0, 0x0, 0x0, 0xd}
		reader := bytes.NewReader(data)

		msg, err := Read(reader)
		assert.Error(t, err)
		assert.Nil(t, msg)
	})
}

func TestParseHave(t *testing.T) {
	t.Run("Invalid Message ID", func(t *testing.T) {
		msg := &Message{Id: MsgChoke}
		n, err := ParseHave(msg)
		assert.Equal(t, 0, n)
		assert.EqualError(t, err, "expected 'have' message, got choke with ID as 0")
	})

	t.Run("Payload Length Too Short", func(t *testing.T) {
		msg := &Message{Id: MsgHave, Payload: []byte{0x0, 0x0, 0x0}}
		n, err := ParseHave(msg)
		assert.Equal(t, 0, n)
		assert.EqualError(t, err, "expected payload length of 4, got 3")
	})

	t.Run("Valid Message", func(t *testing.T) {
		msg := &Message{Id: MsgHave, Payload: []byte{0x0, 0x0, 0x0, 0x1}}
		n, err := ParseHave(msg)
		assert.Equal(t, 1, n)
		assert.NoError(t, err)
	})

	t.Run("Payload Length Too Long", func(t *testing.T) {
		msg := &Message{Id: MsgHave, Payload: []byte{0x0, 0x0, 0x0, 0x1, 0x0}}
		n, err := ParseHave(msg)
		assert.Equal(t, 0, n)
		assert.EqualError(t, err, "expected payload length of 4, got 5")
	})
}
