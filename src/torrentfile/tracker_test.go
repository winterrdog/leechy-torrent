package torrentfile

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/winterrdog/lean-bit-torrent-client/peers"
)

func TestBuildTrackerUrl(t *testing.T) {
	/*
		test cases:
		1. valid announce URL with valid info_hash, peer_id, port, uploaded, downloaded, compact, and left
		2. when an invalid announce URL is provided
	*/

	t.Run("valid announce URL with valid info_hash, peer_id, port, uploaded, downloaded, compact, and left", func(t *testing.T) {
		// create a new TorrentFile
		var torrFile = TorrentFile{
			Name:     "debian-10.2.0-amd64-netinst.iso",
			Length:   351272960,
			Announce: "http://bttracker.debian.org:6969/announce",
			InfoHash: [20]byte{216, 247, 57, 206, 195, 40, 149, 108, 204, 91, 191, 31, 134, 217, 253, 207, 219, 168, 206, 182},
			PiecesHashes: [][20]byte{
				{49, 50, 51, 52, 53, 54, 55, 56, 57, 48, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106},
				{97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 49, 50, 51, 52, 53, 54, 55, 56, 57, 48},
			},
			PieceLength: 262144,
		}
		var peerId = [20]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
		const port uint16 = 6789

		// call the BuildTrackerUrl method
		url, err := torrFile.BuildTrackerUrl(peerId, port)
		var expected = "http://bttracker.debian.org:6969/announce?compact=1&downloaded=0&info_hash=%D8%F79%CE%C3%28%95l%CC%5B%BF%1F%86%D9%FD%CF%DB%A8%CE%B6&left=351272960&peer_id=%00%01%02%03%04%05%06%07%08%09%0A%0B%0C%0D%0E%0F%10%11%12%13&port=6789&uploaded=0"

		assert.Nil(t, err)
		assert.Equal(t, expected, url)
	})

	t.Run("when an invalid announce URL is provided", func(t *testing.T) {
		// create a new TorrentFile
		var torrFile = TorrentFile{
			Name:     "debian-10.2.0-amd64-netinst.iso",
			Length:   351272960,
			InfoHash: [20]byte{216, 247, 57, 206, 195, 40, 149, 108, 204, 91, 191, 31, 134, 217, 253, 207, 219, 168, 206, 182},
			PiecesHashes: [][20]byte{
				{49, 50, 51, 52, 53, 54, 55, 56, 57, 48, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106},
				{97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 49, 50, 51, 52, 53, 54, 55, 56, 57, 48},
			},
			PieceLength: 262144,
		}
		var peerId = [20]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
		const port uint16 = 6789

		// call the BuildTrackerUrl method
		torrFile.Announce = "://example.com/invalid|path"
		url, err := torrFile.BuildTrackerUrl(peerId, port)

		assert.NotNil(t, err)
		assert.Empty(t, url)
	})
}

func TestRequestPeers(t *testing.T) {
	/*
		test cases:
		1. valid tracker response with valid peers
		2. when a malformed announce URL is provided
		3. when an error occurs while sending the HTTP GET request
		4. when an error occurs while unmarshaling the tracker response
	*/

	t.Run("valid tracker response with valid peers", func(t *testing.T) {
		var reqHandler = func(w http.ResponseWriter, r *http.Request) {
			var response = []byte(
				"d" +
					"8:interval" + "i1900e" +
					"5:peers" + "12:" +
					string([]byte{
						192, 168, 1, 1, 0x1A, 0x1B, // 0x1a1b = 6683
						198, 51, 0, 1, 0x1A, 0x1B, // 0x1a1b = 6683
					}) + "e",
			)

			w.Write(response)
		}
		var mockServer = httptest.NewServer(http.HandlerFunc(reqHandler))
		defer mockServer.Close()

		var torrFile = TorrentFile{
			Name:     "debian-10.2.0-amd64-netinst.iso",
			Length:   351272960,
			Announce: mockServer.URL,
			InfoHash: [20]byte{216, 247, 57, 206, 195, 40, 149, 108, 204, 91, 191, 31, 134, 217, 253, 207, 219, 168, 206, 182},
			PiecesHashes: [][20]byte{
				{49, 50, 51, 52, 53, 54, 55, 56, 57, 48, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106},
				{97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 49, 50, 51, 52, 53, 54, 55, 56, 57, 48},
			},
			PieceLength: 262144,
		}
		var peerId = [20]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
		const port uint16 = 6789

		// call the RequestPeers method
		ps, err := torrFile.RequestPeers(peerId, port)
		var expected = []peers.Peer{
			{IP: net.IP{192, 168, 1, 1}, Port: 0x1a1b},
			{IP: net.IP{198, 51, 0, 1}, Port: 0x1a1b},
		}

		assert.Nil(t, err)
		assert.Equal(t, expected, ps)
	})

	t.Run("when an error occurs while unmarshaling the tracker response", func(t *testing.T) {
		var reqHandler = func(w http.ResponseWriter, r *http.Request) {
			// send an invalid response
			w.Write([]byte("clearly invalid response :)"))
		}
		var mockServer = httptest.NewServer(http.HandlerFunc(reqHandler))
		defer mockServer.Close()

		var torrFile = TorrentFile{
			Name:     "debian-10.2.0-amd64-netinst.iso",
			Length:   351272960,
			Announce: mockServer.URL,
			InfoHash: [20]byte{216, 247, 57, 206, 195, 40, 149, 108, 204, 91, 191, 31, 134, 217, 253, 207, 219, 168, 206, 182},
			PiecesHashes: [][20]byte{
				{49, 50, 51, 52, 53, 54, 55, 56, 57, 48, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106},
				{97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 49, 50, 51, 52, 53, 54, 55, 56, 57, 48},
			},
			PieceLength: 262144,
		}
		var peerId = [20]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
		const port uint16 = 6789

		// call the RequestPeers method
		ps, err := torrFile.RequestPeers(peerId, port)

		assert.NotNil(t, err)
		assert.Empty(t, ps)
	})

	t.Run("when a malformed announce URL is provided", func(t *testing.T) {
		var torrFile = TorrentFile{
			Name:     "debian-10.2.0-amd64-netinst.iso",
			Length:   351272960,
			Announce: "://example.com/invalid|path", // path is invalid
			InfoHash: [20]byte{216, 247, 57, 206, 195, 40, 149, 108, 204, 91, 191, 31, 134, 217, 253, 207, 219, 168, 206, 182},
			PiecesHashes: [][20]byte{
				{49, 50, 51, 52, 53, 54, 55, 56, 57, 48, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106},
				{97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 49, 50, 51, 52, 53, 54, 55, 56, 57, 48},
			},
			PieceLength: 262144,
		}
		var peerId = [20]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
		const port uint16 = 6789

		// call the RequestPeers method
		ps, err := torrFile.RequestPeers(peerId, port)

		assert.NotNil(t, err)
		assert.Empty(t, ps)
	})

	t.Run("when an error occurs while sending the HTTP GET request", func(t *testing.T) {
		var torrFile = TorrentFile{
			Name:     "debian-10.2.0-amd64-netinst.iso",
			Length:   351272960,
			Announce: "http://127.0.0.1:6969/announce", // a server address that doesn't exist
			InfoHash: [20]byte{216, 247, 57, 206, 195, 40, 149, 108, 204, 91, 191, 31, 134, 217, 253, 207, 219, 168, 206, 182},
			PiecesHashes: [][20]byte{
				{49, 50, 51, 52, 53, 54, 55, 56, 57, 48, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106},
				{97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 49, 50, 51, 52, 53, 54, 55, 56, 57, 48},
			},
			PieceLength: 262144,
		}
		var peerId = [20]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
		const port uint16 = 6789

		// call the RequestPeers method
		ps, err := torrFile.RequestPeers(peerId, port)

		assert.NotNil(t, err)
		assert.Empty(t, ps)
	})
}
