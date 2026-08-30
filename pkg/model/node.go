package model

import "time"

// Node represents a tested proxy node with latency and metadata matching the user specification.
type Node struct {
	Config      string    `json:"config" db:"config"`
	Ping        string    `json:"ping" db:"ping"`       // e.g. "62"
	Country     string    `json:"country" db:"country"` // 2-letter ISO country code e.g. "IR", "DE", "US"
	IP          string    `json:"ip" db:"ip"`           // Resolved outbound/server IP
	Protocol    string    `json:"protocol" db:"protocol"`
	Hash        string    `json:"hash" db:"hash"`
	Source      string    `json:"source" db:"source"`
	IsAlive     bool      `json:"is_alive" db:"is_alive"`
	FailCount   int       `json:"fail_count" db:"fail_count"`
	FirstSeenAt time.Time `json:"first_seen_at" db:"first_seen_at"`
	LastCheckAt time.Time `json:"last_check_at" db:"last_check_at"`
}

// UserNodeResponse matches the exact requested JSON API format
type UserNodeResponse struct {
	Config  string `json:"config"`
	Ping    string `json:"ping"`
	Country string `json:"country"`
	IP      string `json:"ip"`
}
