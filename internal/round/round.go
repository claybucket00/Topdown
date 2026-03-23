package round

import common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"

type Score struct {
	CT int `json:"ct"`
	T  int `json:"t"`
}

type PlayerEquipment struct {
	Equipment []string `json:"equipment"`
	Money     int      `json:"money"`
}

type Round struct {
	StartTick         int
	EndTick           int
	Score             Score
	PlayerTeams       map[int]common.Team     // Key: playerId, Val: team the player started the round on
	PlayerToEquipment map[int]PlayerEquipment // Key: playerId, Val: list of player's equipment (see PlayerEquipment) at the start of round.
}
