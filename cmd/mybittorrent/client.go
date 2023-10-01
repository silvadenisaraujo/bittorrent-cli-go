package main

import "fmt"

func decodeValue(bencodedValue string) (interface{}, error) {
	decoded, _, err := decodeBencode(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return decoded, nil
}
