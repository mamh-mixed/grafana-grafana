package quotaimpl

import (
	"context"
	"testing"

	"github.com/grafana/grafana/pkg/services/quota"
	"github.com/stretchr/testify/require"
)

func TestQuotaService(t *testing.T) {
	quotaStore := &FakeQuotaStore{}
	quotaService := Service{
		store: quotaStore,
	}

	t.Run("delete quota", func(t *testing.T) {
		err := quotaService.DeleteByUser(context.Background(), 1)
		require.NoError(t, err)
	})
}

type FakeQuotaStore struct {
	ExpectedError error
}

func (f *FakeQuotaStore) DeleteByUser(ctx context.Context, userID int64) error {
	return f.ExpectedError
}

func (f *FakeQuotaStore) Get(ctx context.Context, scopeParams *quota.ScopeParameters) (quota.Limits, error) {
	return nil, f.ExpectedError
}

func (f *FakeQuotaStore) Update(ctx context.Context, cmd *quota.UpdateQuotaCmd) error {
	return f.ExpectedError
}
