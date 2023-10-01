package main

import (
	"fmt"
	"sort"
	"strconv"
)

// Encodes a Bencode string
func encodeBencodeString(value interface{}) (string, error) {
	var encodedString string = strconv.Itoa(len(value.(string))) + ":" + value.(string)
	return encodedString, nil
}

// Encodes a Bencode integer
func encodeBencodeInteger(value interface{}) (string, error) {
	var encodedInteger string = "i" + strconv.Itoa(value.(int)) + "e"
	return encodedInteger, nil
}

// Encodes a Bencode list
func encodeBencodeList(value interface{}) (string, error) {
	var encodedList string = "l"
	var err error

	for _, value := range value.([]interface{}) {
		encodedValue, _ := encodeBencode(value)
		encodedList += encodedValue
	}

	encodedList += "e"

	return encodedList, err
}

// Encodes a Bencode list of strings
func encodeBencodeStrList(value []string) (string, error) {
	var encodedList string = "l"
	var err error

	for _, value := range value {
		encodedValue, _ := encodeBencodeString(value)
		encodedList += encodedValue
	}

	encodedList += "e"

	return encodedList, err
}

// Encodes a Bencode dictionary
func encodeBencodeDictionary(value interface{}) (string, error) {
	var encodedDictionary string = "d"
	var err error

	// Sort the keys to guarantee the same hash
	var keys []string
	for key := range value.(map[string]interface{}) {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {

		// Encode the key
		encodedKey, err := encodeBencode(key)
		if err != nil {
			return "", err
		}

		// Encode the value
		value := value.(map[string]interface{})[key]
		encodedValue, err := encodeBencode(value)
		if err != nil {
			return "", err
		}

		encodedDictionary += encodedKey
		encodedDictionary += encodedValue
	}

	encodedDictionary += "e"

	return encodedDictionary, err
}

// Encodes a Bencode value
func encodeBencode(value interface{}) (string, error) {
	if value == nil {
		return "", nil
	}
	switch value.(type) {
	case string:
		return encodeBencodeString(value)
	case int:
		return encodeBencodeInteger(value)
	case []interface{}:
		return encodeBencodeList(value)
	case []string:
		return encodeBencodeStrList(value.([]string)) // assert type to []string
	case map[string]interface{}:
		return encodeBencodeDictionary(value)
	default:
		return "", fmt.Errorf("Type not recognized %T", value)
	}
}
