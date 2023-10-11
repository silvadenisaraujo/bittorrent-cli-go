package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
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

// Decodes a bencoded value
func printDecodeValue(bencodedValue string) {
	decoded, _, err := decodeBencode(bencodedValue)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	jsonOutput, _ := json.Marshal(decoded)
	fmt.Println(string(jsonOutput))
}

// Prints the information about the torrent file
func printFileInfo(torrent *TorrentFile) {
	// Print the tracker URL and the file length
	fmt.Println("Tracker URL:", torrent.Announce)
	fmt.Println("Length:", torrent.Info.Length)
	fmt.Printf("Info Hash: %x\n", torrent.InfoHash)

	// Print the pieces
	fmt.Printf("Piece Length: %d\n", torrent.Info.PieceLen)
	fmt.Printf("Piece Hashes:\n")
	for _, piece := range torrent.Info.Pieces {
		fmt.Printf("%x\n", piece)
	}
}

// Prints the peers for the torrent file
func printPeers(torrent *TorrentFile) {

	// Get the peer ID
	peerId, err := getLocalId()
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
}

// Does the handshake with a peer and print the peer ID
func doPeerHandshake(torrent *TorrentFile, peer *Peer) {
	// Get the peer ID
	localId, err := getLocalId()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Do the handshake
	handshakedPeer, _, err := handshakePeer(peer, localId, torrent.InfoHash)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the handshake
	fmt.Println("Peer ID:", handshakedPeer)
}

// Downloads a piece from a peer and print the piece hash
func downloadPiece(destFile string, torrent *TorrentFile, pieceIndex int) {
	localPeerId, err := getLocalId()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Local Peer ID: %s\n", localPeerId)

	peers, err := getPeers(torrent, localPeerId)
	fmt.Printf("Peers: %v\n", peers)

	// Encodes and hash the info
	fmt.Printf("Info Hash: %x\n", torrent.InfoHash)

	// Get random peer from the peers list
	peer := peers[0]

	// Do the handshake
	handshakePeer, conn, err := handshakePeer(&peer, localPeerId, []byte(torrent.InfoHash))
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
}

// Downloads the file from a peer and print the file hash
func download(destFile string, torrent *TorrentFile) {
	// Perform the tracker GET request to get a list of peers
	localPeerId, err := getLocalId()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Local Peer ID: %s\n", localPeerId)

	peers, err := getPeers(torrent, localPeerId)
	fmt.Printf("Peers: %v\n", peers)

	// Encodes and hash the info
	fmt.Printf("Info Hash: %x\n", torrent.InfoHash)

	// Get random peer from the peers list
	peer := peers[0]

	// Do the handshake
	handshakePeer, conn, err := handshakePeer(&peer, localPeerId, torrent.InfoHash)
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
	piecesNum := int(torrent.Info.Length / torrent.Info.PieceLen)
	fmt.Printf("Num of Pieces: %d\n", piecesNum)
	data := []byte{}

	for i := 0; i <= piecesNum; i++ {

		fmt.Printf("Downloading piece %d\n", i)

		// Request piece
		pieceData, err := requestPiece(torrent, conn, i)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Downloaded piece %d length: %d\n", i, len(pieceData))

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

	fmt.Printf("Downloaded %s to %s\n", torrent.Path, destFile)
}

// Request a piece from a peer
func requestPiece(torrent *TorrentFile, conn *net.TCPConn, pieceIndex int) ([]byte, error) {
	pieceLength := int64(torrent.Info.PieceLen)
	length := int64(torrent.Info.Length)

	if pieceIndex >= int(length/pieceLength) {
		pieceLength = length - (pieceLength * int64(pieceIndex))
	}

	data := make([]byte, pieceLength)
	lastBlockSize := pieceLength % BlockSize
	piecesNum := (pieceLength - lastBlockSize) / BlockSize
	if lastBlockSize > 0 {
		piecesNum++
	}

	for i := int64(0); i < pieceLength; i += int64(BlockSize) {
		length := BlockSize
		if i+int64(BlockSize) > pieceLength {
			fmt.Printf("reached last block, changing size to %d\n", lastBlockSize)
			length = pieceLength - i
			if length > BlockSize {
				length = BlockSize
			}
		}
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

// Sends a TCP message according to the protocol
func sendMessage(conn *net.TCPConn, messageType MessageType, payload []byte) (int, error) {
	messageLength := 4 + 1 + len(payload)
	message := make([]byte, messageLength)
	binary.BigEndian.PutUint32(message[0:4], uint32(len(payload)+1))
	message[4] = uint8(messageType)
	copy(message[5:], payload)
	return conn.Write(message)
}

// Reads a TCP message according to the protocol
func readMessage(conn *net.TCPConn) (MessageType, []byte) {

	var messageLength uint32
	binary.Read(conn, binary.BigEndian, &messageLength)

	var messageTypeByte byte
	binary.Read(conn, binary.BigEndian, &messageTypeByte)
	messageType := MessageType(messageTypeByte)

	if messageLength > 1 {
		payload := make([]byte, messageLength-1)
		io.ReadAtLeast(conn, payload, len(payload))
		return messageType, payload
	}
	return messageType, nil
}
