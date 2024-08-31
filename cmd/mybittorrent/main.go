package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string, start int) (any, int, error) {
	bencodedString = bencodedString[start:]
	switch {
	case unicode.IsDigit(rune(bencodedString[0])):
		var firstColonIndex int

		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == ':' {
				firstColonIndex = i
				break
			}
		}

		lengthStr := bencodedString[:firstColonIndex]

		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return "", 0, err
		}
		return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], length + 2, nil
	case bencodedString[0] == 'i':
		var eIndex int

		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == 'e' {
				eIndex = i
				break
			}
		}

		number, err := strconv.Atoi(bencodedString[1:eIndex])
		if err != nil {
			return "", 0, err
		}
		return number, eIndex + 1, nil
	case bencodedString[0] == 'l':
		bencodedString = bencodedString[1:]
		elements := make([]any, 0)
		consumed := 2 // l ... e -> l, e -> consume 2 chars
		for bencodedString[0] != 'e' {
			decoded, nextStartsAt, err := decodeBencode(bencodedString, 0)
			if err != nil {
				return nil, 0, err
			}
			elements = append(elements, decoded)
			if nextStartsAt >= len(bencodedString) {
				return elements, nextStartsAt, nil
			}
			bencodedString = bencodedString[nextStartsAt:]
			consumed += nextStartsAt
		}
		return elements, consumed, nil
	case bencodedString[0] == 'd':
		bencodedString = bencodedString[1:]
		elements := make(map[string]any)
		consumed := 2
		for bencodedString[0] != 'e' {
			key, nextStartsAt, err := decodeBencode(bencodedString, 0)
			if err != nil {
				return nil, 0, err
			}
			bencodedString = bencodedString[nextStartsAt:]
			consumed += nextStartsAt
			value, nextStartsAt, err := decodeBencode(bencodedString, 0)
			if err != nil {
				return nil, 0, err
			}
			elements[key.(string)] = value
			if nextStartsAt >= len(bencodedString) {
				return elements, nextStartsAt, nil
			}
			bencodedString = bencodedString[nextStartsAt:]
			consumed += nextStartsAt
		}
		return elements, consumed, nil
	}
	return nil, 0, fmt.Errorf("invalid input")
}

func main() {
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, _, err := decodeBencode(bencodedValue, 0)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, err := json.Marshal(decoded)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
