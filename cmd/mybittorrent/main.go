package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
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

		req, err := torrent.GetHandShakeRequest(IPAddress(os.Args[3]))
		if err != nil {
			log.Fatal(err)
		}

		resp, _, err := req.Do()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(resp.StringPeerID())
	case "download_piece":
		var torrentFile, outputPath string
		var index int
		var err error
		if os.Args[2] == "-o" {
			torrentFile = os.Args[4]
			outputPath = os.Args[3]
			index, err = strconv.Atoi(os.Args[5])
			if err != nil {
				log.Fatal(err)
			}
		} else {
			torrentFile = os.Args[3]
			outputPath = "."
			index, err = strconv.Atoi(os.Args[4])
			if err != nil {
				log.Fatal(err)
			}
		}

		f, err := os.ReadFile(torrentFile)
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

		if len(gotResp.Peers) == 0 {
			log.Fatal("no peers found")
		}

		hReq, err := torrent.GetHandShakeRequest(gotResp.Peers[0])
		if err != nil {
			log.Fatal(err)
		}

		_, conn, err := hReq.Do()
		if err != nil {
			log.Fatal(err)
		}

		data, err := torrent.DownloadPiece(conn, uint32(index))
		if err != nil {
			log.Fatal(err)
		}

		file, err := os.Create(outputPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		if _, err := file.Write(data); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Piece downloaded to %s.\n", outputPath)
	case "download":
		var torrentFile, outputPath string
		var err error
		if os.Args[2] == "-o" {
			torrentFile = os.Args[4]
			outputPath = os.Args[3]
		} else {
			torrentFile = os.Args[3]
			outputPath = "."
		}

		f, err := os.ReadFile(torrentFile)
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

		if len(gotResp.Peers) == 0 {
			log.Fatal("no peers found")
		}

		hReq, err := torrent.GetHandShakeRequest(gotResp.Peers[0])
		if err != nil {
			log.Fatal(err)
		}

		_, conn, err := hReq.Do()
		if err != nil {
			log.Fatal(err)
		}

		data, err := torrent.DownloadFile(conn)
		if err != nil {
			log.Fatal(err)
		}

		file, err := os.Create(outputPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		if _, err := file.Write(data); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Downloaded %s to %s.\n", torrentFile, outputPath)
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
