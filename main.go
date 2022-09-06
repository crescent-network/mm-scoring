package main

import (
	"os"

	"github.com/crescent-network/mm-scoring/cmd"
)

func main() {
	if err := cmd.NewScoringCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
