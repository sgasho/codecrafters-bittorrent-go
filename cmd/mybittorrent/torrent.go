package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

type Torrent struct {
	TrackerURL string
	CreatedBy  string
	Info       *Info
}

func (t *Torrent) String() (string, error) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Tracker URL: %s\n", t.TrackerURL))
	info, err := t.Info.String()
	if err != nil {
		return "", err
	}
	b.WriteString(info)
	return b.String(), nil
}

type Info struct {
	Length      int
	Name        string
	PieceLength int
	Pieces      []uint8
}

func NewTorrent(decodedBencode map[string]any) (*Torrent, error) {
	trackerURL, ok := decodedBencode["announce"].(string)
	if !ok {
		return nil, errors.New("no announce field in Bencode")
	}
	createdBy, ok := decodedBencode["created by"].(string)
	if !ok {
		return nil, errors.New("no created by field in Bencode")
	}
	info, ok := decodedBencode["info"].(map[string]any)
	if !ok {
		return nil, errors.New("no info field in Bencode")
	}
	length, ok := info["length"].(int)
	if !ok {
		return nil, errors.New("no length field in Bencode")
	}
	name, ok := info["name"].(string)
	if !ok {
		return nil, errors.New("no name field in Bencode")
	}
	pieceLength, ok := info["piece length"].(int)
	if !ok {
		return nil, errors.New("no piece length field in Bencode")
	}
	pieces, ok := info["pieces"].(string)
	if !ok {
		return nil, errors.New("no pieces field in Bencode")
	}
	return &Torrent{
		TrackerURL: trackerURL,
		CreatedBy:  createdBy,
		Info: &Info{
			length,
			name,
			pieceLength,
			[]byte(pieces),
		},
	}, nil
}

func (i *Info) String() (string, error) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Length: %d\n", i.Length))
	h, err := i.SHA1()
	if err != nil {
		return "", err
	}
	b.WriteString(fmt.Sprintf("Info Hash: %s\n", h))
	return b.String(), nil
}

func (i *Info) SHA1() (string, error) {
	h := sha1.New()
	e, err := i.Encode()
	if err != nil {
		return "", err
	}
	h.Write([]byte(e))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (i *Info) Encode() (string, error) {
	return fmt.Sprintf("d6:lengthi%de4:name%d:%s12:piece lengthi%de6:pieces%d:%se",
		i.Length, len(i.Name), i.Name, i.PieceLength, len(i.Pieces), i.Pieces), nil
}

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func DecodeBencode(bencodedString string, start int) (any, int, error) {
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
			decoded, nextStartsAt, err := DecodeBencode(bencodedString, 0)
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
			key, nextStartsAt, err := DecodeBencode(bencodedString, 0)
			if err != nil {
				return nil, 0, err
			}
			bencodedString = bencodedString[nextStartsAt:]
			consumed += nextStartsAt
			value, nextStartsAt, err := DecodeBencode(bencodedString, 0)
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
