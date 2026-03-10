package round

import common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"

type Score struct {
	CT int `json:"ct"`
	T  int `json:"t"`
}

type Round struct {
	StartTick   int
	EndTick     int
	Score       Score
	PlayerTeams map[int]common.Team // Key: playerId, Val: team the player started the round on
}
