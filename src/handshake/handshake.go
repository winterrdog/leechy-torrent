package handshake

import (
	"fmt"
	"io"

	"github.com/winterrdog/lean-bit-torrent-client/common"
)

type Handshake struct {
	Pstr     string
	InfoHash common.Sha1Hash
	PeerId   common.Sha1Hash
}

func New(infoHash, peerId *common.Sha1Hash) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: *infoHash,
		PeerId:   *peerId,
	}
}

func (hs *Handshake) Serialize() []byte {
	var bufLen = len(hs.Pstr) + 49 // 20 + 20 + 8 + 1 = 49
	var buf = make([]byte, bufLen)

	// len of protocol ID
	buf[0] = byte(len(hs.Pstr))

	// protocol ID
	var offset = 1
	offset += copy(buf[offset:], []byte(hs.Pstr))

	// reserved 8 bytes -- for extensions
	var reservedBytes = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
	offset += copy(buf[offset:], reservedBytes[:])

	// info hash
	offset += copy(buf[offset:], hs.InfoHash[:])

	// peer id
	offset += copy(buf[offset:], hs.PeerId[:])

	return buf
}

func Read(reader io.Reader) (*Handshake, error) {
	var hs = Handshake{}

	// get protocol ID length from wire
	var rawProtocolIdLen [1]byte
	var _, err = io.ReadFull(reader, rawProtocolIdLen[:])
	if err != nil {
		return nil, err
	}

	var protocolIdLength = int(rawProtocolIdLen[0])
	if protocolIdLength == 0 {
		err = fmt.Errorf("protocol id string length cannot be 0")
		return nil, err
	}

	// read in the rest of the Bittorrent response from wire
	var handshakeBuf = make([]byte, 48+protocolIdLength)
	_, err = io.ReadFull(reader, handshakeBuf)
	if err != nil {
		return nil, err
	}

	// get protocol ID
	var protocolIdStr = string(handshakeBuf[0:protocolIdLength])

	// get the info hash and peer id
	var offset = protocolIdLength + 8
	var infoHash, peerId common.Sha1Hash
	offset += copy(infoHash[:], handshakeBuf[offset:offset+20])
	copy(peerId[:], handshakeBuf[offset:offset+20])

	// setup the handshake values
	hs.InfoHash = infoHash
	hs.PeerId = peerId
	hs.Pstr = protocolIdStr

	return &hs, nil
}
