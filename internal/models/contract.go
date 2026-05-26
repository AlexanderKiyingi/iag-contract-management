package models

import (
	"fmt"
	"strings"
	"time"
)

func (s *Store) CreateContract(in ContractInput) (Contract, error) {
	no := strings.TrimSpace(in.No)
	if no == "" {
		no = s.nextContractNo()
	}
	if strings.TrimSpace(in.Name) == "" {
		return Contract{}, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if strings.TrimSpace(in.Zone) == "" {
		return Contract{}, fmt.Errorf("%w: zone is required", ErrValidation)
	}

	cs := int64(45_000_000)
	if in.Cs != nil {
		cs = *in.Cs
	}
	paid := int64(0)
	if in.Paid != nil {
		paid = *in.Paid
	}
	workers := 10
	if in.Workers != nil {
		workers = *in.Workers
	}

	c := Contract{
		No:      no,
		Name:    in.Name,
		Zone:    in.Zone,
		Cs:      cs,
		Paid:    paid,
		Bal:     cs - paid,
		Prog:    clamp(in.Prog, 0, 100),
		Status:  defaultStatus(in.Status),
		Pri:     defaultPriority(in.Pri),
		Workers: workers,
		Sup:     in.Sup,
		Remarks: in.Remarks,
		Created: time.Now().Format("2006-01-02"),
	}
	if err := s.insertContract(c); err != nil {
		return Contract{}, err
	}
	return c, nil
}

func (s *Store) PatchContract(no string, patch ContractPatch) (Contract, error) {
	no = strings.TrimSpace(no)
	if no == "" {
		return Contract{}, ErrValidation
	}
	return s.updateContract(no, func(c *Contract) error {
		if patch.Name != nil {
			c.Name = *patch.Name
		}
		if patch.Zone != nil {
			c.Zone = *patch.Zone
		}
		if patch.Status != nil {
			c.Status = *patch.Status
		}
		if patch.Pri != nil {
			c.Pri = *patch.Pri
		}
		if patch.Prog != nil {
			c.Prog = clamp(*patch.Prog, 0, 100)
		}
		if patch.Sup != nil {
			c.Sup = *patch.Sup
		}
		if patch.Remarks != nil {
			c.Remarks = *patch.Remarks
		}
		if patch.Workers != nil {
			c.Workers = *patch.Workers
		}
		if patch.Cs != nil {
			c.Cs = *patch.Cs
		}
		if patch.Paid != nil {
			c.Paid = *patch.Paid
		}
		c.Bal = c.Cs - c.Paid
		return nil
	})
}

func (s *Store) DeleteContract(no string) error {
	no = strings.TrimSpace(no)
	if no == "" {
		return ErrValidation
	}
	return s.deleteContract(no)
}

func (s *Store) FindContract(no string) (Contract, error) {
	no = strings.TrimSpace(no)
	if no == "" {
		return Contract{}, ErrValidation
	}
	return s.GetContract(no)
}

func defaultStatus(status ContractStatus) ContractStatus {
	if status == "" {
		return StatusPlanning
	}
	return status
}

func defaultPriority(p Priority) Priority {
	if p == "" {
		return PriorityMedium
	}
	return p
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
