package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Peer struct {
	Ip   string
	Port int
}

func getPeers(torrent *TorrentFile, peerId string) ([]Peer, error) {

	// Do HTTP GET request to the tracker
	req, err := http.NewRequest("GET", torrent.Announce, nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Encodes and hash the info
	infoHash, err := hashInfo(torrent)

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
	peers := make([]Peer, len(responsePeers)/6)
	for i := 0; i < len(responsePeers); i += 6 {
		peers[i/6].Ip = fmt.Sprintf("%d.%d.%d.%d", responsePeers[i], responsePeers[i+1], responsePeers[i+2], responsePeers[i+3])
		peers[i/6].Port = int(responsePeers[i+4])<<8 + int(responsePeers[i+5])
	}

	return peers, nil
}

func parsePeer(peerStr string) (*Peer, error) {
	parts := strings.Split(peerStr, ":")
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	peer := Peer{Ip: parts[0], Port: port}
	return &peer, nil
}

func handshakePeer(peer *Peer, localPeerId string, infoHash []byte) (string, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", peer.Ip+":"+strconv.Itoa(peer.Port))
	if err != nil {
		return "", err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return "", err
	}
	// msg := make([]byte, 1+19+8+20+20)
	msg := []byte{}
	msg = append(msg, 19)
	msg = append(msg, []byte("BitTorrent protocol")...)
	msg = append(msg, make([]byte, 8)...)
	msg = append(msg, infoHash...)
	msg = append(msg, []byte(localPeerId)...)
	_, err = conn.Write(msg)
	if err != nil {
		return "", err
	}
	reply := make([]byte, 1+19+8+20+20)
	_, err = conn.Read(reply)
	if err != nil {
		return "", err
	}
	replyPeerId := reply[1+19+8+20:]
	return hex.EncodeToString(replyPeerId), nil
}