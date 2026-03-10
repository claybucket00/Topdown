package playerposition

import (
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

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
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
