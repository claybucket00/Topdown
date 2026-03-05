package playerposition

import (
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

type NadePath struct {
	weapon   common.Equipment
	path     []common.TrajectoryEntry
	thrownBy int // UserID of the player who threw the nade
}

type PlayerPosition struct {
	Tick int
	X    float64
	Y    float64
}
