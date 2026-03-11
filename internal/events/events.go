package events

import (
	r2 "github.com/golang/geo/r2"
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
	EventInferno
	EventInfernoStart
	EventInfernoEnd
)

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
