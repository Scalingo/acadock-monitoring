package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/Scalingo/go-netstat"
)

func main() {
	stats, err := netstat.Stats()
	if err != nil {
		log.Println("Error when getting network stats", err)
		os.Exit(-1)
	}
	json.NewEncoder(os.Stdout).Encode(&stats)
}
