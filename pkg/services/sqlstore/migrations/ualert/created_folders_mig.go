package ualert

import (
	"encoding/json"
	"fmt"
	"time"

	"xorm.io/xorm"

	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

// createdFoldersKey is a vendored migration.createdFoldersKey.
var createdFoldersKey = "createdFolders"

// CreatedFoldersMigration moves the record of created folders during legacy migration from Dashboard created_by=-8
// to the kvstore. If there are no dashboards with created_by=-.8, then nothing needs to be done.
func CreatedFoldersMigration(mg *migrator.Migrator) {
	mg.AddMigration("migrate record of created folders during legacy migration to kvstore", &createdFoldersToKVStore{})
}

type createdFoldersToKVStore struct {
	migrator.MigrationBase
}

func (c createdFoldersToKVStore) SQL(migrator.Dialect) string {
	return codeMigration
}

func (c createdFoldersToKVStore) Exec(sess *xorm.Session, mg *migrator.Migrator) error {
	var results []struct {
		UID   string `xorm:"uid"`
		OrgID int64  `xorm:"org_id"`
	}
	folderCreatedBy := -8
	if err := sess.SQL("select * from dashboard where created_by = ?", folderCreatedBy).Find(&results); err != nil {
		return err
	}

	if len(results) == 0 {
		mg.Logger.Debug("no dashboards with created_by=-8, nothing to set in kvstore")
		return nil
	}

	orgFolderUids := make(map[int64][]string)
	for _, r := range results {
		orgFolderUids[r.OrgID] = append(orgFolderUids[r.OrgID], r.UID)
	}

	raw, err := json.Marshal(orgFolderUids)
	if err != nil {
		return err
	}

	var anyOrg int64 = 0
	now := time.Now()
	entry := kvStoreV1Entry{
		OrgID:     &anyOrg,
		Namespace: &KVNamespace,
		Key:       &createdFoldersKey,
		Value:     string(raw),
		Created:   now,
		Updated:   now,
	}
	if _, errCreate := sess.Table("kv_store").Insert(&entry); errCreate != nil {
		mg.Logger.Error("failed to insert record of created folders to kvstore", "err", errCreate)
		return fmt.Errorf("failed to insert record of created folders to kvstore: %w", errCreate)
	}
	return nil
}
