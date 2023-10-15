package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {

	// Read command
	command := os.Args[1]

	if command == "decode" {
		// Example: ./your_bittorrent.sh decode 4:spam
		bencodedValue := os.Args[2]

		PrintDecodeValue(bencodedValue)
	} else if command == "info" {
		// Example: ./your_bittorrent.sh info sample.torrent
		torrentFile := os.Args[2]

		torrent := ParseFile(torrentFile)

		PrintFileInfo(torrent)
	} else if command == "peers" {
		// Example: ./your_bittorrent.sh peers sample.torrent
		torrentFile := os.Args[2]

		torrent := ParseFile(torrentFile)

		PrintPeers(torrent)
	} else if command == "handshake" {
		// Example: ./your_bittorrent.sh handshake sample.torrent PEER_ID
		torrentFile := os.Args[2]
		peerStr := os.Args[3]

		torrent := ParseFile(torrentFile)

		// Get the peer ID
		peer, err := ParsePeerFromStr(peerStr)
		if err != nil {
			fmt.Println(err)
			return
		}

		DoPeerHandshake(torrent, peer)
	} else if command == "download_piece" {
		// Example: ./your_bittorrent.sh download_piece -o /tmp/test-piece-0 sample.torrent 0
		destFile := os.Args[3]
		torrentFile := os.Args[4]
		pieceIndex, err := strconv.Atoi(os.Args[5])
		if err != nil {
			fmt.Println(err)
			return
		}

		// 	Read the torrent file to get the tracker URL
		torrent := ParseFile(torrentFile)

		// Download piece from the torrent
		DownloadPiece(destFile, torrent, pieceIndex)

	} else if command == "download" {
		// Example: ./your_bittorrent.sh download_piece -o /tmp/test-piece-0 sample.torrent 0
		destFile := os.Args[3]
		torrentFile := os.Args[4]

		// 	Read the torrent file to get the tracker URL
		torrent := ParseFile(torrentFile)

		// Download the file
		Download(destFile, torrent)
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
