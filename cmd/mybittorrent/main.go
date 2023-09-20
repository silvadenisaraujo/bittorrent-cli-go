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

		// Open the torrent file
		file, err := os.Open(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Read the torrent file
		fileInfo, err := file.Stat()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Read the torrent file
		fileContent := make([]byte, fileInfo.Size())
		_, err = file.Read(fileContent)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Decode the torrent file
		decoded, _, err := decodeBencodeDictionary(string(fileContent))
		if err != nil {
			fmt.Println(err)
			return
		}

		// Get the tracker URL and the file length
		var trackerURL string = decoded.(map[string]interface{})["announce"].(string)
		var length int = decoded.(map[string]interface{})["info"].(map[string]interface{})["length"].(int)

		// Encodes the info
		info := decoded.(map[string]interface{})["info"].(map[string]interface{})
		encodedInfo, err := encodeBencode(info)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Hash the info
		sha := sha1.New()
		sha.Write([]byte(encodedInfo))
		infoHash := sha.Sum(nil)

		// Print the tracker URL and the file length
		fmt.Println("Tracker URL:", trackerURL)
		fmt.Println("Length:", length)
		fmt.Printf("Info Hash: %x\n", infoHash)

		// Print the pieces
		fmt.Printf("Piece Length: %d\n", info["piece length"].(int))
		fmt.Println("Piece Hashes:")
		pieces, ok := info["pieces"].(string)
		if !ok {
			fmt.Println("Could not decode pieces")
			return
		}
		for i := 0; i < len(pieces); i += 20 {
			fmt.Printf("%x\n", pieces[i:i+20])
		}

	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
