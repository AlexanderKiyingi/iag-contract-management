package models

import "time"

// SupplyStatus is the lifecycle of a coffee off-take contract. Kept aligned with
// the TraceAG portal's Contract.status values.
type SupplyStatus string

const (
	SupplyDraft     SupplyStatus = "Draft"
	SupplySigned    SupplyStatus = "Signed"
	SupplyActive    SupplyStatus = "Active"
	SupplyCompleted SupplyStatus = "Completed"
)

func (s SupplyStatus) Valid() bool {
	switch s {
	case SupplyDraft, SupplySigned, SupplyActive, SupplyCompleted:
		return true
	}
	return false
}

// SupplyContract is a coffee off-take agreement with a farmer. JSON tags match
// the TraceAG portal's Contract model so the portal can consume it directly.
type SupplyContract struct {
	ID                  string       `json:"id"`
	FarmerBusinessID    string       `json:"farmerId"`
	FarmerName          string       `json:"farmerName"`
	Variety             string       `json:"variety"`
	CommittedWeightKg   float64      `json:"committedWeightKg"`
	BasePricePerKgUgx   int64        `json:"basePricePerKgUgx"`
	QualityBonusRateUgx int64        `json:"qualityBonusRateUgx"`
	SignDate            *time.Time   `json:"signDate,omitempty"`
	Status              SupplyStatus `json:"status"`
	ContractText        string       `json:"contractText"`
	Signature           string       `json:"signature,omitempty"`
	CreatedAt           time.Time    `json:"createdAt"`
	UpdatedAt           time.Time    `json:"updatedAt"`
}

// SupplyContractInput is the create/update payload.
type SupplyContractInput struct {
	FarmerBusinessID    string       `json:"farmerId"`
	FarmerName          string       `json:"farmerName"`
	Variety             string       `json:"variety"`
	CommittedWeightKg   float64      `json:"committedWeightKg"`
	BasePricePerKgUgx   int64        `json:"basePricePerKgUgx"`
	QualityBonusRateUgx int64        `json:"qualityBonusRateUgx"`
	ContractText        string       `json:"contractText"`
	Status              SupplyStatus `json:"status"`
}
