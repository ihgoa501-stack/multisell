package profit

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getCNYRate_NilRateSvc_UsesDefault(t *testing.T) {
	svc := &Service{defaultCNYRate: 7.0}
	assert.Equal(t, 7.0, svc.getCNYRate())
}

func Test_getCNYRate_InvalidRate_UsesDefault(t *testing.T) {
	svc := &Service{defaultCNYRate: 7.3}
	assert.Equal(t, 7.3, svc.getCNYRate())
}

func Test_getCNYRate_ConfiguredDefault(t *testing.T) {
	svc := NewService(nil, nil, nil, 7.5)
	assert.Equal(t, 7.5, svc.getCNYRate())
}

func TestNewService_ZeroDefault_GracefulFallback(t *testing.T) {
	svc := NewService(nil, nil, nil, 0)
	assert.Equal(t, 7.2, svc.getCNYRate()) // original default preserved
}

func TestNewService_NegativeDefault_GracefulFallback(t *testing.T) {
	svc := NewService(nil, nil, nil, -1)
	assert.Equal(t, 7.2, svc.getCNYRate())
}

func TestCalculate_ProductNotFound(t *testing.T) {
	db := dbtest.NewDB(t)
	log := dbtest.NewLogger(t)
	svc := NewService(db, log, nil, 7.2)

	_, err := svc.Calculate(99999, "test")
	assert.Error(t, err)
}

func TestCalculate_WithValidProduct(t *testing.T) {
	db := dbtest.NewDB(t, &candidate.CandidateProduct{}, &ProfitSummary{})
	log := dbtest.NewLogger(t)
	svc := NewService(db, log, nil, 7.2)

	platformID := int64(1)
	prod := candidate.CandidateProduct{
		Title:              "test product",
		PurchasePrice:      50,
		PurchaseCurrency:   "CNY",
		TargetSalePrice:    100,
		TargetPlatformID:   &platformID,
		DestinationCountry: "US",
		PackageWeightKg:    0.5,
	}
	require.NoError(t, db.Create(&prod).Error)

	result, err := svc.Calculate(prod.ID, "test")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Status)
	assert.Greater(t, result.TotalCost, 0.0)
}
