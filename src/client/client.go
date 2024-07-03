package client

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/winterrdog/lean-bit-torrent-client/bitfield"
	"github.com/winterrdog/lean-bit-torrent-client/common"
	"github.com/winterrdog/lean-bit-torrent-client/handshake"
	"github.com/winterrdog/lean-bit-torrent-client/message"
	"github.com/winterrdog/lean-bit-torrent-client/peers"
)

// Client represents a BitTorrent client.
type Client struct {
	Conn     net.Conn          // connection to the peer
	Choked   bool              // whether the client is choked by the peer. If true, the client cannot request pieces from the peer
	Bitfield bitfield.Bitfield // bitfield representing the pieces the client has
	Peer     peers.Peer        // peer information
	InfoHash common.Sha1Hash   // infohash of the torrent
	PeerId   common.Sha1Hash   // peer ID
}

// CompleteHandshake performs a complete handshake with a BitTorrent peer.
// It sends a handshake request to the peer and reads the handshake response.
// The function checks if the infohash in the response matches the provided infohash.
// If successful, it returns the handshake response.
// If there is an error during the handshake process, it returns an error.
func CompleteHandshake(conn *net.Conn, infoHash, peerId *common.Sha1Hash) (*handshake.Handshake, error) {
	var pConn = *conn

	pConn.SetDeadline(time.Now().Add(3 * time.Second))
	defer pConn.SetDeadline(time.Time{}) // disable deadline after this scope

	// send handshake request
	var req = handshake.New(infoHash, peerId)
	var _, err = pConn.Write(req.Serialize())
	if err != nil {
		return nil, err
	}

	// read handshake response i.e. what the peer sends back
	var handshakeResponse *handshake.Handshake
	handshakeResponse, err = handshake.Read(pConn)
	if err != nil {
		return nil, err
	}

	// check if infohash matches
	if !bytes.Equal(handshakeResponse.InfoHash[:], (*infoHash)[:]) {
		return nil, fmt.Errorf("expected infohash %x but got %x", handshakeResponse.InfoHash, *infoHash)
	}

	return handshakeResponse, nil
}

// RecvBitField receives a bitfield message from the provided network connection.
// It reads the message from the connection and verifies that it is a 'bitfield' message.
// If successful, it returns the bitfield payload.
// If an error occurs while reading or if the received message is not a 'bitfield' message,
// it returns an error.
func RecvBitField(conn *net.Conn) (bitfield.Bitfield, error) {
	var pConn = *conn

	pConn.SetDeadline(time.Now().Add(3 * time.Second))
	defer pConn.SetDeadline(time.Time{}) // disable deadline

	var msg, err = message.Read(pConn)
	if err != nil {
		return nil, err
	}

	if msg.Id != message.MsgBitfield {
		return nil, fmt.Errorf("expected 'bitfield' message, got %s", msg.Name())
	}

	return msg.Payload, nil
}

// New creates a new client for a BitTorrent peer connection.
// It takes a `peer` object representing the peer to connect to,
// `peerId` and `infoHash` representing the peer ID and info hash respectively.
// It returns a pointer to the created `Client` object and an error, if any.
// The function establishes a TCP connection with the peer, completes the handshake,
// receives the bitfield from the peer, and creates the client with the necessary information.
// If any error occurs during the process, the function cleans up and returns the error.
func New(peer *peers.Peer, peerId, infoHash *common.Sha1Hash) (*Client, error) {
	var bf bitfield.Bitfield

	// connect to peer
	var conn, err = net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}

	// complete handshake
	_, err = CompleteHandshake(&conn, infoHash, peerId)
	if err != nil {
		goto cleanup
	}

	// receive bitfield from peer to know which pieces it has
	bf, err = RecvBitField(&conn)
	if err != nil {
		goto cleanup
	}

	// create client for peer connection
	return &Client{
		Conn:     conn,
		Choked:   true,
		Bitfield: bf,
		Peer:     *peer,
		InfoHash: *infoHash,
		PeerId:   *peerId,
	}, nil

cleanup:
	conn.Close()
	return nil, err
}

// Read reads a message from the client's connection.
// It returns the read message and any error encountered.
func (client *Client) Read() (*message.Message, error) {
	// avoid blocking forever which can happen
	// if the connection is not closed or the peer is not responding
	client.Conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer client.Conn.SetDeadline(time.Time{})

	var msg, err = message.Read(client.Conn)
	return msg, err
}

// SendRequest sends a request message to the connected peer with the specified index, begin, and length.
// It returns an error if there was a problem sending the request.
func (client *Client) SendRequest(index, begin, length int) error {
	var req = message.FormatRequestMsg(index, begin, length)
	var _, err = client.Conn.Write(req.Serialize())

	return err
}

// SendHave sends a "have" message to the connected peer, indicating
// that the client has a particular piece of the file.
// It takes an index parameter specifying the index of the piece.
// Returns an error if there was a problem sending the message.
func (client *Client) SendHave(index int) error {
	var msg = message.FormatHave(index)
	var _, err = client.Conn.Write(msg.Serialize())

	return err
}

// SendInterested sends an "Interested" message to the connected peer.
// It serializes the message and writes it to the client's connection.
// Returns an error if there was a problem writing the message.
func (client *Client) SendInterested() error {
	var msg = message.Message{Id: message.MsgInterested}
	var _, err = client.Conn.Write(msg.Serialize())

	return err
}

// SendNotInterested sends a "Not Interested" message to the connected peer.
// It serializes the message and writes it to the client's connection.
// Returns an error if there was a problem writing the message.
func (client *Client) SendNotInterested() error {
	var msg = message.Message{Id: message.MsgNotInterested}
	var _, err = client.Conn.Write(msg.Serialize())

	return err
}

// SendUnchoke sends an unchoke message to the connected peer.
// An unchoke message is used to inform the peer that it is allowed to request pieces from the client.
// Returns an error if there was a problem sending the message.
func (client *Client) SendUnchoke() error {
	var msg = message.Message{Id: message.MsgUnchoke}
	var _, err = client.Conn.Write(msg.Serialize())

	return err
}
