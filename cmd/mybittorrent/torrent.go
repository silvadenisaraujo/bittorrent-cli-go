package main

import (
	"crypto/sha1"
	"fmt"
	"os"
)

// TorrentFile represents a torrent file
type TorrentFile struct {
	Announce string
	Info     Info
	InfoHash []byte
	Path     string
}

type Info struct {
	Length   int
	Name     string
	PieceLen int
	Pieces   []string
}

// Creates a TorrentFile instance from a torrent file path
func ParseFile(filepath string) *TorrentFile {
	torrent, err := parseFile(filepath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return torrent
}

// Parses a torrent file
func parseFile(filepath string) (*TorrentFile, error) {
	// Open the torrent file
	file, err := os.Open(filepath)
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

	// Get values
	announce := decoded.(map[string]interface{})["announce"].(string)
	infoDecoded := decoded.(map[string]interface{})["info"].(map[string]interface{})
	length := infoDecoded["length"].(int)
	name := infoDecoded["name"].(string)
	pieceLen := infoDecoded["piece length"].(int)
	piecesStr := infoDecoded["pieces"].(string)
	pieces := []string{}
	for i := 0; i < len(piecesStr); i += 20 {
		pieces = append(pieces, piecesStr[i:i+20])
	}

	info := Info{
		Length:   length,
		Name:     name,
		PieceLen: pieceLen,
		Pieces:   pieces,
	}

	// Hash info
	infoHash, err := hashInfo(infoDecoded)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	torrent := TorrentFile{
		Announce: announce,
		Info:     info,
		InfoHash: infoHash,
		Path:     filepath,
	}

	return &torrent, nil
}

// Hashes the info dictionary
func hashInfo(infoDecoded map[string]interface{}) ([]byte, error) {
	encodedInfo, err := encodeBencode(infoDecoded)
	if err != nil {
		return nil, err
	}
	sha := sha1.New()
	sha.Write([]byte(encodedInfo))
	infoHash := sha.Sum(nil)
	return infoHash, nil
}
