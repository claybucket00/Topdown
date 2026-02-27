package main

import (
	"log"
	"os"
	"topdown/internal/replayhandler"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
)

// https://www.hltv.org/matches/2382614/spirit-vs-mouz-blasttv-austin-major-2025
var DEMO_PATH = "./demos/spirit-vs-mouz-m1-mirage.dem"

func main() {
	f, err := os.Open(DEMO_PATH)
	if err != nil {
		log.Panicf("Failed to open demo: %v", err)
	}
	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	rh := replayhandler.NewReplayHandler(p)

	err = p.ParseToEnd()
	if err != nil {
		log.Panicf("Failed to parse demo: %v", err)
	}

	// rh.PrintRounds()
	rh.PrintPlayerPositionsLength()
}
