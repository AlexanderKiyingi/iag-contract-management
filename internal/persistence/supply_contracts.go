package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// ErrSupplyNotFound is returned when a supply contract id does not exist.
var ErrSupplyNotFound = errors.New("supply contract not found")

// SupplyStore persists coffee off-take contracts.
type SupplyStore struct {
	pool *pgxpool.Pool
}

func NewSupplyStore(pool *pgxpool.Pool) *SupplyStore { return &SupplyStore{pool: pool} }

// supplyCols is shared by list/get SELECTs and create RETURNING so column order
// stays in lockstep with scanSupplyContract.
const supplyCols = `id, farmer_business_id, farmer_name, variety, committed_weight_kg,
	base_price_per_kg_ugx, quality_bonus_rate_ugx, sign_date, status, contract_text,
	signature, created_at, updated_at`

func scanSupplyContract(row interface{ Scan(dest ...any) error }) (*models.SupplyContract, error) {
	var c models.SupplyContract
	var status string
	if err := row.Scan(&c.ID, &c.FarmerBusinessID, &c.FarmerName, &c.Variety, &c.CommittedWeightKg,
		&c.BasePricePerKgUgx, &c.QualityBonusRateUgx, &c.SignDate, &status, &c.ContractText,
		&c.Signature, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	c.Status = models.SupplyStatus(status)
	return &c, nil
}

func (s *SupplyStore) ListContracts(ctx context.Context) ([]models.SupplyContract, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+supplyCols+` FROM supply_contracts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.SupplyContract{}
	for rows.Next() {
		c, err := scanSupplyContract(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (s *SupplyStore) GetContract(ctx context.Context, id string) (*models.SupplyContract, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+supplyCols+` FROM supply_contracts WHERE id = $1`, id)
	c, err := scanSupplyContract(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSupplyNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *SupplyStore) CreateContract(ctx context.Context, c models.SupplyContract) (*models.SupplyContract, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO supply_contracts
			(id, farmer_business_id, farmer_name, variety, committed_weight_kg,
			 base_price_per_kg_ugx, quality_bonus_rate_ugx, status, contract_text)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING `+supplyCols,
		c.ID, c.FarmerBusinessID, c.FarmerName, c.Variety, c.CommittedWeightKg,
		c.BasePricePerKgUgx, c.QualityBonusRateUgx, string(c.Status), c.ContractText)
	return scanSupplyContract(row)
}

// Sign stamps the signature + sign date and advances a Draft contract to Signed
// (idempotent for already-signed contracts, which just refresh the signature).
func (s *SupplyStore) Sign(ctx context.Context, id, signature string) (*models.SupplyContract, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE supply_contracts
		SET signature = $2,
			sign_date = COALESCE(sign_date, now()),
			status = CASE WHEN status = 'Draft' THEN 'Signed' ELSE status END,
			updated_at = now()
		WHERE id = $1
		RETURNING `+supplyCols, id, signature)
	c, err := scanSupplyContract(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSupplyNotFound
		}
		return nil, err
	}
	return c, nil
}

// UpdateStatus transitions a contract to the given status.
func (s *SupplyStore) UpdateStatus(ctx context.Context, id string, status models.SupplyStatus) (*models.SupplyContract, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE supply_contracts SET status = $2, updated_at = $3 WHERE id = $1
		RETURNING `+supplyCols, id, string(status), time.Now().UTC())
	c, err := scanSupplyContract(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSupplyNotFound
		}
		return nil, err
	}
	return c, nil
}
