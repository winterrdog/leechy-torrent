package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MessageId byte

const (
	MsgChoke         MessageId = iota // tells the peer that the client is choked( not allowed to request pieces from the peer)
	MsgUnchoke                        // tells the peer that the client is unchoked( allowed to request pieces from the peer)
	MsgInterested                     // tells the peer that the client is interested in the file the peer has
	MsgNotInterested                  // tells the peer that the client is not interested in the file the peer has
	MsgHave                           // announces that the peer has downloaded and validated a piece
	MsgBitfield                       // encodes which pieces the peer has and doesn't have where each piece is a bit in the bitfield
	MsgRequest                        // requests a piece of the file from the peer
	MsgPiece                          // contains a piece of the file requested
	MsgCancel                         // cancels a request sent to the peer. useful when a piece is no longer needed
)

// Message represents a message that can be sent to or received from a peer in the BitTorrent protocol.
type Message struct {
	Id      MessageId // message ID (1 byte)
	Payload []byte    // message payload (variable length)
}

// Serialize serializes a message into a buffer of the form
// <length prefix><message ID><payload>
// Interprets `nil` as a keep-alive message
func (msg *Message) Serialize() []byte {
	if msg == nil {
		// send a "keep-alive" message
		var payload [4]byte
		return payload[:]
	}

	var length = uint32(len(msg.Payload) + 1) // +1 for Id
	var buf = make([]byte, 4+length)          // +4 for length prefix

	// create bit stream to send to peer
	binary.BigEndian.PutUint32(buf[:4], length)
	buf[4] = byte(msg.Id)
	copy(buf[5:], msg.Payload)

	return buf
}

// String returns a string representation of the Message.
// If the Message is nil, it returns keep-alive.
// Otherwise, it returns the name of the Message followed by the length of the Payload.
func (msg *Message) String() string {
	if msg == nil {
		return msg.Name()
	}

	return fmt.Sprintf("%s [%d]", msg.Name(), len(msg.Payload))
}

// Name returns the name of the message based on its ID.
// If the message is nil, it returns "keep-alive".
// If the message ID is not recognized, it returns "unknown id: {ID}".
// Otherwise, it returns the name of the message.
func (msg *Message) Name() string {
	if msg == nil {
		return "keep-alive"
	}

	switch msg.Id {
	case MsgChoke:
		return "choke"
	case MsgUnchoke:
		return "unchoke"
	case MsgInterested:
		return "interested"
	case MsgNotInterested:
		return "not-interested"
	case MsgHave:
		return "have"
	case MsgBitfield:
		return "bitfield"
	case MsgRequest:
		return "request"
	case MsgPiece:
		return "piece"
	case MsgCancel:
		return "cancel"
	default:
		return fmt.Sprintf("unknown id: %d", msg.Id)
	}
}

// FormatRequestMsg formats a request message with the given index, begin, and length.
// It returns a pointer to a Message struct containing the formatted message.
// The payload of the message is a 12-byte buffer containing the index, begin, and length
// to be sent to the peer.
func FormatRequestMsg(index, begin, length int) *Message {
	var payload [(4 * 3)]byte

	binary.BigEndian.PutUint32(payload[:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:], uint32(length))

	return &Message{
		Id:      MsgRequest,
		Payload: payload[:],
	}
}

// FormatHave formats a "have" message with the given index.
// It returns a pointer to a Message struct containing the formatted message.
// The payload of the message is a 4-byte buffer containing the index to be sent to the peer.
func FormatHave(index int) *Message {
	var payload [4]byte

	binary.BigEndian.PutUint32(payload[:], uint32(index))
	return &Message{Id: MsgHave, Payload: payload[:]}
}

// ParsePiece parses a 'piece' message and extracts the piece index, begin offset, and data from the message payload.
// It verifies that the message ID is 'piece' and checks the payload length.
// If the parsed index does not match the expected index, or if the begin offset is beyond the buffer size,
// or if the data is too large for the buffer, an error is returned.
// Otherwise, the data is copied from the message payload to the data buffer and the length of the copied data is returned.
func ParsePiece(index int, dataBuf []byte, msg *Message) (int, error) {
	if msg.Id != MsgPiece {
		return 0, fmt.Errorf("expected 'piece' message, got %s with ID as %d", msg.Name(), msg.Id)
	}

	if !(len(msg.Payload) >= 8) {
		return 0, fmt.Errorf("expected payload length of at least 8, got %d", len(msg.Payload))
	}

	// parse piece index from message payload
	var parsedIndex = int(binary.BigEndian.Uint32(msg.Payload[:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("expected piece index %d, got %d instead", index, parsedIndex)
	}

	// parse begin offset from message payload
	var dataBufLen = len(dataBuf)
	var begin = int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= dataBufLen {
		return 0, fmt.Errorf("begin offset %d is way beyond buffer size %d", begin, dataBufLen)
	}

	// parse data from message payload
	var data = msg.Payload[8:]
	var dataLen = len(data)
	if begin+dataLen > dataBufLen {
		return 0, fmt.Errorf("data too large[%d] for offset %d with buffer size %d", dataLen, begin, dataBufLen)
	}

	// copy data from message payload to data buffer
	copy(dataBuf[begin:], data)

	return dataLen, nil
}

// ParseHave parses a 'have' message and returns the index of the piece that the sender has.
// It expects the message ID to be MsgHave and the payload length to be 4.
// If the message ID or payload length is not as expected, it returns an error.
func ParseHave(msg *Message) (int, error) {
	if msg.Id != MsgHave {
		return 0, fmt.Errorf("expected 'have' message, got %s with ID as %d", msg.Name(), msg.Id)
	}

	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("expected payload length of 4, got %d", len(msg.Payload))
	}

	var index = int(binary.BigEndian.Uint32(msg.Payload))

	return index, nil
}

// Read reads a message from the given reader and returns a pointer to the Message struct.
// It returns an error if there was a problem reading from the reader.
// If the message is a keep-alive message (payload length is 0), it returns nil.
// The function reads the payload length from the reader and then reads the message from the reader.
// The message ID is extracted from the first byte of the message buffer, and the remaining bytes are set as the payload.
func Read(reader io.Reader) (*Message, error) {
	var msg Message

	var payloadLenBuf [4]byte
	var _, err = io.ReadFull(reader, payloadLenBuf[:])
	if err != nil {
		return nil, err
	}

	var payloadLen = binary.BigEndian.Uint32(payloadLenBuf[:])
	if payloadLen == 0 {
		// keep-alive message
		return nil, nil
	}

	var msgBuf = make([]byte, payloadLen)
	_, err = io.ReadFull(reader, msgBuf)
	if err != nil {
		return nil, err
	}

	msg.Id = MessageId(msgBuf[0])
	msg.Payload = msgBuf[1:]

	return &msg, nil
}
