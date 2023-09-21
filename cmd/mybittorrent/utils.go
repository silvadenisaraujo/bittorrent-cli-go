package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
)

// type TorrentFileInfo struct {
// 	PiecesLength int
// 	Pieces       string
// 	Name         string
// 	Length       int
// }

type TorrentFile struct {
	Announce string
	Info     map[string]interface{}
}

type Peers struct {
	Ip   string
	Port int
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

	// Map to struct
	// torrentInfo := TorrentFileInfo{
	// 	PiecesLength: decoded.(map[string]interface{})["info"].(map[string]interface{})["piece length"].(int),
	// 	Pieces:       decoded.(map[string]interface{})["info"].(map[string]interface{})["pieces"].(string),
	// 	Name:         decoded.(map[string]interface{})["info"].(map[string]interface{})["name"].(string),
	// 	Length:       decoded.(map[string]interface{})["info"].(map[string]interface{})["length"].(int),
	// }

	// print(&torrentInfo)

	torrent := TorrentFile{
		Announce: decoded.(map[string]interface{})["announce"].(string),
		Info:     decoded.(map[string]interface{})["info"].(map[string]interface{}),
	}

	return &torrent, nil
}

// Generate Peer ID based on MAC address
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

func getPeers(torrent *TorrentFile, peerId string) ([]Peers, error) {

	// Do HTTP GET request to the tracker
	req, err := http.NewRequest("GET", torrent.Announce, nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Hash the info
	encodedInfo, err := encodeBencode(torrent.Info)

	// Hash the info
	sha := sha1.New()
	sha.Write([]byte(encodedInfo))
	infoHash := sha.Sum(nil)

	// Add the query parameters
	q := req.URL.Query()
	q.Add("info_hash", string(infoHash))
	q.Add("peer_id", peerId)
	q.Add("port", "6881")
	q.Add("uploaded", "0")
	q.Add("downloaded", "0")
	q.Add("left", fmt.Sprint(torrent.Info["length"].(int)))
	q.Add("compact", "1")
	req.URL.RawQuery = q.Encode()

	// Do the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Decode the response
	decoded, _, err := decodeBencode(string(responseBody))

	// Get the peers from the string
	responsePeers := decoded.(map[string]interface{})["peers"].(string)
	peers := make([]Peers, len(responsePeers)/6)
	for i := 0; i < len(responsePeers); i += 6 {
		peers[i/6].Ip = fmt.Sprintf("%d.%d.%d.%d", responsePeers[i], responsePeers[i+1], responsePeers[i+2], responsePeers[i+3])
		peers[i/6].Port = int(responsePeers[i+4])<<8 + int(responsePeers[i+5])
	}

	return peers, nil
}
