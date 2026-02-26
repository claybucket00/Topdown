package replayhandler

import (
	round "topdown/internal/round"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	event "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type ReplayHandler struct {
	parser       demoinfocs.Parser
	rounds       []round.Round
	currentRound *round.Round
}

func NewReplayHandler(parser demoinfocs.Parser) *ReplayHandler {
	rh := &ReplayHandler{
		parser: parser,
		rounds: make([]round.Round, 0),
	}

	parser.RegisterEventHandler(rh.onRoundStart)
	parser.RegisterEventHandler(rh.onRoundEnd)

	return rh
}

func (rh *ReplayHandler) onRoundStart(roundStart event.RoundStart) {
	if rh.parser.GameState().IsWarmupPeriod() {
		return
	}
	rh.currentRound = &round.Round{
		Number:    rh.parser.GameState().TotalRoundsPlayed() + 1,
		StartTick: rh.parser.GameState().IngameTick(),
	}
}

func (rh *ReplayHandler) onRoundEnd(roundEnd event.RoundEnd) {
	if rh.parser.GameState().IsWarmupPeriod() {
		return
	}
	if rh.currentRound == nil {
		return
	}
	rh.currentRound.EndTick = rh.parser.GameState().IngameTick()
	rh.rounds = append(rh.rounds, *rh.currentRound)
	rh.currentRound = nil
}

func (rh *ReplayHandler) PrintRounds() {
	for _, r := range rh.rounds {
		println("Round", r.Number, "started at tick", r.StartTick, "and ended at tick", r.EndTick)
	}
}
