package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

type Torrent struct {
	TrackerURL string `json:"announce"`
	CreatedBy  string `json:"created_by"`
	Info       *Info  `json:"info"`
}

func (t *Torrent) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("TrackerURL: %s\n", t.TrackerURL))
	b.WriteString(t.Info.String())
	return b.String()
}

type Info struct {
	Length      uint64 `json:"length"`
	Name        string `json:"name"`
	PieceLength uint64 `json:"piece length"`
	Pieces      string `json:"pieces"`
}

func (i *Info) String() string {
	return fmt.Sprintf("Length: %d\n", i.Length)
}

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
		return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], length + len(lengthStr) + 1, nil
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

	switch command {
	case "decode":
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
	case "info":
		filename := os.Args[2]
		f, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}

		decoded, _, err := decodeBencode(string(f), 0)
		if err != nil {
			log.Fatal(err)
		}

		jsonOutput, err := json.Marshal(decoded)
		if err != nil {
			log.Fatal(err)
		}

		var torrent Torrent
		if err := json.Unmarshal(jsonOutput, &torrent); err != nil {
			log.Fatal(err)
		}

		fmt.Print(torrent.String())
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
