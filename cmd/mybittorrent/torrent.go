package main

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

const PieceByteLength = 20

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
	Pieces      []byte
}

func NewTorrent(decodedBencode map[string]any) (*Torrent, error) {
	trackerURL, ok := decodedBencode["announce"].(string)
	if !ok {
		return nil, errors.New("no announce field in Bencode")
	}
	createdBy, ok := decodedBencode["created by"].(string)
	if !ok {
		createdBy = ""
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
	h, err := i.Hash()
	if err != nil {
		return "", err
	}
	b.WriteString(fmt.Sprintf("Info Hash: %s\n", h))
	b.WriteString(fmt.Sprintf("Piece Length: %d\n", i.PieceLength))
	b.WriteString("Piece Hashes:\n")
	for start := 0; start < len(i.Pieces); start += PieceByteLength {
		piece := i.Pieces[start : start+PieceByteLength]
		b.WriteString(fmt.Sprintf("%s\n", hex.EncodeToString(piece)))
	}
	return b.String(), nil
}

func (i *Info) Hash() (string, error) {
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

func (t *Torrent) Get() ([]byte, error) {
	infoHash, err := t.Info.Hash()
	if err != nil {
		return nil, err
	}
	infoHashBytes, _ := hex.DecodeString(infoHash)

	params := url.Values{}
	params.Add("info_hash", string(infoHashBytes))
	params.Add("peer_id", "00112233445566778899")
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", fmt.Sprint(t.Info.Length))
	params.Add("compact", "1")

	getURL := fmt.Sprintf("%s?%s", t.TrackerURL, params.Encode())
	response, err := http.Get(getURL)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(response.Body)

	return io.ReadAll(response.Body)
}

type IPAddress string

type IPAddresses []IPAddress

func (as IPAddresses) String() string {
	var b strings.Builder
	for _, a := range as {
		b.WriteString(fmt.Sprintf("%s\n", a))
	}
	return b.String()
}

func newIPAddress(ipBytes []byte) IPAddress {
	return IPAddress(fmt.Sprintf("%d.%d.%d.%d:%d", ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3], binary.BigEndian.Uint16(ipBytes[4:6])))
}

func NewIPAddresses(ipBytes []byte) IPAddresses {
	addrs := make([]IPAddress, 0)
	for i := 0; i < len(ipBytes); i += 6 {
		addrs = append(addrs, newIPAddress(ipBytes[i:i+4]))
	}
	return addrs
}

type GetResponse struct {
	Interval int
	Peers    IPAddresses
}

func NewGetResponse(dict map[string]any) (*GetResponse, error) {
	interval, ok := dict["interval"].(int)
	if !ok {
		return nil, errors.New("no interval field in Dict")
	}
	peers, ok := dict["peers"].(string)
	if !ok {
		return nil, errors.New("no peers field in Dict")
	}
	return &GetResponse{
		Interval: interval,
		Peers:    NewIPAddresses([]byte(peers)),
	}, nil
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
