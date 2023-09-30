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

const (
	BlockSize int64 = 16 * 1024 // 16kb
)

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

func handshakePeer(peer *Peer, localPeerId string, infoHash []byte) (string, *net.TCPConn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", peer.Ip+":"+strconv.Itoa(peer.Port))
	if err != nil {
		return "", nil, err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return "", nil, err
	}
	msg := []byte{}
	msg = append(msg, 19)
	msg = append(msg, []byte("BitTorrent protocol")...)
	msg = append(msg, make([]byte, 8)...)
	msg = append(msg, infoHash...)
	msg = append(msg, []byte(localPeerId)...)
	_, err = conn.Write(msg)
	if err != nil {
		return "", nil, err
	}
	reply := make([]byte, 1+19+8+20+20)
	_, err = conn.Read(reply)
	if err != nil {
		return "", nil, err
	}
	replyPeerId := reply[1+19+8+20:]
	return hex.EncodeToString(replyPeerId), conn, nil
}

// func sendPeerMessages(peer *Peer, localPeerId string, index int, torrent *TorrentFile, pieceIndex int, conn net.TCPConn) (string, error) {
// 	totalPieces := len(torrent.Info["pieces"].(string))
// 	if pieceIndex > totalPieces {
// 		return "nil", fmt.Errorf("this torrent only has %d pieces but %d-th requested", totalPieces-1, pieceIndex)
// 	}

// 	pieceLength := torrent.Info["piece length"].(int)
// 	if pieceIndex == totalPieces-1 {
// 		pieceLength = totalPieces - (pieceIndex * totalPieces)
// 	}
// 	lastBlockSize := pieceLength % int(BlockSize)
// 	numBlocks := (pieceLength - lastBlockSize) / int(BlockSize)
// 	log.Printf("there are %d blocks in piece %d\n", numBlocks, pieceIndex)

// 	if lastBlockSize > 0 {
// 		log.Printf("piece %d has an unaligned block of size %d\n", pieceIndex, lastBlockSize)
// 		numBlocks++
// 	} else {
// 		log.Printf("piece %d has size of %d and is aligned with blocksize of %d\n", pieceIndex, t.PieceLength, BlockSize)
// 	}

// 	piece := make([]byte, pieceLength)
// 	for i := 0; i < int(numBlocks); i++ {
// 		begin := i * BlockSize
// 		length := BlockSize
// 		if lastBlockSize > 0 && i == int(numBlocks)-1 {
// 			log.Printf("reached last block, changing size to %d\n", lastBlockSize)
// 			length = int(lastBlockSize)
// 		}
// 		log.Printf("requesting block %d of %d (offset=%d, size=%d)\n", i, numBlocks-1, begin, length)

// 		// Request block for piece
// 		waitForMessage()

// 		sendMessage()

// 		waitForMessage()

// 		downloadPiece()

// 	}
// }

// func waitForMessage(conn net.TCPConn) ([]byte, error) {
// 	var messageLength uint32
// 	var messageID byte
// 	if err := binary.Read(conn, binary.BigEndian, &messageLength); err != nil {
// 		return nil, fmt.Errorf("error while reading message length: %s", err.Error())
// 	}
// 	if err := binary.Read(conn, binary.BigEndian, &messageID); err != nil {
// 		return nil, fmt.Errorf("error while reading message ID: %s", err.Error())
// 	}
// 	if m != MessageType(messageID) {
// 		return nil, fmt.Errorf("unexpected message ID: (actual=%d, expected=%s)", messageID, m)
// 	}
// 	log.Printf("received message %s\n", m)
// 	if messageLength > 1 {
// 		log.Printf("message %s has attached payload of size %d\n", m, messageLength-1)
// 		payload := make([]byte, messageLength-1)
// 		if _, err := io.ReadAtLeast(t.peerConnection, payload, len(payload)); err != nil {
// 			return nil, fmt.Errorf("error while reading payload: %s", err.Error())
// 		}
// 		return payload, nil
// 	}
// 	return nil, nil
// }
