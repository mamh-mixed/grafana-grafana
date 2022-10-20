package quotatest

import (
	"context"

	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/quota"
)

type FakeQuotaService struct {
	reached bool
	err     error
}

func NewQuotaServiceFake(reached bool, err error) *FakeQuotaService {
	return &FakeQuotaService{reached, err}
}

func (f *FakeQuotaService) Get(ctx context.Context, scope string, id int64) ([]quota.QuotaDTO, error) {
	return []quota.QuotaDTO{}, nil
}

func (f *FakeQuotaService) Update(ctx context.Context, cmd *quota.UpdateQuotaCmd) error {
	return nil
}

func (f *FakeQuotaService) QuotaReached(c *models.ReqContext, target string) (bool, error) {
	return f.reached, f.err
}

func (f *FakeQuotaService) CheckQuotaReached(c context.Context, target string, params *quota.ScopeParameters) (bool, error) {
	return f.reached, f.err
}

func (f *FakeQuotaService) DeleteByUser(c context.Context, userID int64) error {
	return f.err
}
