package replayhandler

import (
	framedata "topdown/internal/frames"
	playerposition "topdown/internal/playerposition"
	round "topdown/internal/round"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	event "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// TODO: Track all players not just the first one
var FIRST_PLAYER *common.Player

type ReplayHandler struct {
	parser demoinfocs.Parser
	Rounds []round.Round
	// currentRound *round.Round
	Frames   map[int]*framedata.FrameData // Key: Tick, Val: FrameData for that tick
	MapName  string
	TickRate float64
	prevTick int
}

func NewReplayHandler(parser demoinfocs.Parser) *ReplayHandler {
	rh := &ReplayHandler{
		parser:   parser,
		Rounds:   make([]round.Round, 0),
		Frames:   make(map[int]*framedata.FrameData),
		prevTick: 0.0,
	}

	parser.RegisterNetMessageHandler(rh.getMapName)
	parser.RegisterEventHandler(rh.onRoundStart)
	parser.RegisterEventHandler(rh.onRoundEnd)
	parser.RegisterEventHandler(rh.onTickDone)                   // Player positions
	parser.RegisterEventHandler(rh.onGrenadeProjectileDestroyed) // Grenade positions

	return rh
}

func (rh *ReplayHandler) getMapName(msg *msg.CSVCMsg_ServerInfo) {
	rh.MapName = msg.GetMapName()
	rh.TickRate = rh.parser.TickRate()
}

func (rh *ReplayHandler) onRoundStart(roundStart event.RoundStart) {
	if rh.parser.GameState().IsWarmupPeriod() {
		return
	}

	newRound := round.Round{
		StartTick: rh.parser.GameState().IngameTick(),
	}
	rh.Rounds = append(rh.Rounds, newRound)
	// rh.currentRound = &round.Round{
	// 	Number:    rh.parser.GameState().TotalRoundsPlayed() + 1,
	// 	StartTick: rh.parser.GameState().IngameTick(),
	// 	// PlayerPositions: make(map[int][]playerposition.PlayerPosition),
	// }
}

func (rh *ReplayHandler) onRoundEnd(roundEnd event.RoundEnd) {
	if rh.parser.GameState().IsWarmupPeriod() {
		return
	}
	rh.Rounds[len(rh.Rounds)-1].EndTick = rh.parser.GameState().IngameTick()
	// if rh.currentRound == nil {
	// 	return
	// }
	// rh.currentRound.EndTick = rh.parser.GameState().IngameTick()
	// rh.Rounds = append(rh.Rounds, *rh.currentRound)
	// rh.currentRound = nil
}

func (rh *ReplayHandler) onTickDone(tickDone event.FrameDone) {
	tick := rh.parser.GameState().IngameTick()

	if tick <= rh.prevTick {
		return
	}
	rh.prevTick = tick

	// Only track first round for now
	if len(rh.Rounds) > 1 {
		return
	}

	// TODO: Track all players
	if FIRST_PLAYER == nil {
		players := rh.parser.GameState().Participants().Playing()
		if len(players) > 0 {
			FIRST_PLAYER = players[0]
		} else {
			return // No players to track
		}
	}
	players := rh.parser.GameState().Participants().ByUserID()
	firstPlayer, exists := players[FIRST_PLAYER.UserID]
	if !exists {
		return // Player no longer exists
	}

	// rh.currentRound.PlayerPositions[firstPlayer.UserID] = append(rh.currentRound.PlayerPositions[firstPlayer.UserID], playerposition.PlayerPosition{
	// 	X: firstPlayer.Position().X,
	// 	Y: firstPlayer.Position().Y,
	// })
	// log.Printf("Tick: %d", tick)
	// rh.currentRound.FrameDatum[tick-rh.currentRound.StartTick].PlayerPositions[firstPlayer.UserID] = playerposition.PlayerPosition{
	// 	X: firstPlayer.Position().X,
	// 	Y: firstPlayer.Position().Y,
	// }
	rh.Frames[tick].PlayerPositions[firstPlayer.UserID] = playerposition.PlayerPosition{
		X: firstPlayer.Position().X,
		Y: firstPlayer.Position().Y,
	}

}

func (rh *ReplayHandler) onGrenadeProjectileDestroyed(grenadeDestroyed event.GrenadeProjectileDestroy) {
	// Only track first round for now
	if len(rh.Rounds) > 1 {
		return
	}
	// In theory, onTickEnd event should have already been called and created an entry in FrameDatum for the current tick
	grenadeProjectile := grenadeDestroyed.Projectile

	for _, trajectoryEntry := range grenadeProjectile.Trajectory {
		rh.Frames[trajectoryEntry.Tick].NadePositions[grenadeProjectile.UniqueID()] = playerposition.NadePosition{
			X: trajectoryEntry.Position.X,
			Y: trajectoryEntry.Position.Y,
		}
	}
}

// func (rh *ReplayHandler) PrintRounds() {
// 	for _, r := range rh.Rounds {
// 		println("Round", r.Number, "started at tick", r.StartTick, "and ended at tick", r.EndTick)
// 	}
// }

// func (rh *ReplayHandler) PrintPlayerPositionsLength() {
// 	if len(rh.Rounds[0].PlayerPositions) == 0 {
// 		println("No player positions recorded for round 1")
// 		return
// 	}
// 	for playerID, positions := range rh.Rounds[0].PlayerPositions {
// 		println("Player ID:", playerID, "Positions Length:", len(positions))
// 	}

// }
