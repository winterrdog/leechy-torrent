package p2p

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/winterrdog/lean-bit-torrent-client/client"
	"github.com/winterrdog/lean-bit-torrent-client/common"
	"github.com/winterrdog/lean-bit-torrent-client/message"
	"github.com/winterrdog/lean-bit-torrent-client/peers"
)

const (
	MaxBlockSize = 16384 // largest number of bytes a request can ask for
	MaxBacklog   = 8     // number of unfulfilled requests a client can have in its pipeline
)

// Torrent represents a BitTorrent file.
type Torrent struct {
	Name         string            // Name of the torrent file.
	Length       int               // Length of the torrent file in bytes.
	Peers        []peers.Peer      // List of peers connected to the torrent.
	PeerId       common.Sha1Hash   // Peer ID of the client.
	InfoHash     common.Sha1Hash   // Info hash of the torrent file.
	PieceLength  int               // Length of each piece in bytes.
	PiecesHashes []common.Sha1Hash // List of SHA-1 hashes for each piece.
}

// PieceWork represents a piece of work in the BitTorrent client.
type PieceWork struct {
	Index  int             // Index of the piece in the torrent file.
	Hash   common.Sha1Hash // SHA-1 hash of the piece.
	Length int             // Length of the piece in bytes.
}

// PieceResult represents the result of a piece download operation.
type PieceResult struct {
	Index int    // Index is the index of the downloaded piece.
	Buf   []byte // Buf is the byte buffer containing the downloaded piece data.
}

// PieceProgress represents the progress of a piece in the BitTorrent client.
type PieceProgress struct {
	Index      int            // Index of the piece
	Client     *client.Client // Client associated with the piece
	Buf        []byte         // Buffer for storing the piece data
	Downloaded int            // Number of bytes downloaded for the piece
	Requested  int            // Number of bytes requested for the piece
	Backlog    int            // Number of bytes in the backlog for the piece
}

// ReadMessage reads a message from the client and updates the state accordingly.
// It blocks until a message is received or an error occurs.
// If the message is a keep-alive message, it returns nil.
// If the message is a choke message, it sets the client's Choked flag to true.
// If the message is an unchoke message, it sets the client's Choked flag to false.
// If the message is a have message, it parses the index from the message and sets the corresponding piece in the client's Bitfield.
// If the message is a piece message, it parses the piece size from the message, updates the downloaded count and backlog count in the state.
// Returns an error if any error occurs during reading or parsing the message.
func (state *PieceProgress) ReadMessage() error {
	var msg, err = state.Client.Read() // call blocks
	if err != nil {
		return err
	}

	// send keep-alive
	if msg == nil {
		return nil
	}

	// handle message
	switch msg.Id {
	case message.MsgChoke:
		state.Client.Choked = true
	case message.MsgUnchoke:
		state.Client.Choked = false
	case message.MsgHave:
		var index int

		index, err = message.ParseHave(msg)
		if err != nil {
			return err
		}

		state.Client.Bitfield.SetPiece(index)
	case message.MsgPiece:
		var pieceSize int

		// parse piece size
		pieceSize, err = message.ParsePiece(state.Index, state.Buf, msg)
		if err != nil {
			return err
		}

		// update state with downloaded piece size
		state.Downloaded += pieceSize
		state.Backlog--
	}

	return nil
}

// checkIntegrity checks the integrity of a piece of data by comparing its hash with the expected hash.
// It returns an error if the integrity check fails.
func checkIntegrity(pw *PieceWork, buf []byte) error {
	var hash = sha1.Sum(buf)
	if !bytes.Equal(hash[:], pw.Hash[:]) {
		return fmt.Errorf("index %d failed integrity check. it could be corrupt somehow", pw.Index)
	}

	return nil
}

// calculateBoundsForPiece calculates the start and end bounds for a given piece index.
// It takes the index of the piece as input and returns the start and end bounds as output.
// The start bound is calculated as the index multiplied by the piece length.
// The end bound is calculated as the start bound plus the piece length.
// If the end bound exceeds the total length of the torrent, it is adjusted to the torrent length.
func (torrent *Torrent) calculateBoundsForPiece(index int) (start int, end int) {
	start = index * torrent.PieceLength
	end = start + torrent.PieceLength

	if end > torrent.Length {
		end = torrent.Length
	}

	return start, end
}

// calculatePieceSize calculates the size of a piece in bytes for the given index.
// It takes the index of the piece as a parameter and returns the size of the piece.
func (torrent *Torrent) calculatePieceSize(index int) int {
	var start, end = torrent.calculateBoundsForPiece(index)
	return end - start
}

// attemptDownloadPiece attempts to download a piece of the torrent file.
// It returns the downloaded piece as a byte slice and an error if any.
// The function sets a deadline to get unresponsive peers unstuck and disables the deadline afterwards.
// It sends requests to unchoked peers until enough requests are in the pipeline.
// The function reads messages from the peers and continues until the entire piece is downloaded.
func attemptDownloadPiece(torrentClient *client.Client, pw *PieceWork) ([]byte, error) {
	var state = PieceProgress{
		Index:  pw.Index,
		Client: torrentClient,
		Buf:    make([]byte, pw.Length),
	}

	// Setting a deadline helps get unresponsive peers unstuck.
	// 30 seconds is more than enough time to download a 262 KB piece
	torrentClient.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer torrentClient.Conn.SetDeadline(time.Time{}) // disable deadline

	// actually download the piece
	var err error
	var blockSize, remainingBytes int
	for state.Downloaded < pw.Length {
		// if unchoked, send requests until we've enough requests in our pipeline
		if !state.Client.Choked {
			for state.Backlog != MaxBacklog && state.Requested < pw.Length {
				blockSize = MaxBlockSize

				// last block might be shorter than the typical block
				remainingBytes = pw.Length - state.Requested
				if remainingBytes < blockSize {
					blockSize = remainingBytes
				}

				err = torrentClient.SendRequest(pw.Index, state.Requested, blockSize)
				if err != nil {
					return nil, err
				}

				state.Backlog++
				state.Requested += blockSize
			}
		}

		err = state.ReadMessage()
		if err != nil {
			return nil, err
		}
	}

	return state.Buf, nil
}

// startDownloadWorker starts a download worker for a given peer in the BitTorrent client.
// It performs the handshake with the peer, sends necessary messages, and downloads the requested pieces.
// The downloaded pieces are sent to the results channel.
// If an error occurs during the download process, the function logs the error and returns.
func (torrent *Torrent) startDownloadWorker(peer *peers.Peer, workQueue chan *PieceWork, results chan *PieceResult) {
	var torrentClient, err = client.New(peer, &torrent.PeerId, &torrent.InfoHash)
	if err != nil {
		log.Printf("failed to handshake with %s: %s\n", peer.IP, err)
		return
	}
	defer torrentClient.Conn.Close()
	log.Printf("completed handshake with %s\n", peer.IP)

	torrentClient.SendUnchoke()
	torrentClient.SendInterested()

	var buf []byte
	for pw := range workQueue {
		// check if peer has the piece we want
		if !torrentClient.Bitfield.HasPiece(pw.Index) {
			workQueue <- pw // put piece back on queue
			continue
		}

		// download the piece
		buf, err = attemptDownloadPiece(torrentClient, pw)
		if err != nil {
			log.Println("exiting...", err)
			workQueue <- pw
			return
		}

		err = checkIntegrity(pw, buf)
		if err != nil {
			log.Printf("piece #%d failed an integrity check\n", pw.Index)
			workQueue <- pw
			continue
		}

		// send "have" message to all peers and send piece to results channel
		torrentClient.SendHave(pw.Index)
		results <- &PieceResult{Index: pw.Index, Buf: buf}
	}
}

// Download downloads the torrent file and returns the downloaded data as a byte slice.
// It initializes workers to send work to consumers and starts downloading pieces from peers.
// The downloaded pieces are collected into a buffer until the download is complete.
// It logs the progress of the download, including the percentage completed and the number of peers involved.
// Returns the downloaded data as a byte slice and any error encountered during the download process.
func (torrent *Torrent) Download(path string) error {
	log.Println("starting download for", torrent.Name+"...")

	// init workers. generally setup the producers to send work to consumers
	var (
		length    int
		workQueue = make(chan *PieceWork, len(torrent.PiecesHashes))
		results   = make(chan *PieceResult)
	)
	for index, hash := range torrent.PiecesHashes {
		length = torrent.calculatePieceSize(index)
		workQueue <- &PieceWork{Index: index, Hash: hash, Length: length}
	}
	defer close(workQueue)
	defer close(results)

	// start workers which will download pieces from peers
	for _, peer := range torrent.Peers {
		go torrent.startDownloadWorker(&peer, workQueue, results)
	}

	// write results into a file until end
	var outputFile, err = os.Create(path)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	/*
		todo: implement a buffering
			algorithm to determine when to write to file:
			- create a buffer as big as 4 blocks of size MaxBlockSize
			- write to buffer until it's full.
			- write buffer to file
	*/

	var (
		downloadedPiece   *PieceResult
		start, numWorkers int
		percent           float64
		donePieces        = 0
		totalPieces       = len(torrent.PiecesHashes)
	)
	for donePieces != totalPieces {
		// collect results
		downloadedPiece = <-results
		start, _ = torrent.calculateBoundsForPiece(downloadedPiece.Index)

		// write piece into file
		_, err = outputFile.WriteAt(downloadedPiece.Buf, int64(start))
		if err != nil {
			return err
		}

		donePieces++

		// log progress
		percent = (float64(donePieces) / float64(totalPieces)) * 100
		numWorkers = runtime.NumGoroutine() - 1 // remove the main thread
		log.Printf("(%0.2f%%) downloaded piece number %d from %d peer(s)\n", percent, downloadedPiece.Index, numWorkers)
	}

	return nil
}
