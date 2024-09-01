package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

func main() {
	command := os.Args[1]

	switch command {
	case "decode":
		bencodedValue := os.Args[2]

		decoded, _, err := DecodeBencode(bencodedValue, 0)
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

		decoded, _, err := DecodeBencode(string(f), 0)
		if err != nil {
			log.Fatal(err)
		}

		decodedMap, ok := decoded.(map[string]any)
		if !ok {
			log.Fatal("expected map[string]any")
		}
		torrent, err := NewTorrent(decodedMap)
		if err != nil {
			log.Fatal(err)
		}

		info, err := torrent.String()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(info)
	case "peers":
		filename := os.Args[2]
		f, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}

		decoded, _, err := DecodeBencode(string(f), 0)
		if err != nil {
			log.Fatal(err)
		}

		decodedMap, ok := decoded.(map[string]any)
		if !ok {
			log.Fatal("expected map[string]any")
		}

		torrent, err := NewTorrent(decodedMap)
		if err != nil {
			log.Fatal(err)
		}

		got, err := torrent.Get()
		if err != nil {
			log.Fatal(err)
		}

		decodedResp, _, err := DecodeBencode(string(got), 0)
		if err != nil {
			log.Fatal(err)
		}

		decodedRespMap, ok := decodedResp.(map[string]any)
		if !ok {
			log.Fatal("expected map[string]any")
		}

		gotResp, err := NewGetResponse(decodedRespMap)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Print(gotResp.Peers.String())
	case "handshake":
		filename := os.Args[2]
		f, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}

		decoded, _, err := DecodeBencode(string(f), 0)
		if err != nil {
			log.Fatal(err)
		}

		decodedMap, ok := decoded.(map[string]any)
		if !ok {
			log.Fatal("expected map[string]any")
		}

		torrent, err := NewTorrent(decodedMap)
		if err != nil {
			log.Fatal(err)
		}

		req, err := torrent.GetHandShakeRequest(os.Args[3])
		if err != nil {
			log.Fatal(err)
		}

		resp, err := req.Do()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(resp.StringPeerID())
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
