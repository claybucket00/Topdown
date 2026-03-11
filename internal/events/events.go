package events

import ulid "github.com/oklog/ulid/v2"

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
	EventInfernoStart
	EventInfernoEnd
	EventKill
)

type SmokeEvent struct {
	X      float64
	Y      float64
	NadeId ulid.ULID
}

type KillEvent struct {
	VictimID int
}
