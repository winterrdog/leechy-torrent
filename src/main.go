package main

import (
	"log"
	"os"

	"github.com/winterrdog/lean-bit-torrent-client/torrentfile"
)

func main() {
	var (
		inPath, outPath string
		err             error
		torrentFile     *torrentfile.TorrentFile
	)

	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <input.torrent> <output.file>", os.Args[0])
	}

	inPath = os.Args[1]
	outPath = os.Args[2]

	// open torrent file to get details
	torrentFile, err = torrentfile.Open(inPath)
	if err != nil {
		goto handleErrorAndExit
	}

	// download the file via Bittorrent
	err = torrentFile.DownloadToFile(outPath)
	if err != nil {
		goto handleErrorAndExit
	}

	return

handleErrorAndExit:
	log.Fatal(err)
}
