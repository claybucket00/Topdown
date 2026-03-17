package events

import (
	r2 "github.com/golang/geo/r2"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	ulid "github.com/oklog/ulid/v2"
)

type GameEvent struct {
	Tick int
	Type EventType
	Data any
}

type EventType int

const (
	_ EventType = iota
	EventFlash
	EventSmokeStart
	EventSmokeEnd
	EventKill
	EventHe
	EventTeamChange
	EventInferno
	EventInfernoStart
	EventInfernoEnd
)

type TeamChangeEvent struct {
	PlayerID int
	Team     common.Team
}

type SmokeEvent struct {
	X      float64
	Y      float64
	NadeId ulid.ULID
}

type GrenadeEvent struct {
	X      float64
	Y      float64
	NadeId ulid.ULID
}

type KillEvent struct {
	VictimID int
}

type InfernoEvent struct {
	Points []r2.Point // 2D convex hull of all the fires active in the inferno.
	NadeId int64      // We cannot use the usual ULID id as infernos are seperate entities than incendiary grenades.
}
