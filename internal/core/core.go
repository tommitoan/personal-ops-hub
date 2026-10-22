package core

import "time"

// Card is a provider or health observation with explicit freshness and provenance.
type Card struct {
	Key       string    `json:"key"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	Unit      string    `json:"unit,omitempty"`
	Source    string    `json:"source"`
	Link      string    `json:"link,omitempty"`
	Window    string    `json:"window,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IncidentState tracks failure and recovery transitions.
type IncidentState struct {
	Key                 string
	ConsecutiveFailures int
	Open                bool
	ChangedAt           time.Time
}

// Transition applies one probe result and returns a notification event when state changes.
func Transition(s IncidentState, healthy bool, threshold int, now time.Time) (IncidentState, string) {
	if threshold < 1 {
		threshold = 1
	}
	if healthy {
		s.ConsecutiveFailures = 0
		if s.Open {
			s.Open = false
			s.ChangedAt = now
			return s, "resolved"
		}
		return s, ""
	}
	s.ConsecutiveFailures++
	if !s.Open && s.ConsecutiveFailures >= threshold {
		s.Open = true
		s.ChangedAt = now
		return s, "down"
	}
	return s, ""
}
