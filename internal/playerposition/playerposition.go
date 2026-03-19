package playerposition

import (
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

type PlayerID int // TODO: Refactor code to use PlayerID type instead of raw int

type NadePath struct {
	weapon   common.Equipment
	path     []common.TrajectoryEntry
	thrownBy int // UserID of the player who threw the nade
}

type NadePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PlayerPosition struct {
	X   float64 `json:"x"`
	Y   float64 `json:"y"`
	Yaw float32 `json:"yaw"`
}
