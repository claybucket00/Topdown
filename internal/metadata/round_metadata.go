package metadata

import (
	round "topdown/internal/round"

	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

type RoundMetadata struct {
	Score         round.Score         `json:"score"`
	PlayerToTeams map[int]common.Team `json:"player_to_teams"` // Map of team each player started the round on indexed by playerId
}
