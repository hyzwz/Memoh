package schedule

import "time"

type Schedule struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Pattern      string    `json:"pattern"`
	MaxCalls     *int      `json:"max_calls,omitempty"`
	CurrentCalls int       `json:"current_calls"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Enabled      bool      `json:"enabled"`
	Command      string    `json:"command"`
	UserID       string    `json:"user_id"`
}

type CreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
	MaxCalls    *int   `json:"max_calls,omitempty"`
	Command     string `json:"command"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Pattern     *string `json:"pattern,omitempty"`
	MaxCalls    *int    `json:"max_calls,omitempty"`
	Command     *string `json:"command,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

type ListResponse struct {
	Items []Schedule `json:"items"`
}

