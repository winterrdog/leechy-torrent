package torrentfile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/jackpal/bencode-go"
	"github.com/winterrdog/lean-bit-torrent-client/common"
	"github.com/winterrdog/lean-bit-torrent-client/p2p"
	"github.com/winterrdog/lean-bit-torrent-client/peers"
)

/*
	serialization structs
*/

// BencodeInfo represents the information about a torrent file.
type BencodeInfo struct {
	Name        string `bencode:"name"`         // Name of the file or directory.
	Length      uint32 `bencode:"length"`       // Length of the file in bytes.
	PieceLength uint32 `bencode:"piece length"` // Length of each piece in bytes.
	Pieces      string `bencode:"pieces"`       // Concatenated SHA-1 hash values of all the pieces.
}

// BencodeTorrent represents a torrent file in the BitTorrent client.
type BencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     BencodeInfo `bencode:"info"`
}

/*
	Application structs
*/

// TorrentFile represents a torrent file.
type TorrentFile struct {
	Announce     string            // Announce is the URL of the tracker.
	InfoHash     common.Sha1Hash   // InfoHash is the SHA-1 hash of the info dictionary.
	PiecesHashes []common.Sha1Hash // PiecesHashes is a list of SHA-1 hashes of the pieces.
	PieceLength  uint32            // PieceLength is the length of each piece in bytes.
	Length       uint32            // Length is the total length of the file in bytes.
	Name         string            // Name is the name of the file.
}

// Open opens a torrent file at the specified path and returns a TorrentFile object.
// It reads the contents of the file, unmarshals them into a struct, and converts
// the struct into a TorrentFile object for use by the application.
// If an error occurs during the process, it returns an empty TorrentFile object and the error.
func Open(path string) (*TorrentFile, error) {
	// open file
	var file, err = os.Open(path)
	if err != nil {
		return &TorrentFile{}, err
	}
	defer file.Close()

	// unmarshal the contents into a struct for use by application
	var bto = BencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return &TorrentFile{}, err
	}

	var torrentFile TorrentFile
	torrentFile, err = bto.ToTorrentFile()
	if err != nil {
		return &TorrentFile{}, err
	}

	return &torrentFile, err
}

// Hash calculates the SHA-1 Hash of the BencodeInfo struct.
// It returns the calculated Hash and any error encountered during the process.
func (info *BencodeInfo) Hash() (common.Sha1Hash, error) {
	var buf bytes.Buffer

	var err = bencode.Marshal(&buf, *info)
	if err != nil {
		return common.Sha1Hash{}, err
	}

	var digest = sha1.Sum(buf.Bytes())

	return digest, err
}

// SplitPiecesHashes splits the pieces of the BencodeInfo struct into individual SHA1 hashes.
// It returns a slice of Sha1Hashes and an error if the pieces are malformed.
func (info *BencodeInfo) SplitPiecesHashes() ([]common.Sha1Hash, error) {
	var hashLen = 20
	var buf = []byte(info.Pieces)
	var bufLen = len(buf)

	if bufLen%hashLen != 0 {
		var err = fmt.Errorf("received malformed pieces of length %d", bufLen)
		return []common.Sha1Hash{}, err
	}

	var numHashes = bufLen / hashLen
	var hashes = make([]common.Sha1Hash, numHashes)

	var dest, src []byte
	var start, end int
	for i := 0; i != numHashes; i++ {
		dest = hashes[i][:]

		start, end = i*hashLen, (i+1)*hashLen
		src = buf[start:end]

		copy(dest, src)
	}

	return hashes, nil
}

// ToTorrentFile converts a BencodeTorrent into a TorrentFile.
// It calculates the info hash and splits the pieces hashes.
// Returns the converted TorrentFile and any error encountered.
func (bto *BencodeTorrent) ToTorrentFile() (TorrentFile, error) {
	var infoHash, err = bto.Info.Hash()
	if err != nil {
		return TorrentFile{}, err
	}

	var piecesHashes []common.Sha1Hash
	piecesHashes, err = bto.Info.SplitPiecesHashes()
	if err != nil {
		return TorrentFile{}, err
	}

	var torrentFile = TorrentFile{
		Name:         bto.Info.Name,
		Length:       bto.Info.Length,
		Announce:     bto.Announce,
		InfoHash:     infoHash,
		PiecesHashes: piecesHashes,
		PieceLength:  bto.Info.PieceLength,
	}

	return torrentFile, nil
}

// DownloadToFile downloads the torrent file and saves it to the specified path.
// It generates a peer ID, requests for peers, and then downloads the torrent file.
// The downloaded file is saved to the specified path.
//
// Parameters:
// - path: The path where the downloaded file will be saved.
//
// Returns:
// - error: An error if any occurred during the download process, otherwise nil.
func (tf *TorrentFile) DownloadToFile(path string) error {
	// generate peer ID
	var peerId common.Sha1Hash
	var _, err = rand.Read(peerId[:])
	if err != nil {
		return err
	}

	// request for peers
	var peers []peers.Peer
	peers, err = tf.RequestPeers(peerId, common.DefaultBittorrentPort)
	if err != nil {
		return err
	}

	// download torrent
	var torrent = p2p.Torrent{
		Peers:        peers,
		PeerId:       peerId,
		InfoHash:     tf.InfoHash,
		Name:         tf.Name,
		Length:       int(tf.Length),
		PieceLength:  int(tf.PieceLength),
		PiecesHashes: tf.PiecesHashes,
	}
	err = torrent.Download(path)
	if err != nil {
		return err
	}

	return nil
}
