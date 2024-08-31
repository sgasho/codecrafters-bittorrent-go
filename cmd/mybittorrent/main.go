package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/torrent"
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

func main() {
	command := os.Args[1]

	switch command {
	case "decode":
		bencodedValue := os.Args[2]

		decoded, _, err := torrent.DecodeBencode(bencodedValue, 0)
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

		decoded, _, err := torrent.DecodeBencode(string(f), 0)
		if err != nil {
			log.Fatal(err)
		}

		m, ok := decoded.(map[string]any)
		if !ok {
			log.Fatal(err)
		}

		ti := &torrent.Info{}
		info, ok := m["info"].(map[string]any)
		if !ok {
			log.Fatal(err)
		}
		ti.Length = info["length"].(int)
		ti.Name = info["name"].(string)
		ti.PieceLength = info["piece length"].(int)
		ti.Pieces = []byte(info["pieces"].(string))

		fmt.Println(ti.String())
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
