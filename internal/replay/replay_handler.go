package replay

import (
	framedata "topdown/internal/frames"
	metadata "topdown/internal/metadata"
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
	parser         demoinfocs.Parser
	Rounds         []round.Round
	currentRound   *round.Round
	Frames         map[int]*framedata.FrameData // Key: Tick, Val: FrameData for that tick
	MapName        string
	TickRate       float64
	prevTick       int
	mapMetdata     metadata.MapMetadata
	PlayerMetadata map[int]metadata.PlayerMetadata // Key: playerId, Val: PlayerMetadata
	NadeMetadata   map[int64]metadata.NadeMetadata // Key: nadeId, Val: NadeMetadata
}

func NewReplayHandler(parser demoinfocs.Parser) *ReplayHandler {
	rh := &ReplayHandler{
		parser:         parser,
		Rounds:         []round.Round{},
		Frames:         make(map[int]*framedata.FrameData),
		PlayerMetadata: make(map[int]metadata.PlayerMetadata),
		NadeMetadata:   make(map[int64]metadata.NadeMetadata),
		prevTick:       0.0,
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

	rh.mapMetdata = metadata.GetMapMetadata(rh.MapName)
}

func (rh *ReplayHandler) getFrame(tick int) *framedata.FrameData {
	frame, ok := rh.Frames[tick]
	if !ok {
		frame = &framedata.FrameData{
			PlayerPositions: make(map[int]playerposition.PlayerPosition),
			NadePositions:   make(map[int64]playerposition.NadePosition),
		}
		rh.Frames[tick] = frame
	}
	return frame
}

func (rh *ReplayHandler) onRoundStart(roundStart event.RoundStart) {
	if rh.parser.GameState().IsWarmupPeriod() {
		return
	}

	rh.currentRound = &round.Round{
		StartTick: rh.parser.GameState().IngameTick(),
		// PlayerPositions: make(map[int][]playerposition.PlayerPosition),
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
	if rh.currentRound.StartTick == rh.currentRound.EndTick {
		return // Skip rounds that start and end on the same tick
	}
	rh.Rounds = append(rh.Rounds, *rh.currentRound)
	rh.currentRound = nil

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
		for _, player := range players {
			rh.PlayerMetadata[player.UserID] = metadata.PlayerMetadata{
				Name: player.Name,
			}
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

	frame := rh.getFrame(tick)
	radarX, radarY := rh.mapMetdata.WorldToRadarCoords(firstPlayer.Position().X, firstPlayer.Position().Y)
	frame.PlayerPositions[firstPlayer.UserID] = playerposition.PlayerPosition{
		X: radarX,
		Y: radarY,
	}

}

func (rh *ReplayHandler) onGrenadeProjectileDestroyed(grenadeDestroyed event.GrenadeProjectileDestroy) {
	// Record nades for all rounds
	// if len(rh.Rounds) > 0 {
	// 	return
	// }
	grenadeProjectile := grenadeDestroyed.Projectile
	rh.NadeMetadata[grenadeProjectile.UniqueID()] = metadata.NadeMetadata{
		Type:    grenadeProjectile.WeaponInstance.String(),
		Thrower: grenadeProjectile.Thrower.UserID,
	}

	// var prevTrajectoryEntry *common.TrajectoryEntry = nil
	for _, trajectoryEntry := range grenadeProjectile.Trajectory {
		frame := rh.getFrame(trajectoryEntry.Tick)
		radarX, radarY := rh.mapMetdata.WorldToRadarCoords(trajectoryEntry.Position.X, trajectoryEntry.Position.Y)
		frame.NadePositions[grenadeProjectile.UniqueID()] = playerposition.NadePosition{
			X: radarX,
			Y: radarY,
		}
		// if prevTrajectoryEntry != nil && trajectoryEntry.Tick > prevTrajectoryEntry.Tick+1 {
		// 	// Interpolate positions for ticks between prevTick and trajectoryEntry.Tick
		// 	for t := prevTrajectoryEntry.Tick + 1; t < trajectoryEntry.Tick; t++ {
		// 		ratio := float64(t-prevTrajectoryEntry.Tick) / float64(trajectoryEntry.Tick-prevTrajectoryEntry.Tick)
		// 		interpolatedX := prevTrajectoryEntry.Position.X + ratio*(trajectoryEntry.Position.X-prevTrajectoryEntry.Position.X)
		// 		interpolatedY := prevTrajectoryEntry.Position.Y + ratio*(trajectoryEntry.Position.Y-prevTrajectoryEntry.Position.Y)
		// 		radarX, radarY := rh.mapMetdata.WorldToRadarCoords(interpolatedX, interpolatedY)
		// 		interpolatedFrame := rh.getFrame(t)
		// 		interpolatedFrame.NadePositions[grenadeProjectile.UniqueID()] = playerposition.NadePosition{
		// 			X: radarX,
		// 			Y: radarY,
		// 		}
		// 	}
		// }
		// prevTrajectoryEntry = &trajectoryEntry
	}
}

func (rh *ReplayHandler) PrintNadePositions() {
	for i, round := range rh.Rounds {
		println("Round", i+1, "nade positions:")
		nadePositionsLengths := 0
		for tick := round.StartTick; tick <= round.EndTick; tick++ {
			frameData, exists := rh.Frames[tick]
			if exists {
				nadePositionsLengths += len(frameData.NadePositions)
			}
		}
		println("Nade positions recorded:", nadePositionsLengths)
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
