package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, _, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else if command == "info" {

		torrentFile := os.Args[2]
		torrent, err := parse_file(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Encodes and hash the info
		infoHash, err := hashInfo(torrent)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Print the tracker URL and the file length
		fmt.Println("Tracker URL:", torrent.Announce)
		fmt.Println("Length:", torrent.Info["length"].(int))
		fmt.Printf("Info Hash: %x\n", infoHash)

		// Print the pieces
		fmt.Printf("Piece Length: %d\n", torrent.Info["piece length"].(int))
		fmt.Println("Piece Hashes:")
		pieces := torrent.Info["pieces"].(string)
		for i := 0; i < len(pieces); i += 20 {
			fmt.Printf("%x\n", pieces[i:i+20])
		}
	} else if command == "peers" {
		torrentFile := os.Args[2]
		torrent, err := parse_file(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Get the peer ID
		peerId, err := generatePeerId()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Do HTTP GET request to the tracker
		peers, err := getPeers(torrent, peerId)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Print the peers
		for _, peer := range peers {
			fmt.Printf("%s:%d\n", peer.Ip, peer.Port)
		}
	} else if command == "handshake" {
		torrentFile := os.Args[2]
		torrent, err := parse_file(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Get the peer ID
		peerStr := os.Args[3]
		peer, err := parsePeer(peerStr)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Get the local peer ID
		localPeerId, err := generatePeerId()

		// Encodes and hash the info
		infoHash, err := hashInfo(torrent)

		// Do the handshake
		handshakePeer, err := handshakePeer(peer, localPeerId, infoHash)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Print the handshake
		fmt.Println("Peer ID:", handshakePeer)

	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
