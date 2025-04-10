package limiter

import "time"

type event struct {
	Start      time.Time     `json:"start-ts"`
	Duration   time.Duration `json:"duration"`
	StatusCode int           `json:"status-code"`
}

//Path       string        `json:"path"` // uri path
//	Start      time.Time     `json:"start-ts"`
