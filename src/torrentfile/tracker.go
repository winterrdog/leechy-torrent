package torrentfile

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
	"github.com/winterrdog/lean-bit-torrent-client/common"
	"github.com/winterrdog/lean-bit-torrent-client/peers"
)

// BencodeTrackerResp represents the response from a Bittorrent tracker
type BencodeTrackerResp struct {
	Interval int    `bencode:"interval"` // interval in seconds to wait between requests
	Peers    string `bencode:"peers"`    // peers in compact format
}

// BuildTrackerUrl builds the tracker URL for the torrent file.
// It takes the peer ID and port as parameters and returns the built URL as a string.
// The URL includes query parameters such as info_hash, peer_id, port, uploaded, downloaded, compact, and left.
// If there is an error while parsing the announce URL, it returns an empty string and the error.
func (torrFile *TorrentFile) BuildTrackerUrl(peerId common.Sha1Hash, port uint16) (string, error) {
	var base, err = url.Parse(torrFile.Announce)
	if err != nil {
		return "", err
	}

	// URL query parameters to send to the tracker attached to the base URL
	var params = url.Values{
		"info_hash":  []string{string(torrFile.InfoHash[:])},
		"peer_id":    []string{string(peerId[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(int(torrFile.Length))},
	}

	base.RawQuery = params.Encode()

	return base.String(), nil
}

// RequestPeers sends a request to the tracker to get a list of peers for the given torrent file.
// It takes the peerId and port as parameters and returns a slice of peers and an error, if any.
// The function builds the tracker URL using the peerId and port, sends an HTTP GET request to the tracker,
// and unmarshals the response to extract the list of peers.
// It returns the unmarshaled list of peers or an error if any error occurs during the process.
func (tf *TorrentFile) RequestPeers(peerId common.Sha1Hash, port uint16) ([]peers.Peer, error) {
	var url, err = tf.BuildTrackerUrl(peerId, port)
	if err != nil {
		return nil, err
	}

	var response *http.Response
	var httpClient = &http.Client{Timeout: 15 * time.Second}
	response, err = httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var trackerResp = BencodeTrackerResp{}
	err = bencode.Unmarshal(response.Body, &trackerResp)
	if err != nil {
		return nil, err
	}

	return peers.Unmarshal([]byte(trackerResp.Peers))
}
