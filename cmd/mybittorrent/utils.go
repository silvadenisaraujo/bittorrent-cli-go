package main

import (
	"fmt"
	"net"
	"os"
)

type TorrentFile struct {
	Announce string
	Info     map[string]interface{}
}

func parse_file(filepath string) (*TorrentFile, error) {
	torrentFile := os.Args[2]

	// Open the torrent file
	file, err := os.Open(torrentFile)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Read the torrent file
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Read the torrent file
	fileContent := make([]byte, fileInfo.Size())
	_, err = file.Read(fileContent)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Decode the torrent file
	decoded, _, err := decodeBencodeDictionary(string(fileContent))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	torrent := TorrentFile{
		Announce: decoded.(map[string]interface{})["announce"].(string),
		Info:     decoded.(map[string]interface{})["info"].(map[string]interface{}),
	}

	return &torrent, nil
}

// Generates a Peer ID based on MAC address max 20 characters
func generatePeerId() (string, error) {

	// Get MAC address
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// Get the first MAC address
	var macAddress string
	for _, i := range interfaces {
		if i.HardwareAddr.String() != "" {
			macAddress = i.HardwareAddr.String()
			break
		}
	}

	// Generate Peer ID with 20 bytes
	peerId := "MY0001-" + macAddress

	return peerId[:20], nil
}
