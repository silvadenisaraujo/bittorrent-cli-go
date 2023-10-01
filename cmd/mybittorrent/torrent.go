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

	// Has info
	infoHash, err := info.Hash()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	torrent := TorrentFile{
		Announce: announce,
		Info:     info,
		InfoHash: infoHash,
	}

	return &torrent, nil
}

// Hashes the info dictionary
func (info *Info) Hash() ([]byte, error) {
	encodedInfo, err := encodeBencode(info.MapToDict())
	if err != nil {
		return nil, err
	}
	sha := sha1.New()
	sha.Write([]byte(encodedInfo))
	infoHash := sha.Sum(nil)
	return infoHash, nil
}

// Maps the Info struct to a dictionary
func (info *Info) MapToDict() map[string]interface{} {
	dict := make(map[string]interface{})
	dict["length"] = info.Length
	dict["name"] = info.Name
	dict["piece length"] = info.PieceLen
	dict["pieces"] = info.Pieces
	return dict
}
