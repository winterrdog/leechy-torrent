package torrentfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	/*
		test cases:
		1. when a valid torrent file is provided
		2. when a non-existent torrent file is provided
		3. when a file does not contain valid bencode data
		4. when a file contains valid bencode data but is not a valid torrent file
	*/

	t.Run("when a valid torrent file is provided", func(t *testing.T) {
		var expected = &TorrentFile{
			Announce:    "http://bttracker.debian.org:6969/announce",
			Length:      661651456,
			PieceLength: 262144,
			Name:        "debian-12.6.0-amd64-netinst.iso",
		}
		var torrFile, err = Open("./test-torrent-files/debian-12.6.0-amd64-netinst.iso.torrent")

		assert.Nil(t, err)
		assert.Equal(t, expected.Announce, torrFile.Announce)
		assert.Equal(t, expected.Length, torrFile.Length)
		assert.Equal(t, expected.PieceLength, torrFile.PieceLength)
		assert.Equal(t, expected.Name, torrFile.Name)
	})

	t.Run("when a non-existent torrent file is provided", func(t *testing.T) {
		var torrFile, err = Open("./test-torrent-files/non-existent-file.torrent")

		assert.NotNil(t, err)
		assert.Empty(t, torrFile)
	})

	t.Run("when a file does not contain valid bencode data", func(t *testing.T) {
		var torrFile, err = Open("./test-torrent-files/invalid-bencode-data.torrent")

		assert.NotNil(t, err)
		assert.Empty(t, torrFile)
	})

	t.Run("when a file contains valid bencode data but is not a valid torrent file", func(t *testing.T) {
		var torrFile, err = Open("./test-torrent-files/invalid-torrent-file.torrent")

		assert.NotNil(t, err)
		assert.Empty(t, torrFile)
	})
}

func TestToTorrentFile(t *testing.T) {
	/*
		test cases:
		1. when few bytes are passed in pieces field
	*/

	t.Run("when few bytes are passed in pieces field", func(t *testing.T) {
		var info = BencodeInfo{
			Name:        "debian-12.6.0-amd64-netinst.iso",
			Length:      661651456,
			PieceLength: 262144,
			Pieces:      "1234567890abcdef",
		}
		var bencodeTorrent = &BencodeTorrent{
			Announce: "http://bttracker.debian.org:6969/announce",
			Info:     info,
		}

		var torrentFile, err = bencodeTorrent.ToTorrentFile()
		assert.NotNil(t, err)
		assert.Empty(t, torrentFile)
	})
}
