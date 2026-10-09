package models

import "time"

type Deal struct {
	ID              string    `json:"deal_id"`
	AccountID       string    `json:"account_id"`
	OwnerID         string    `json:"owner_id"`
	Stage           string    `json:"stage"`
	StageSince      time.Time `json:"stage_since"`
	CreatedAt       time.Time `json:"created_at"`
	OutletCount     int       `json:"outlet_count"`
	PotentialACVIDR int64     `json:"potential_acv_idr"`
	Status          string    `json:"status"`
}
