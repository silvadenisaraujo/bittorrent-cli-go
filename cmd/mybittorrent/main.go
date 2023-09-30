package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
)

type MessageType int

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

func main() {

	// Read command
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
		// Arguments
		torrentFile := os.Args[2]
		peerStr := os.Args[3]

		torrent, err := parse_file(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Get the peer ID
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
		handshakePeer, _, err := handshakePeer(peer, localPeerId, infoHash)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Print the handshake
		fmt.Println("Peer ID:", handshakePeer)

	} else if command == "download_piece" {

		// Arguments
		// Example: ./your_bittorrent.sh download_piece -o /tmp/test-piece-0 sample.torrent 0
		destFile := os.Args[3]
		torrentFile := os.Args[4]
		pieceIndex, err := strconv.Atoi(os.Args[5])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Arguments - destFile: %s, torrentFile: %s, pieceIndex: %d\n", destFile, torrentFile, pieceIndex)

		// 	Read the torrent file to get the tracker URL
		torrent, err := parse_file(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Perform the tracker GET request to get a list of peers
		localPeerId, err := generatePeerId()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Local Peer ID: %s\n", localPeerId)

		peers, err := getPeers(torrent, localPeerId)
		fmt.Printf("Peers: %v\n", peers)

		// Encodes and hash the info
		infoHash, err := hashInfo(torrent)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Info Hash: %x\n", infoHash)

		// Get random peer from the peers list
		peer := peers[0]

		// Do the handshake
		handshakePeer, conn, err := handshakePeer(&peer, localPeerId, infoHash)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Handshake Peer: %s\n", handshakePeer)

		defer conn.Close()

		// Exchange multiple peer messages to download the file
		// Wait for bitfield 5 message
		messageType, _ := readMessage(conn)

		if messageType != Bitfield {
			fmt.Println("Bitfield message not received!")
			os.Exit(1)
		}

		// Send interested message
		interestedMessage := []byte{0, 0, 0, 1, 2}
		_, err = conn.Write(interestedMessage)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Wait for unchoke message
		messageType, _ = readMessage(conn)
		if messageType != Unchoke {
			fmt.Println("Unchoke message not received!")
			os.Exit(1)
		}

		// Request piece
		pieceData, err := requestPiece(torrent, conn, pieceIndex)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Write piece to file
		file, err := os.Create(destFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		file.Write(pieceData)

		fmt.Printf("Piece %d downloaded to %s\n", pieceIndex, destFile)

	} else if command == "download" {
		// Arguments
		// Example: ./your_bittorrent.sh download_piece -o /tmp/test-piece-0 sample.torrent 0
		destFile := os.Args[3]
		torrentFile := os.Args[4]
		fmt.Printf("Arguments - destFile: %s, torrentFile: %s\n", destFile, torrentFile)

		// 	Read the torrent file to get the tracker URL
		torrent, err := parse_file(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Perform the tracker GET request to get a list of peers
		localPeerId, err := generatePeerId()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Local Peer ID: %s\n", localPeerId)

		peers, err := getPeers(torrent, localPeerId)
		fmt.Printf("Peers: %v\n", peers)

		// Encodes and hash the info
		infoHash, err := hashInfo(torrent)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Info Hash: %x\n", infoHash)

		// Get random peer from the peers list
		peer := peers[0]

		// Do the handshake
		handshakePeer, conn, err := handshakePeer(&peer, localPeerId, infoHash)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Handshake Peer: %s\n", handshakePeer)

		defer conn.Close()

		// Exchange multiple peer messages to download the file
		// Wait for bitfield 5 message
		messageType, _ := readMessage(conn)

		if messageType != Bitfield {
			fmt.Println("Bitfield message not received!")
			os.Exit(1)
		}

		// Send interested message
		interestedMessage := []byte{0, 0, 0, 1, 2}
		_, err = conn.Write(interestedMessage)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Wait for unchoke message
		messageType, _ = readMessage(conn)
		if messageType != Unchoke {
			fmt.Println("Unchoke message not received!")
			os.Exit(1)
		}

		// Dowload all pieces
		piecesNum := int(torrent.Info["length"].(int) / torrent.Info["piece length"].(int))
		fmt.Printf("Num of Pieces: %d\n", piecesNum)
		data := []byte{}

		for i := 0; i < piecesNum; i++ {
			// Request piece
			pieceData, err := requestPiece(torrent, conn, i)
			if err != nil {
				fmt.Println(err)
				return
			}
			// Write piece to file
			data = append(data, pieceData...)
		}

		file, err := os.Create(destFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		file.Write(data)

		fmt.Printf("Downloaded %s to %s\n", torrentFile, destFile)

	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

func requestPiece(torrent *TorrentFile, conn *net.TCPConn, pieceIndex int) ([]byte, error) {
	pieceLength := int64(torrent.Info["piece length"].(int))
	length := int64(torrent.Info["length"].(int))

	if pieceIndex >= int(length/pieceLength) {
		pieceLength = length - (pieceLength * int64(pieceIndex))
	}

	fmt.Printf("[requestPiece] - Piece Length: %d - Length: %d - Piece Index: %d\n", pieceLength, length, pieceIndex)

	data := make([]byte, pieceLength)
	lastBlockSize := pieceLength % BlockSize
	piecesNum := (pieceLength - lastBlockSize) / BlockSize
	if lastBlockSize > 0 {
		piecesNum++
	}

	fmt.Printf("[requestPiece] - Piece Length: %d # of Pieces: %d\n", pieceLength, piecesNum)

	for i := int64(0); i < pieceLength; i += int64(BlockSize) {
		length := BlockSize
		if i+int64(BlockSize) > pieceLength {
			fmt.Printf("reached last block, changing size to %d\n", lastBlockSize)
			length = pieceLength - i
			if length > BlockSize {
				length = BlockSize
			}
		}
		fmt.Printf("[requestPiece] - Requesting block %d of %d (offset=%d, size=%d)\n", i, pieceLength, i, length)
		requestMessage := make([]byte, 12)
		binary.BigEndian.PutUint32(requestMessage[0:4], uint32(pieceIndex))
		binary.BigEndian.PutUint32(requestMessage[4:8], uint32(i))
		binary.BigEndian.PutUint32(requestMessage[8:], uint32(length))
		_, err := sendMessage(conn, Request, requestMessage)
		if err != nil {
			return nil, err
		}

		messageType, responseMsg := readMessage(conn)
		if messageType != Piece {
			fmt.Printf("Piece message not received! Received %v\n", messageType)
			os.Exit(1)
		}

		fmt.Printf("Recieved piece message\n")
		if responseMsg == nil {
			return data, nil
		}

		// Copy payload to data
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

func sendMessage(conn *net.TCPConn, messageType MessageType, payload []byte) (int, error) {
	messageLength := 4 + 1 + len(payload)
	message := make([]byte, messageLength)
	binary.BigEndian.PutUint32(message[0:4], uint32(len(payload)+1))
	message[4] = uint8(messageType)
	copy(message[5:], payload)
	fmt.Printf("[sendMessage] - Message: %v\n", message)
	return conn.Write(message)
}

func readMessage(conn *net.TCPConn) (MessageType, []byte) {

	var messageLength uint32
	binary.Read(conn, binary.BigEndian, &messageLength)
	fmt.Printf("[readMessage] - Message length: %d\n", messageLength)

	var messageTypeByte byte
	binary.Read(conn, binary.BigEndian, &messageTypeByte)
	messageType := MessageType(messageTypeByte)
	fmt.Printf("[readMessage] - Message type: %d\n", messageType)

	if messageLength > 1 {
		payload := make([]byte, messageLength-1)
		io.ReadAtLeast(conn, payload, len(payload))
		return messageType, payload
	}
	return messageType, nil
}
