package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Peer represents a peer in the bittorrent network
type Peer struct {
	Ip   string
	Port int
}

// PeerConnection represents a peer that is connected to the local client
type PeerConnection struct {
	PeerId string
	Peer   *Peer
	Conn   *net.TCPConn
}

const (
	BlockSize int64 = 16 * 1024 // 16kb
)

type MessageType int // Bittorrent available message types
const (
	Choke MessageType = iota
	Unchoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
)

// Given a torrent file, we collect the Announce URL together with the InfoHash
// and we enable the client to request peers from the tracker server.
func RequestPeers(torrent *TorrentFile) ([]Peer, error) {

	// Get the local peer ID
	localPeerId, err := getLocalId()

	// Do HTTP GET request to the tracker
	req, err := http.NewRequest("GET", torrent.Announce, nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Add the query parameters
	q := req.URL.Query()
	q.Add("info_hash", string(torrent.InfoHash))
	q.Add("peer_id", localPeerId)
	q.Add("port", "6881")
	q.Add("uploaded", "0")
	q.Add("downloaded", "0")
	q.Add("left", fmt.Sprint(torrent.Info.Length))
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

	// Get the peers from the decoded string
	responsePeers := decoded.(map[string]interface{})["peers"].(string)
	peers := make([]Peer, len(responsePeers)/6)
	for i := 0; i < len(responsePeers); i += 6 {
		peers[i/6].Ip = fmt.Sprintf("%d.%d.%d.%d", responsePeers[i], responsePeers[i+1], responsePeers[i+2], responsePeers[i+3])
		peers[i/6].Port = int(responsePeers[i+4])<<8 + int(responsePeers[i+5])
	}

	return peers, nil
}

// Given a peer decoded string, we collect the peer IP and port.
func ParsePeerFromStr(peerStr string) (*Peer, error) {
	parts := strings.Split(peerStr, ":")
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	peer := Peer{Ip: parts[0], Port: port}
	return &peer, nil
}

// Executes a handshake with a peer and returns the peer ID and the TCP connection.
func (peer *Peer) Handshake(infoHash []byte) (*PeerConnection, error) {

	// Get the local peer ID
	localPeerId, err := getLocalId()

	// Map the TCP address
	tcpAddr, err := net.ResolveTCPAddr("tcp", peer.Ip+":"+strconv.Itoa(peer.Port))
	if err != nil {
		return nil, err
	}

	// Connect to the peer
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	// Send the handshake message according to BitTorrent protocol
	msg := []byte{}
	msg = append(msg, 19)
	msg = append(msg, []byte("BitTorrent protocol")...)
	msg = append(msg, make([]byte, 8)...)
	msg = append(msg, infoHash...)
	msg = append(msg, []byte(localPeerId)...)
	_, err = conn.Write(msg)
	if err != nil {
		return nil, err
	}

	// Read the handshake response according to BitTorrent protocol
	// (1 byte + 19 bytes + 8 bytes + 20 bytes + 20 bytes)
	// 1 byte: length of the protocol string
	// 19 bytes: protocol string
	// 8 bytes: reserved bytes
	// 20 bytes: info hash
	// 20 bytes: peer ID
	reply := make([]byte, 1+19+8+20+20)
	_, err = conn.Read(reply)
	if err != nil {
		return nil, err
	}

	// Get the peer ID
	replyPeerId := reply[1+19+8+20:]
	encodedPeerId := hex.EncodeToString(replyPeerId)

	// Create a peer connection
	peerConnection := PeerConnection{
		PeerId: encodedPeerId,
		Peer:   peer,
		Conn:   conn,
	}

	// Return the encoded peer ID and the TCP connection
	return &peerConnection, nil
}

// Sends a TCP message according to the protocol
// Return the number of bytes sent and an error if any
func (peerConnection *PeerConnection) sendMessage(messageType MessageType, payload []byte) (int, error) {
	// The message lenght follows the protocol
	// Payload Lenght: 4 bytes
	// Message Type: 1 byte
	// Payload: variable
	messageLength := 4 + 1 + len(payload)

	// Creates the message
	message := make([]byte, messageLength)

	// Populates the message
	binary.BigEndian.PutUint32(message[0:4], uint32(len(payload)+1))
	message[4] = uint8(messageType)
	copy(message[5:], payload)

	// Sends the message
	n, err := peerConnection.Conn.Write(message)
	if err != nil {
		return 0, err
	}

	return n, nil
}

// Reads a TCP message according to the protocol
func (peerConnection *PeerConnection) readMessage() (MessageType, []byte) {

	// First reads the message length
	var messageLength uint32
	binary.Read(peerConnection.Conn, binary.BigEndian, &messageLength)

	// Then reads the message type
	var messageTypeByte byte
	binary.Read(peerConnection.Conn, binary.BigEndian, &messageTypeByte)
	messageType := MessageType(messageTypeByte)

	// If there is a payload, reads it
	if messageLength > 1 {
		payload := make([]byte, messageLength-1)
		io.ReadAtLeast(peerConnection.Conn, payload, len(payload))
		return messageType, payload
	}

	// If there is no payload, returns nil
	return messageType, nil
}

// Request a piece from a peer given an index and the length of the piece.
// Returns the piece data and an error if any.
func (peerConnection *PeerConnection) RequestPiece(pieceLength int64, pieceIndex int, torrentLength int64) ([]byte, error) {

	// Check if this the last piece of a torrent
	if pieceIndex >= int(torrentLength/pieceLength) {
		pieceLength = torrentLength - (pieceLength * int64(pieceIndex))
	}

	// Create a byte array to store the piece
	data := make([]byte, pieceLength)

	// Check the size of the last Block Size
	lastBlockSize := pieceLength % BlockSize

	// Calculate the number of pieces
	piecesNum := (pieceLength - lastBlockSize) / BlockSize

	// If the last block size is greater than 0, we need to add one more piece
	if lastBlockSize > 0 {
		piecesNum++
	}

	// For each block, request the piece
	for i := int64(0); i < pieceLength; i += int64(BlockSize) {

		length := BlockSize

		if i+int64(BlockSize) > pieceLength {
			fmt.Printf("reached last block, changing size to %d\n", lastBlockSize)
			length = pieceLength - i
			if length > BlockSize {
				length = BlockSize
			}
		}

		// Create a piece request message
		// Piece Index: 4 bytes
		// Block Offset: 4 bytes
		// Block Length: 4 bytes
		requestMessage := make([]byte, 12)
		binary.BigEndian.PutUint32(requestMessage[0:4], uint32(pieceIndex))
		binary.BigEndian.PutUint32(requestMessage[4:8], uint32(i))
		binary.BigEndian.PutUint32(requestMessage[8:], uint32(length))

		// Send the request message
		_, err := peerConnection.sendMessage(Request, requestMessage)
		if err != nil {
			return nil, err
		}

		// Read the response message
		messageType, responseMsg := peerConnection.readMessage()

		// Check if the response is a piece message
		if messageType != Piece {
			fmt.Printf("Piece message not received! Received %v\n", messageType)
			os.Exit(1)
		}

		// If there is no response message, return the data read so far
		if responseMsg == nil {
			return data, nil
		}

		// Copy payload to data given its offset
		index := binary.BigEndian.Uint32(responseMsg[0:4])
		if uint32(pieceIndex) != index {
			return nil, fmt.Errorf("Expected piece index: %d, got=%d\n", pieceIndex, index)
		}
		begin := binary.BigEndian.Uint32(responseMsg[4:8])
		block := responseMsg[8:]
		copy(data[begin:], block)

	}

	return data, nil
}

// Generates a Peer ID based on MAC address max 20 characters
func getLocalId() (string, error) {

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
	peerId := "LOCAL-ID-" + macAddress
	return peerId[:20], nil
}
