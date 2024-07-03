package peers

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

// Peer represents a peer on the Bittorrent network
type Peer struct {
	IP   net.IP // peer's IP address
	Port uint16 // peer's application port
}

// Unmarshal takes a byte slice representing binary data of peers and unmarshals it into a slice of Peer structs.
// Each Peer struct contains an IP address and a port number.
// The function returns the unmarshaled slice of Peer structs and an error if the input is malformed.
func Unmarshal(peersBin []byte) ([]Peer, error) {
	const peersSize = 6 // 4 for IP and 2 for Port
	var peersBinSize = len(peersBin)
	var numPeers = peersBinSize / peersSize

	if peersBinSize%peersSize != 0 {
		var err = fmt.Errorf("received malformed peers")
		return nil, err
	}

	var peers = make([]Peer, numPeers)

	var offset int
	for i := 0; i != numPeers; i++ {
		offset = i * peersSize

		peers[i].IP = net.IP(peersBin[offset : offset+4])
		offset += 4
		peers[i].Port = binary.BigEndian.Uint16(peersBin[offset : offset+2])
	}

	return peers, nil
}

// String returns a string representation of the Peer's IP address and port.
func (p *Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}
