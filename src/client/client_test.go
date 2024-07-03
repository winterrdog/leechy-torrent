package client

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/winterrdog/lean-bit-torrent-client/bitfield"
	"github.com/winterrdog/lean-bit-torrent-client/common"
	"github.com/winterrdog/lean-bit-torrent-client/handshake"
	"github.com/winterrdog/lean-bit-torrent-client/message"
	"github.com/winterrdog/lean-bit-torrent-client/peers"
)

func createClientAndServer(t *testing.T) (clientConn, serverConn net.Conn) {
	// ask the OS for an available ephemeral port by passing port 0
	var tcpListener, err = net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)

	// net.Dial does not block, so we need this signalling channel
	// to make sure we don't return before serverConn is ready
	var done = make(chan net.Conn)
	go func() {
		defer tcpListener.Close()

		conn, err := tcpListener.Accept()
		require.Nil(t, err)

		done <- conn
	}()

	// make client connection
	clientConn, _ = net.Dial("tcp", tcpListener.Addr().String())
	serverConn = <-done

	return
}

func TestNew(t *testing.T) {
	/*
		test cases:
		1. create a new valid client
		2. when a wrong infohash is provided
		3. when client receives a message that's not a bitfield message
		4. when fails to connect to peer
	*/

	// Start a mock server
	var listener, err = net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer listener.Close()

	var (
		peerPort = uint16(listener.Addr().(*net.TCPAddr).Port)
		peer     = &peers.Peer{IP: net.IP{127, 0, 0, 1}, Port: peerPort}
		infoHash = &common.Sha1Hash{
			0x01, 0x02, 0x03, 0x04, 0x05,
			0x06, 0x07, 0x08, 0x09, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15,
			0x16, 0x17, 0x18, 0x19, 0x20,
		}
		peerId = &common.Sha1Hash{
			0x20, 0x19, 0x18, 0x17, 0x16,
			0x15, 0x14, 0x13, 0x12, 0x11,
			0x10, 0x09, 0x08, 0x07, 0x06,
			0x05, 0x04, 0x03, 0x02, 0x01,
		}
		expected = &Client{
			Choked:   true,
			Bitfield: bitfield.Bitfield{0x00, 0x00, 0x00, 0x03, 0x02},
			Peer:     *peer,
			InfoHash: *infoHash,
			PeerId:   *peerId,
		}
	)

	t.Run("create a new valid client", func(t *testing.T) {
		var done = make(chan struct{})
		defer close(done)

		var bitfieldMsg = []byte{
			0x00, 0x00, 0x00, 0x06, 0x05, 0x00, 0x00, 0x00, 0x03, 0x02,
		}
		var runServer = func(tcpListener net.Listener) {
			serverConn, err := tcpListener.Accept()
			require.Nil(t, err)
			defer serverConn.Close()

			// Read the handshake message from the client
			buf := make([]byte, 48)
			_, err = serverConn.Read(buf)
			require.Nil(t, err)

			// Send a handshake response
			var handshakeResponse = handshake.New(infoHash, peerId).Serialize()
			_, err = serverConn.Write([]byte(handshakeResponse))
			require.Nil(t, err)

			// send a bitfield message
			_, err = serverConn.Write([]byte(bitfieldMsg))
			require.Nil(t, err)

			done <- struct{}{}
		}

		go runServer(listener)

		client, err := New(peer, peerId, infoHash)
		assert.Nil(t, err)
		assert.Equal(t, expected.Bitfield, client.Bitfield)
		assert.Equal(t, expected.Peer, client.Peer)
		assert.Equal(t, expected.InfoHash, client.InfoHash)
		assert.Equal(t, expected.PeerId, client.PeerId)
		assert.Equal(t, expected.Choked, client.Choked)

		// Ensure the goroutine completes
		<-done
	})

	/*
		test case 2: when a wrong infohash is provided
	*/
	t.Run("a wrong infohash is provided", func(t *testing.T) {
		var wrongInfoHash = &common.Sha1Hash{
			0x01, 0x02, 0x03, 0x04, 0x05,
			0x06, 0x07, 0x08, 0x09, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15,
			0x16, 0x17, 0x18, 0x19, 0x21,
		}

		var client, err = New(peer, peerId, wrongInfoHash)
		assert.NotNil(t, err)
		assert.Nil(t, client)
	})

	t.Run("client receives a message that's not a bitfield message", func(t *testing.T) {
		var listener, err = net.Listen("tcp", "127.0.0.1:0")
		require.Nil(t, err)
		defer listener.Close()

		peerPort = uint16(listener.Addr().(*net.TCPAddr).Port)
		peer = &peers.Peer{IP: net.IP{127, 0, 0, 1}, Port: peerPort}
		var nonBitfieldMsg = []byte{
			0x00, 0x00, 0x00, 0x05, 0x04, 0x00, 0x00, 0x00, 0x3,
		}
		var taskComplete = make(chan struct{})
		var runServerWithNonBitFieldMsg = func(tcpListener net.Listener) {
			defer close(taskComplete)

			var serverConn, err = tcpListener.Accept()
			require.Nil(t, err)
			defer serverConn.Close()

			// Read the handshake message from the client
			buf := make([]byte, 48)
			_, err = serverConn.Read(buf)
			require.Nil(t, err)

			// Send a handshake response
			var handshakeResponse = handshake.New(infoHash, peerId).Serialize()
			_, err = serverConn.Write([]byte(handshakeResponse))
			require.Nil(t, err)

			// send a bitfield message
			_, err = serverConn.Write([]byte(nonBitfieldMsg))
			require.Nil(t, err)

			taskComplete <- struct{}{}
		}

		go runServerWithNonBitFieldMsg(listener)

		client, err := New(peer, peerId, infoHash)
		assert.NotNil(t, err)
		assert.Nil(t, client)

		// Ensure the goroutine completes
		<-taskComplete
	})

	t.Run("fails to connect to peer", func(t *testing.T) {
		peer = &peers.Peer{IP: net.IP{127, 0, 0, 1}, Port: 12345}
		var client, err = New(peer, peerId, infoHash)

		assert.NotNil(t, err)
		assert.Nil(t, client)
	})
}

func TestRecvBitField(t *testing.T) {
	/*
		test cases:
		1. valid bitfield message
		2. malformed bitfield message
		3. read a message that is not a bitfield message
	*/

	// create client and server connections
	var clientConn, serverConn = createClientAndServer(t)
	defer clientConn.Close()
	defer serverConn.Close()

	var client = Client{Conn: clientConn}

	t.Run("valid bitfield message", func(t *testing.T) {
		msgBytes := []byte{0x00, 0x00, 0x00, 0x06, 0x05, 0x00, 0x00, 0x00, 0x03, 0x02}
		serverConn.Write(msgBytes)

		expected := bitfield.Bitfield{0x00, 0x00, 0x00, 0x03, 0x02}
		bitfield, err := RecvBitField(&client.Conn)
		assert.Nil(t, err)
		assert.Equal(t, expected, bitfield)
	})

	t.Run("malformed bitfield message", func(t *testing.T) {
		malformedMsgBytes := []byte{0x00, 0x00, 0x00, 0x06, 0x05, 0x00, 0x00, 0x00, 0x03}
		serverConn.Write(malformedMsgBytes)

		bitfield, err := RecvBitField(&client.Conn)
		assert.NotNil(t, err)
		assert.Nil(t, bitfield)
	})

	t.Run("read a message that is not a bitfield message", func(t *testing.T) {
		nonBitfieldMsg := []byte{0x00, 0x00, 0x00, 0x05, 0x04, 0x00, 0x00, 0x00, 0x3}
		serverConn.Write(nonBitfieldMsg)

		bitfield, err := RecvBitField(&client.Conn)
		assert.NotNil(t, err)
		assert.Nil(t, bitfield)
	})
}

func TestCompleteHandshake(t *testing.T) {
	/*
		test cases:
		1. valid handshake
		2. wrong infohash
		3. send only a part of the handshake
		4. send a handshake with wrong protocol name
		5. when the server does not respond and the deadline is reached
	*/

	// create client and server connections
	var clientConn, serverConn = createClientAndServer(t)
	defer clientConn.Close()
	defer serverConn.Close()

	var infoHash = common.Sha1Hash{
		0x01, 0x02, 0x03, 0x04, 0x05,
		0x06, 0x07, 0x08, 0x09, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15,
		0x16, 0x17, 0x18, 0x19, 0x20,
	}
	var peerId = common.Sha1Hash{
		0x20, 0x19, 0x18, 0x17, 0x16,
		0x15, 0x14, 0x13, 0x12, 0x11,
		0x10, 0x09, 0x08, 0x07, 0x06,
		0x05, 0x04, 0x03, 0x02, 0x01,
	}

	t.Run("valid handshake", func(t *testing.T) {
		// send correct infohash to the client
		serverConn.Write(handshake.New(&infoHash, &peerId).Serialize())

		var handshakeResponse, err = CompleteHandshake(&clientConn, &infoHash, &peerId)
		assert.Nil(t, err)
		assert.Equal(t, infoHash, handshakeResponse.InfoHash)
	})

	t.Run("wrong infohash", func(t *testing.T) {
		var wrongInfoHash = common.Sha1Hash{
			0x01, 0x02, 0x03, 0x04, 0x05,
			0x06, 0x07, 0x08, 0x09, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15,
			0x16, 0x17, 0x18, 0x19, 0x21,
		}

		// send wrong infohash to the client
		serverConn.Write(handshake.New(&wrongInfoHash, &peerId).Serialize())

		handshakeResponse, err := CompleteHandshake(&clientConn, &infoHash, &peerId)
		assert.NotNil(t, err)
		assert.Nil(t, handshakeResponse)
	})

	t.Run("send only a part of the handshake", func(t *testing.T) {
		serverConn.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00})

		handshakeResponse, err := CompleteHandshake(&clientConn, &infoHash, &peerId)
		assert.NotNil(t, err)
		assert.Nil(t, handshakeResponse)
	})

	t.Run("send a handshake with wrong protocol name", func(t *testing.T) {
		serverConn.Write([]byte{
			0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00,
		})

		handshakeResponse, err := CompleteHandshake(&clientConn, &infoHash, &peerId)
		assert.NotNil(t, err)
		assert.Nil(t, handshakeResponse)
	})

	t.Run("server does not respond and the deadline is reached", func(t *testing.T) {
		serverConn.Close()

		handshakeResponse, err := CompleteHandshake(&clientConn, &infoHash, &peerId)
		assert.NotNil(t, err)
		assert.Nil(t, handshakeResponse)
	})
}

func TestSendRequest(t *testing.T) {
	/*
		test cases:
		1. send a valid request message
		2. server receives a valid request message
		3. connection is closed before sending the message
	*/

	// create client and server connections
	var clientConn, serverConn = createClientAndServer(t)
	defer serverConn.Close()

	// test case 1: send a valid request message
	t.Run("send a valid request message", func(t *testing.T) {
		var client = Client{Conn: clientConn}
		var err = client.SendRequest(3, 5, 7)

		assert.Nil(t, err)
	})

	// test case 2: server receives a valid request message
	t.Run("server receives a valid request message", func(t *testing.T) {
		var expected = []byte{0x00, 0x00, 0x00, 0x0d, 0x06, 0x00, 0x00, 0x00, 0x3, 0x00, 0x00, 0x00, 0x5, 0x00, 0x00, 0x00, 0x7}
		var buf = make([]byte, len(expected))
		var _, err = serverConn.Read(buf)

		assert.Nil(t, err)
		assert.Equal(t, expected, buf)
	})

	// test case 3: connection is closed before sending the message
	t.Run("connection is closed before sending the message", func(t *testing.T) {
		clientConn.Close()

		var client = Client{Conn: clientConn}
		err := client.SendRequest(3, 5, 7)
		assert.NotNil(t, err)
	})
}

func TestRead(t *testing.T) {
	/*
		test cases:
		1. valid request message
		2. malformed request message
	*/

	// create client and server connections
	var clientConn, serverConn = createClientAndServer(t)
	defer clientConn.Close()
	defer serverConn.Close()

	var client = Client{Conn: clientConn}

	t.Run("valid request message", func(t *testing.T) {
		var msgBytes = []byte{0x00, 0x00, 0x00, 0x05, 0x04, 0x00, 0x00, 0x00, 0x3}
		serverConn.Write(msgBytes)

		var expected = &message.Message{
			Id:      message.MsgHave,
			Payload: []byte{0x00, 0x00, 0x00, 0x3},
		}
		var msg, err = client.Read()
		assert.Nil(t, err)
		assert.Equal(t, expected, msg)
	})

	t.Run("malformed request message", func(t *testing.T) {
		serverConn.Write([]byte{0x00, 0x00, 0x00, 0x05, 0x04, 0x00, 0x00, 0x00})

		msg, err := client.Read()
		assert.NotNil(t, err)
		assert.Nil(t, msg)
	})
}

func TestSendUnchoke(t *testing.T) {
	/*
		test cases:
		1. send a valid unchoke message
		2. connection is closed before sending the message
		3. server receives a valid unchoke message
	*/

	// create client and server connections
	var clientConn, serverConn = createClientAndServer(t)
	defer serverConn.Close()

	t.Run("send a valid unchoke message", func(t *testing.T) {
		var client = Client{Conn: clientConn}
		err := client.SendUnchoke()
		assert.Nil(t, err)
	})

	t.Run("server receives a valid unchoke message", func(t *testing.T) {
		expected := []byte{0x00, 0x00, 0x00, 0x01, 0x01}
		var buf = make([]byte, len(expected))
		_, err := serverConn.Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expected, buf)
	})

	t.Run("connection is closed before sending the message", func(t *testing.T) {
		clientConn.Close()
		var client = Client{Conn: clientConn}
		err := client.SendUnchoke()
		assert.NotNil(t, err)
	})
}

func TestSendNotInterested(t *testing.T) {
	/*
		test cases:
		1. send a valid not interested message
		2. connection is closed before sending the message
		3. server receives a valid not interested message
	*/

	// create client and server connections
	var clientConn, serverConn = createClientAndServer(t)
	defer serverConn.Close()

	t.Run("send a valid not interested message", func(t *testing.T) {
		var client = Client{Conn: clientConn}
		err := client.SendNotInterested()
		assert.Nil(t, err)
	})

	t.Run("server receives a valid not interested message", func(t *testing.T) {
		expected := []byte{0x00, 0x00, 0x00, 0x01, 0x03}
		var buf = make([]byte, len(expected))
		_, err := serverConn.Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expected, buf)
	})

	t.Run("connection is closed before sending the message", func(t *testing.T) {
		clientConn.Close()
		var client = Client{Conn: clientConn}
		err := client.SendNotInterested()
		assert.NotNil(t, err)
	})
}

func TestSendInterested(t *testing.T) {
	/*
		test cases:
		1. send a valid interested message
		2. connection is closed before sending the message
		3. server receives a valid interested message
	*/

	// create client and server connections
	var clientConn, serverConn = createClientAndServer(t)
	defer serverConn.Close()

	t.Run("send a valid interested message", func(t *testing.T) {
		var client = Client{Conn: clientConn}
		err := client.SendInterested()
		assert.Nil(t, err)
	})

	t.Run("server receives a valid interested message", func(t *testing.T) {
		expected := []byte{0x00, 0x00, 0x00, 0x01, 0x02}
		var buf = make([]byte, len(expected))
		_, err := serverConn.Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expected, buf)
	})

	t.Run("connection is closed before sending the message", func(t *testing.T) {
		clientConn.Close()
		var client = Client{Conn: clientConn}
		err := client.SendInterested()
		assert.NotNil(t, err)
	})
}

func TestSendHave(t *testing.T) {
	/*
		test cases:
		1. send a valid have message
		2. connection is closed before sending the message
		3. server receives a valid have message
	*/

	// create client and server connections
	var clientConn, serverConn = createClientAndServer(t)
	defer serverConn.Close()

	t.Run("send a valid have message", func(t *testing.T) {
		var client = Client{Conn: clientConn}
		err := client.SendHave(3)
		assert.Nil(t, err)
	})

	t.Run("server receives a valid have message", func(t *testing.T) {
		expected := []byte{0x00, 0x00, 0x00, 0x05, 0x04, 0x00, 0x00, 0x00, 0x3}
		var buf = make([]byte, len(expected))
		_, err := serverConn.Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expected, buf)
	})

	t.Run("connection is closed before sending the message", func(t *testing.T) {
		clientConn.Close()
		var client = Client{Conn: clientConn}
		err := client.SendHave(5)
		assert.NotNil(t, err)
	})
}
