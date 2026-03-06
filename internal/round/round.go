package round

type Round struct {
	StartTick int
	EndTick   int
	// FrameDatum []FrameData // Index: Tick, Val: FrameData for that tick
	// PlayerPositions map[int][]playerposition.PlayerPosition // Key: UserID, Val: List of positions during the round
	// NadePaths       map[int][]playerposition.NadePath       // Key: nadeID, Val: List of nade positions
}
