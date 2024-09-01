package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
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
	infoHashBytes, err := hex.DecodeString(infoHash)
	if err != nil {
		return nil, err
	}

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

type HandShakeRequest struct {
	PeerAddr       IPAddress
	ProtocolStrLen int
	ProtocolString string
	InfoHash       []byte
	PeerID         []byte
}

func (t *Torrent) GetHandShakeRequest(peerAddr IPAddress) (*HandShakeRequest, error) {
	infoHash, err := t.Info.Hash()
	if err != nil {
		return nil, err
	}
	return &HandShakeRequest{
		PeerAddr:       peerAddr,
		ProtocolStrLen: 19,
		ProtocolString: "BitTorrent protocol",
		InfoHash:       []byte(infoHash),
		PeerID:         []byte("00112233445566778899"),
	}, nil
}

type HandShakeResponse struct {
	ProtocolStrLen int
	ProtocolString string
	InfoHash       []byte
	PeerID         []byte
}

const (
	reservedBytesLen = 8
	infoHashLen      = 20
)

func NewHandShakeResponse(b []byte) *HandShakeResponse {
	protocolStrLen := int(b[0])
	protocolStr := b[1 : 1+protocolStrLen]
	infoHash := b[1+protocolStrLen+reservedBytesLen : 1+protocolStrLen+reservedBytesLen+infoHashLen]
	peerID := b[1+protocolStrLen+reservedBytesLen+infoHashLen:]
	return &HandShakeResponse{
		ProtocolStrLen: protocolStrLen,
		ProtocolString: string(protocolStr),
		InfoHash:       infoHash,
		PeerID:         peerID,
	}
}

func (r *HandShakeResponse) StringPeerID() string {
	return fmt.Sprintf("Peer ID: %s", hex.EncodeToString(r.PeerID))
}

func (h *HandShakeRequest) Do() (*HandShakeResponse, net.Conn, error) {
	conn, err := net.Dial("tcp", string(h.PeerAddr))
	if err != nil {
		return nil, nil, err
	}

	var reqBuf bytes.Buffer

	reqBuf.WriteByte(byte(h.ProtocolStrLen))
	reqBuf.WriteString(h.ProtocolString)
	reqBuf.Write(make([]byte, 8)) // reserved bytes
	infoHashBytes, err := hex.DecodeString(string(h.InfoHash))
	if err != nil {
		return nil, nil, err
	}
	reqBuf.Write(infoHashBytes)
	reqBuf.Write(h.PeerID)

	if _, err = conn.Write(reqBuf.Bytes()); err != nil {
		return nil, nil, err
	}

	respBuf := make([]byte, 68)
	if _, err = conn.Read(respBuf); err != nil {
		return nil, nil, err
	}

	return NewHandShakeResponse(respBuf), conn, nil
}

const (
	lengthPrefixLen = 4
)

type PeerMessageType byte

const (
	Choke PeerMessageType = iota
	UnChoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
)

type PeerMessage struct {
	LengthPrefix uint32
	ID           PeerMessageType
	Index        uint32
	Begin        uint32
	Length       uint32
}

func (m *PeerMessage) Send(conn net.Conn) error {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, m); err != nil {
		return err
	}
	if _, err := conn.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func (t *Torrent) DownloadFile(conn net.Conn, index uint32) ([]byte, error) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(conn)

	if _, err := waitUntilMessageType(conn, Bitfield); err != nil {
		return nil, err
	}

	if err := sendMessageType(conn, Interested); err != nil {
		return nil, err
	}

	if _, err := waitUntilMessageType(conn, UnChoke); err != nil {
		return nil, err
	}

	pieceSize := t.Info.PieceLength
	pieceCount := uint32(math.Ceil(float64(t.Info.Length) / float64(pieceSize)))
	if index == pieceCount-1 {
		pieceSize = t.Info.Length % t.Info.PieceLength
	}
	blockSize := 16 * 1024
	blockCount := int(math.Ceil(float64(pieceSize) / float64(blockSize)))
	data := make([]byte, 0)
	for i := 0; i < blockCount; i++ {
		blockLength := blockSize
		if i == blockCount-1 {
			blockLength = pieceSize - ((blockCount - 1) * blockSize)
		}
		msg := &PeerMessage{
			LengthPrefix: 13,
			ID:           Request,
			Index:        index,
			Begin:        uint32(i * blockSize),
			Length:       uint32(blockLength),
		}
		if err := msg.Send(conn); err != nil {
			return nil, err
		}

		payload, err := getPayload(conn)
		if err != nil {
			return nil, err
		}
		data = append(data, payload[9:]...)
	}

	return data, nil
}

// TODO: retry
func waitUntilMessageType(conn net.Conn, msgType PeerMessageType) (payload []byte, err error) {
	lengthPrefixBuf := make([]byte, lengthPrefixLen)
	if _, err := conn.Read(lengthPrefixBuf); err != nil {
		return nil, err
	}

	lengthPrefix := binary.BigEndian.Uint32(lengthPrefixBuf)

	payloadBuf := make([]byte, lengthPrefix)
	if _, err := conn.Read(payloadBuf); err != nil {
		return nil, err
	}

	id := PeerMessageType(payloadBuf[0])
	if id != msgType {
		return nil, fmt.Errorf("expected %v but got %v", msgType, id)
	}
	return payloadBuf, nil
}

func getPayload(conn net.Conn) ([]byte, error) {
	lengthPrefixBuf := make([]byte, lengthPrefixLen)
	if _, err := conn.Read(lengthPrefixBuf); err != nil {
		return nil, err
	}

	lengthPrefix := binary.BigEndian.Uint32(lengthPrefixBuf)

	payloadBuf := make([]byte, lengthPrefix)
	if _, err := io.ReadFull(conn, payloadBuf); err != nil {
		return nil, err
	}

	id := PeerMessageType(payloadBuf[0])
	if id != Piece {
		return nil, fmt.Errorf("expected %v but got %v", Piece, id)
	}
	return payloadBuf, nil
}

func sendMessageType(conn net.Conn, msgType PeerMessageType) error {
	if _, err := conn.Write([]byte{0, 0, 0, 1, byte(msgType)}); err != nil {
		return err
	}
	return nil
}
