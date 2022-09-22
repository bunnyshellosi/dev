package config

// +enum
type Mode string

const (
	TwoWaySafe     Mode = "two-way-safe"
	TwoWayResolved Mode = "two-way-resolved"
	OneWaySafe     Mode = "one-way-safe"
	OneWayReplica  Mode = "one-way-replica"
)
