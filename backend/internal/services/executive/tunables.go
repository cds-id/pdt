package executive

import "time"

var (
	MaxAnchors            = 200
	PerAnchorWorkers      = 8
	CommitDistanceMax     = 0.30
	WADistanceMax         = 0.32
	SemanticCommitLimit   = 10
	SemanticWALimit       = 15
	WAGroupWindow         = 10 * time.Minute
	OrphanWAGroupWindow   = 30 * time.Minute
	OrphanWANoiseFloor    = 3
	StaleThresholdDefault = 7
	MaxRangeDays          = 90
	DefaultRangeDays      = 14
)
