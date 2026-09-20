package controllers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// decodeJSON refuses unknown keys, so a client body that carries one field the
// target struct lacks is a 400 for the whole request. These are the bodies the
// Contract Manager app actually sends (its adapter maps the record it holds);
// each must decode, or that screen's save is dead without any test noticing.
func TestClientBodiesDecode(t *testing.T) {
	cases := []struct {
		name string
		body string
		dst  any
	}{
		{"contract patch with number", `{"number":"CT-1","name":"X","status":"Draft","value":1,"currency":"UGX"}`, &models.GovContractPatch{}},
		{"milestone create with contractId", `{"contractId":"GCT-1","name":"M","value":1,"targetDate":"","status":"Pending"}`, &models.GovMilestoneInput{}},
		{"milestone patch with contractId", `{"contractId":"GCT-1","name":"M","value":1,"status":"Pending"}`, &models.GovMilestonePatch{}},
		{"obligation create with contractId", `{"contractId":"GCT-1","type":"Insurance","owner":"","dueDate":"","frequency":"","evidence":"","status":"Open","escalation":""}`, &models.ObligationInput{}},
		{"obligation patch with contractId", `{"contractId":"GCT-1","type":"Insurance","status":"Met"}`, &models.ObligationPatch{}},
		{"valuation input", `{"contractorName":"A","contractorId":"","period":"","contractSum":0,"amountPaid":0,"verifiedValueOwed":1,"consultantRecommendation":0,"ceoApproval":0,"remarks":"","verifiedDate":"","reference":"R","dueDate":"","currency":"UGX","division":"","tax":0,"status":"Draft","attachments":""}`, &models.ValuationInput{}},
		{"payment patch", `{"amount":1,"retention":0}`, &models.GovPaymentPatch{}},
		{"requisition patch", `{"no":"R1","title":"T","estimate":1,"justification":"why"}`, &models.GovRequisitionPatch{}},
	}
	for _, tc := range cases {
		r := httptest.NewRequest("POST", "/x", strings.NewReader(tc.body))
		if err := decodeJSON(r, tc.dst); err != nil {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}
