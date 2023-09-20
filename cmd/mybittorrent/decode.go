package main

import (
	"fmt"
	"strconv"
	"unicode"
)

// Decodes a Bencode string
func decodeBencodeString(bencodedString string) (interface{}, int, error) {
	var stringLenNumberLenght int

	// fmt.Println("To decode string=", bencodedString)

	for i, char := range bencodedString {
		if char == ':' {
			stringLenNumberLenght = i
			break
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

	// fmt.Println("To decode integer=", bencodedString)

	for i := 1; i < bencodedStringLen; i++ {
		if bencodedString[i] == 'e' {
			integerLen = i
			break
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
	var decodedList []interface{} = make([]interface{}, 0)
	var listLen int = len(bencodedString)

	// fmt.Println("To decode list=", bencodedString)

	// Remove the first (l) and last character (e)
	bencodedPayload := bencodedString[1 : listLen-1]

	for len(bencodedPayload) > 0 {
		decoded, end, err := decodeBencode(bencodedPayload)
		if err != nil {
			return "", -1, err
		}
		decodedList = append(decodedList, decoded)
		bencodedPayload = bencodedPayload[end:]
	}

	return decodedList, listLen, nil
}

// Decodes a Bencode dictionary
func decodeBencodeDictionary(bencodedString string) (interface{}, int, error) {
	var decodedDictionary map[string]interface{} = make(map[string]interface{})
	var dictionaryLen int = len(bencodedString)

	// fmt.Println("To decode dictionary=", bencodedPayload)

	// Remove the first (d) and last character (e)
	bencodedPayload := bencodedString[1 : dictionaryLen-1]

	for len(bencodedPayload) > 0 {

		decodedKey, keyEnd, err := decodeBencode(bencodedPayload)
		if err != nil {
			return "", -1, err
		}

		decodedValue, valueEnd, err := decodeBencode(bencodedPayload[keyEnd:])
		if err != nil {
			return "", -1, err
		}

		decodedDictionary[decodedKey.(string)] = decodedValue
		bencodedPayload = bencodedPayload[keyEnd+valueEnd:]
	}

	return decodedDictionary, dictionaryLen, nil
}

// Decodes a Bencode value
func decodeBencode(bencodedString string) (interface{}, int, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		return decodeBencodeString(bencodedString)
	} else if bencodedString[0] == 'i' {
		return decodeBencodeInteger(bencodedString)
	} else if bencodedString[0] == 'l' {
		return decodeBencodeList(bencodedString)
	} else if bencodedString[0] == 'd' {
		return decodeBencodeDictionary(bencodedString)
	} else {
		return "", -1, fmt.Errorf("Pattern not recognized %s", bencodedString)
	}
}
