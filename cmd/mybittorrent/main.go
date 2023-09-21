package main

import (
	"crypto/sha1"
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

		// Encodes the info
		encodedInfo, err := encodeBencode(torrent.Info)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Hash the info
		sha := sha1.New()
		sha.Write([]byte(encodedInfo))
		infoHash := sha.Sum(nil)

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

	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
