package migration

import (
	"testing"

	"github.com/grafana/grafana/pkg/infra/log/logtest"
	fake_secrets "github.com/grafana/grafana/pkg/services/secrets/fakes"
	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
	"github.com/grafana/grafana/pkg/setting"
)

// newTestMigration generates an empty migration to use in tests.
func newTestMigration(t *testing.T) *migration {
	t.Helper()

	return &migration{
		log: &logtest.Fake{},
		seenUIDs: deduplicator{
			set: make(map[string]struct{}),
		},
		dialect:           migrator.NewMysqlDialect(), // Could allow tests to determine this.
		cfg:               &setting.Cfg{},
		encryptionService: fake_secrets.NewFakeSecretsService(),
	}
}

// newTestOrgMigration generates an empty orgMigration to use in tests.
func newTestOrgMigration(t *testing.T, orgID int64) *orgMigration {
	t.Helper()
	return newOrgMigration(newTestMigration(t), orgID)
}
