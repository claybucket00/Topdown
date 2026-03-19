package replay

import (
	"fmt"
	events "topdown/internal/events"
	framedata "topdown/internal/frames"
	metadata "topdown/internal/metadata"
	playerposition "topdown/internal/playerposition"
	round "topdown/internal/round"
	"topdown/internal/utility"

	r2 "github.com/golang/geo/r2"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	event "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	ulid "github.com/oklog/ulid/v2"
)

type ReplayHandler struct {
	parser         demoinfocs.Parser
	Rounds         []round.Round
	currentRound   *round.Round
	Frames         map[int]*framedata.FrameData // Key: Tick, Val: FrameData for that tick
	Events         []events.GameEvent
	MapName        string
	prevTick       int
	dead           map[int]struct{} // Map to track players who are dead so we can skip collecting their position
	mapMetdata     metadata.MapMetadata
	PlayerMetadata map[int]metadata.PlayerMetadata     // Key: playerId, Val: PlayerMetadata
	NadeMetadata   map[ulid.ULID]metadata.NadeMetadata // TESTING: ULID instead of UniqueID()
	// NadeMetadata   map[int64]metadata.NadeMetadata // Key: nadeId, Val: NadeMetadata
	// EntityIDToNadeID map[int]int64                   // Key: EntityID, Val: NadeID (UniqueID from onGrenadeProjectileDestroyed). Needed because GrenadeEvents do not generate a unique ID
}

func NewReplayHandler(parser demoinfocs.Parser) *ReplayHandler {
	rh := &ReplayHandler{
		parser:         parser,
		Rounds:         []round.Round{},
		Frames:         make(map[int]*framedata.FrameData),
		PlayerMetadata: make(map[int]metadata.PlayerMetadata),
		NadeMetadata:   make(map[ulid.ULID]metadata.NadeMetadata),
		dead:           make(map[int]struct{}),
		prevTick:       0,
	}

	parser.RegisterNetMessageHandler(rh.getMapName)
	parser.RegisterEventHandler(rh.onRoundStart)
	parser.RegisterEventHandler(rh.onRoundEnd)
	parser.RegisterEventHandler(rh.onTickDone)                   // Player positions
	parser.RegisterEventHandler(rh.onGrenadeProjectileDestroyed) // Grenade positions
	parser.RegisterEventHandler(rh.onSmokeStart)                 // Smoke start events
	parser.RegisterEventHandler(rh.onSmokeEnd)                   // Smoke end events
	parser.RegisterEventHandler(rh.onKill)                       // Track dead players
	parser.RegisterEventHandler(rh.onFlashExplode)
	parser.RegisterEventHandler(rh.onHeExplode)
	parser.RegisterEventHandler(rh.onPlayerTeamChange)
	parser.RegisterEventHandler(rh.onPlayerDamage)
	// parser.RegisterEventHandler(rh.onFireGrenadeStart) 		 // Doesn't seem to trigger

	return rh
}

func (rh *ReplayHandler) getMapName(msg *msg.CSVCMsg_ServerInfo) {
	rh.MapName = msg.GetMapName()
	// rh.TickRate = rh.parser.TickRate() // Doesn't appear deterministic??? Running multiple times on same demo file sometimes results in zero.

	rh.mapMetdata = metadata.GetMapMetadata(rh.MapName)
}

func (rh *ReplayHandler) getFrame(tick int) *framedata.FrameData {
	frame, ok := rh.Frames[tick]
	if !ok {
		frame = &framedata.FrameData{
			PlayerPositions: make(map[int]playerposition.PlayerPosition),
			NadePositions:   make(map[ulid.ULID]playerposition.NadePosition),
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
	}
	players := rh.parser.GameState().Participants().Playing()
	rh.currentRound.PlayerTeams = make(map[int]common.Team)
	for _, player := range players {
		rh.currentRound.PlayerTeams[player.UserID] = player.Team
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

	// Set round score on round end as round start event does not expose score information
	winner := roundEnd.Winner
	if winner == common.TeamTerrorists {
		rh.currentRound.Score.T = roundEnd.WinnerState.Score()
		rh.currentRound.Score.CT = roundEnd.LoserState.Score()
	} else {
		rh.currentRound.Score.T = roundEnd.LoserState.Score()
		rh.currentRound.Score.CT = roundEnd.WinnerState.Score()
	}
	rh.Rounds = append(rh.Rounds, *rh.currentRound)
	rh.currentRound = nil
	rh.dead = make(map[int]struct{}) // Reset dead players for next round. We do this on round end instead of round start because we cannot guarantee round start will process before onTickDone

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

	// Only track first two rounds for now
	if len(rh.Rounds) > 1 {
		return
	}

	players := rh.parser.GameState().Participants().Playing()
	if len(players) > len(rh.PlayerMetadata) {
		for _, player := range players {
			if _, exists := rh.PlayerMetadata[player.UserID]; !exists {
				rh.PlayerMetadata[player.UserID] = metadata.PlayerMetadata{
					Name: player.Name,
				}
			}
		}
	}

	frame := rh.getFrame(tick)
	for _, player := range players {
		// Only track alive players to compress replay size
		if _, isDead := rh.dead[player.UserID]; !isDead && player.IsAlive() {
			radarX, radarY := rh.mapMetdata.WorldToRadarCoords(player.Position().X, player.Position().Y)
			yaw := player.ViewDirectionX()
			frame.PlayerPositions[player.UserID] = playerposition.PlayerPosition{
				X:   radarX,
				Y:   radarY,
				Yaw: yaw,
			}
		}
	}

	// TODO: Depending on the complexity, might need to refactor this function to behave like smokes. Currently we track infernos on ticks so we can get accurate 2D convex hulls for rendering the spread.
	// Track infernos.
	currentInfernos := rh.parser.GameState().Infernos()
	if len(currentInfernos) != 0 {
		for _, inferno := range currentInfernos {
			fires := inferno.Fires().Active().List()
			points := make([]r2.Point, len(fires))
			for i, fire := range fires {
				radarX, radarY := rh.mapMetdata.WorldToRadarCoords(fire.Vector.X, fire.Vector.Y)
				points[i] = r2.Point{X: radarX, Y: radarY}
			}
			// for i, point := range points {
			// 	radarX, radarY := rh.mapMetdata.WorldToRadarCoords(point.X, point.Y)
			// 	points[i].X = radarX
			// 	points[i].Y = radarY
			// }
			infernoData := events.InfernoEvent{
				Points: points,
				NadeId: inferno.UniqueID(),
			}
			rh.Events = append(rh.Events, events.GameEvent{
				Tick: tick,
				Type: events.EventInferno,
				Data: infernoData,
			})
		}
	}

}

// func (rh *ReplayHandler) onInfernoStart(infernoStart event.InfernoStart) {
// 	tick := rh.parser.GameState().IngameTick()

// }

func (rh *ReplayHandler) onGrenadeProjectileDestroyed(grenadeDestroyed event.GrenadeProjectileDestroy) {
	grenadeProjectile := grenadeDestroyed.Projectile
	rh.NadeMetadata[grenadeProjectile.WeaponInstance.UniqueID2()] = metadata.NadeMetadata{
		Type:    grenadeProjectile.WeaponInstance.String(),
		Thrower: grenadeProjectile.Thrower.UserID,
	}

	for _, trajectoryEntry := range grenadeProjectile.Trajectory {
		frame := rh.getFrame(trajectoryEntry.Tick)
		radarX, radarY := rh.mapMetdata.WorldToRadarCoords(trajectoryEntry.Position.X, trajectoryEntry.Position.Y)
		frame.NadePositions[grenadeProjectile.WeaponInstance.UniqueID2()] = playerposition.NadePosition{
			X: radarX,
			Y: radarY,
		}
	}
}

func (rh *ReplayHandler) onKill(kill event.Kill) {
	if kill.Victim == nil {
		return // Can be nil if demo is partially corrupt
	}
	rh.dead[kill.Victim.UserID] = struct{}{}
	var attackerID *playerposition.PlayerID
	if kill.Killer != nil {
		attackerID = utility.Ptr[playerposition.PlayerID](playerposition.PlayerID(kill.Killer.UserID))
	}
	var assisterID *playerposition.PlayerID
	if kill.Assister != nil {
		assisterID = utility.Ptr[playerposition.PlayerID](playerposition.PlayerID(kill.Assister.UserID))
	}

	rh.Events = append(rh.Events, events.GameEvent{
		Tick: rh.parser.GameState().IngameTick(),
		Type: events.EventKill,
		Data: events.KillEvent{
			VictimID:      kill.Victim.UserID,
			AttackerID:    attackerID,
			AssisterID:    assisterID,
			Weapon:        kill.Weapon.String(),
			IsWallbang:    kill.IsWallBang(),
			IsHeadshot:    kill.IsHeadshot,
			AssistedFlash: kill.AssistedFlash,
			AttackerBlind: kill.AttackerBlind,
			NoScope:       kill.NoScope,
			ThroughSmoke:  kill.ThroughSmoke,
		},
	})
}

func (rh *ReplayHandler) onPlayerDamage(playerHurt event.PlayerHurt) {
	if playerHurt.Player == nil {
		return
	}
	tick := rh.parser.GameState().IngameTick()
	playerDamage := events.DamageEvent{
		PlayerID:          utility.Ptr[playerposition.PlayerID](playerposition.PlayerID(playerHurt.Player.UserID)),
		Health:            playerHurt.Health,
		Armor:             playerHurt.Armor,
		HealthDamageTaken: playerHurt.HealthDamageTaken,
		ArmorDamageTaken:  playerHurt.ArmorDamageTaken,
	}
	rh.Events = append(rh.Events, events.GameEvent{
		Tick: tick,
		Type: events.EventDamage,
		Data: playerDamage,
	})
}

func (rh *ReplayHandler) onPlayerTeamChange(playerTeamChange event.PlayerTeamChange) {
	tick := rh.parser.GameState().IngameTick()
	newTeamChange := events.TeamChangeEvent{
		PlayerID: playerTeamChange.Player.UserID,
		Team:     playerTeamChange.NewTeam,
	}
	rh.Events = append(rh.Events, events.GameEvent{
		Tick: tick,
		Type: events.EventTeamChange,
		Data: newTeamChange,
	})
}

func (rh *ReplayHandler) onSmokeStart(smokeStart event.SmokeStart) {
	tick := rh.parser.GameState().IngameTick()
	radarX, radarY := rh.mapMetdata.WorldToRadarCoords(smokeStart.Position.X, smokeStart.Position.Y)
	newSmokeStart := events.SmokeEvent{
		X:      radarX,
		Y:      radarY,
		NadeId: smokeStart.Grenade.UniqueID2(), // Using ULID as new Projectiles are not generated for GrenadeEvents
	}
	rh.Events = append(rh.Events, events.GameEvent{
		Tick: tick,
		Type: events.EventSmokeStart,
		Data: newSmokeStart,
	})
}

func (rh *ReplayHandler) onSmokeEnd(smokeExpired event.SmokeExpired) {
	tick := rh.parser.GameState().IngameTick()
	radarX, radarY := rh.mapMetdata.WorldToRadarCoords(smokeExpired.Position.X, smokeExpired.Position.Y)
	newSmokeEnd := events.SmokeEvent{
		X:      radarX,
		Y:      radarY,
		NadeId: smokeExpired.Grenade.UniqueID2(), // Using ULID as new Projectiles are not generated for GrenadeEvents
	}
	rh.Events = append(rh.Events, events.GameEvent{
		Tick: tick,
		Type: events.EventSmokeEnd,
		Data: newSmokeEnd,
	})
}

func (rh *ReplayHandler) onFlashExplode(flashExplode event.FlashExplode) {
	tick := rh.parser.GameState().IngameTick()
	radarX, radarY := rh.mapMetdata.WorldToRadarCoords(flashExplode.Position.X, flashExplode.Position.Y)
	newFlashEvent := events.GrenadeEvent{
		X:      radarX,
		Y:      radarY,
		NadeId: flashExplode.Grenade.UniqueID2(), // Using ULID as new Projectiles are not generated for GrenadeEvents
	}
	rh.Events = append(rh.Events, events.GameEvent{
		Tick: tick,
		Type: events.EventFlash,
		Data: newFlashEvent,
	})
}

func (rh *ReplayHandler) onHeExplode(heExplode event.HeExplode) {
	tick := rh.parser.GameState().IngameTick()
	radarX, radarY := rh.mapMetdata.WorldToRadarCoords(heExplode.Position.X, heExplode.Position.Y)
	newHeEvent := events.GrenadeEvent{
		X:      radarX,
		Y:      radarY,
		NadeId: heExplode.Grenade.UniqueID2(),
	}
	rh.Events = append(rh.Events, events.GameEvent{
		Tick: tick,
		Type: events.EventHe,
		Data: newHeEvent,
	})
}

func (rh *ReplayHandler) GetTickRate() float64 {
	return rh.parser.TickRate()
}

// func (rh *ReplayHandler) onFireGrenadeStart(fireGrenadeStart event.FireGrenadeStart) {
// 	tick := rh.parser.GameState().IngameTick()
// 	radarX, radarY := rh.mapMetdata.WorldToRadarCoords(fireGrenadeStart.Position.X, fireGrenadeStart.Position.Y)
// 	if fireGrenadeStart.Grenade == nil {
// 		// println("No grenade found for fire start")
// 		return
// 	}
// 	newFireStart := events.GrenadeEvent{
// 		X:      radarX,
// 		Y:      radarY,
// 		NadeId: fireGrenadeStart.Grenade.UniqueID2(), // Using ULID as new Projectiles are not generated for GrenadeEvents
// 	}
// 	rh.Events = append(rh.Events, events.GameEvent{
// 		Tick: tick,
// 		Type: events.EventInfernoStart,
// 		Data: newFireStart,
// 	})
// }

// func (rh *ReplayHandler) onFireGrenadeEnd(fireGrenadeEnd event.FireGrenadeExpired) {
// 	tick := rh.parser.GameState().IngameTick()
// 	radarX, radarY := rh.mapMetdata.WorldToRadarCoords(fireGrenadeEnd.Position.X, fireGrenadeEnd.Position.Y)
// 	if fireGrenadeEnd.Grenade == nil {
// 		// println("No grenade found for fire end")
// 		return
// 	}
// 	newFireStart := events.GrenadeEvent{
// 		X:      radarX,
// 		Y:      radarY,
// 		NadeId: fireGrenadeEnd.Grenade.UniqueID2(), // Using ULID as new Projectiles are not generated for GrenadeEvents
// 	}
// 	rh.Events = append(rh.Events, events.GameEvent{
// 		Tick: tick,
// 		Type: events.EventInfernoStart,
// 		Data: newFireStart,
// 	})
// }

func (rh *ReplayHandler) CheckNadeIDs() {
	for _, gameEvent := range rh.Events {
		if gameEvent.Type == events.EventSmokeStart || gameEvent.Type == events.EventSmokeEnd {
			smokeEvent := gameEvent.Data.(events.SmokeEvent)
			nadeId := smokeEvent.NadeId
			if _, exists := rh.NadeMetadata[nadeId]; !exists {
				fmt.Println("Nade ID from smoke event not found in NadeMetadata:", nadeId)
			}
		}
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

func (rh *ReplayHandler) PrintEventLengths() {
	println(len(rh.Events))
}
