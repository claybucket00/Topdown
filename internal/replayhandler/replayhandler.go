package replayhandler

import (
	playerposition "topdown/internal/playerposition"
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
	parser.RegisterEventHandler(rh.onTickDone)

	return rh
}

func (rh *ReplayHandler) onRoundStart(roundStart event.RoundStart) {
	if rh.parser.GameState().IsWarmupPeriod() {
		return
	}
	rh.currentRound = &round.Round{
		Number:          rh.parser.GameState().TotalRoundsPlayed() + 1,
		StartTick:       rh.parser.GameState().IngameTick(),
		PlayerPositions: make(map[int][]playerposition.PlayerPosition),
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

func (rh *ReplayHandler) onTickDone(tickDone event.FrameDone) {
	tick := rh.parser.GameState().IngameTick()

	if rh.currentRound == nil || rh.currentRound.Number != 1 {
		return
	}

	players := rh.parser.GameState().Participants().Playing()
	firstPlayer := players[0]
	rh.currentRound.PlayerPositions[firstPlayer.UserID] = append(rh.currentRound.PlayerPositions[firstPlayer.UserID], playerposition.PlayerPosition{
		Tick: tick,
		X:    firstPlayer.Position().X,
		Y:    firstPlayer.Position().Y,
	})
}

func (rh *ReplayHandler) PrintRounds() {
	for _, r := range rh.rounds {
		println("Round", r.Number, "started at tick", r.StartTick, "and ended at tick", r.EndTick)
	}
}

func (rh *ReplayHandler) PrintPlayerPositionsLength() {
	if len(rh.rounds[0].PlayerPositions) == 0 {
		println("No player positions recorded for round 1")
		return
	}
	for playerID, positions := range rh.rounds[0].PlayerPositions {
		println("Player ID:", playerID, "Positions Length:", len(positions))
	}

}
