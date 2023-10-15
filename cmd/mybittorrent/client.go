package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Decodes a bencoded value
func PrintDecodeValue(bencodedValue string) {
	decoded, _, err := decodeBencode(bencodedValue)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	jsonOutput, _ := json.Marshal(decoded)
	fmt.Println(string(jsonOutput))
}

// Prints the information about the torrent file
func PrintFileInfo(torrent *TorrentFile) {
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
func PrintPeers(torrent *TorrentFile) {

	// Do HTTP GET request to the available peers
	peers, err := RequestPeers(torrent)
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
func DoPeerHandshake(torrent *TorrentFile, peer *Peer) {

	// Do the handshake
	peerConnection, err := peer.Handshake(torrent.InfoHash)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the handshake
	fmt.Println("Peer ID:", peerConnection.PeerId)
}

// Downloads a piece from a peer and print the piece hash
func DownloadPiece(destFile string, torrent *TorrentFile, pieceIndex int) {

	peers, err := RequestPeers(torrent)
	fmt.Printf("Peers: %v\n", peers)

	// Encodes and hash the info
	fmt.Printf("Info Hash: %x\n", torrent.InfoHash)

	// Get random peer from the peers list
	peer := peers[0]

	// Do the handshake
	peerConnection, err := peer.Handshake(torrent.InfoHash)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Handshake Peer: %s\n", peerConnection.PeerId)

	defer peerConnection.Conn.Close()

	// Exchange multiple peer messages to download the file
	// Wait for bitfield 5 message
	messageType, _ := peerConnection.readMessage()

	if messageType != Bitfield {
		fmt.Println("Bitfield message not received!")
		os.Exit(1)
	}

	// Send interested message
	interestedMessage := []byte{0, 0, 0, 1, 2}
	_, err = peerConnection.Conn.Write(interestedMessage)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Wait for unchoke message
	messageType, _ = peerConnection.readMessage()
	if messageType != Unchoke {
		fmt.Println("Unchoke message not received!")
		os.Exit(1)
	}

	// Request piece
	pieceData, err := peerConnection.RequestPiece(int64(torrent.Info.PieceLen), pieceIndex, int64(torrent.Info.Length))
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
func Download(destFile string, torrent *TorrentFile) {

	peers, err := RequestPeers(torrent)
	fmt.Printf("Peers: %v\n", peers)

	// Encodes and hash the info
	fmt.Printf("Info Hash: %x\n", torrent.InfoHash)

	// Get random peer from the peers list
	peer := peers[0]

	// Do the handshake
	peerConnection, err := peer.Handshake(torrent.InfoHash)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Handshake Peer: %s\n", peerConnection.PeerId)

	defer peerConnection.Conn.Close()

	// Exchange multiple peer messages to download the file
	// Wait for bitfield 5 message
	messageType, _ := peerConnection.readMessage()

	if messageType != Bitfield {
		fmt.Println("Bitfield message not received!")
		os.Exit(1)
	}

	// Send interested message
	interestedMessage := []byte{0, 0, 0, 1, 2}
	_, err = peerConnection.Conn.Write(interestedMessage)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Wait for unchoke message
	messageType, _ = peerConnection.readMessage()
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
		pieceData, err := peerConnection.RequestPiece(int64(torrent.Info.PieceLen), i, int64(torrent.Info.Length))
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
