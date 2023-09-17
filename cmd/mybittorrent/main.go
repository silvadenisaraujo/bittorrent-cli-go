package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

// Decodes a Bencode string
func decodeBencodeString(bencodedString string) (interface{}, int, error) {
	var stringLenNumberLenght int

	for i, char := range bencodedString {
		if char == ':' {
			stringLenNumberLenght = i
		}
	}

	stringLen, _ := strconv.Atoi(bencodedString[:stringLenNumberLenght])

	start := stringLenNumberLenght + 1
	end := start + stringLen

	return bencodedString[start:end], end, nil
}

// Decodes a Bencode integer
func decodeBencodeInteger(bencodedString string) (interface{}, int, error) {
	bencodedStringLen := len(bencodedString)
	var integerLen int

	for i := 1; i < bencodedStringLen; i++ {
		if bencodedString[i] == 'e' {
			integerLen = i
		}
	}

	start := 1
	end := integerLen

	num, err := strconv.Atoi(bencodedString[start:end])
	if err != nil {
		return "", -1, err
	}
	return num, end + 1, nil // +1 to include the 'e' character
}

// Decodes a Bencode list
func decodeBencodeList(bencodedString string) (interface{}, int, error) {
	var decodedList []interface{}
	// Remove the first (l) and last character (e)
	bencodedPayload := bencodedString[1 : len(bencodedString)-1]

	for len(bencodedPayload) > 0 {
		decoded, end, err := decodeBencode(bencodedPayload)
		if err != nil {
			return "", -1, err
		}
		bencodedPayload = bencodedPayload[end:]
		decodedList = append(decodedList, decoded)
	}

	return decodedList, -1, nil
}

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string) (interface{}, int, error) {
	bencodedStringLen := len(bencodedString)

	if unicode.IsDigit(rune(bencodedString[0])) {
		return decodeBencodeString(bencodedString)
	} else if bencodedString[0] == 'i' && bencodedString[bencodedStringLen-1] == 'e' {
		return decodeBencodeInteger(bencodedString)
	} else if bencodedString[0] == 'l' {
		return decodeBencodeList(bencodedString)
	} else {
		return "", -1, fmt.Errorf("Pattern not recognized")
	}
}

func main() {
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
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
