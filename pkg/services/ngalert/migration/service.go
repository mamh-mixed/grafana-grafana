package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/kvstore"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/serverlock"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/folder"
	"github.com/grafana/grafana/pkg/services/ngalert/notifier"
	"github.com/grafana/grafana/pkg/services/ngalert/store"
	"github.com/grafana/grafana/pkg/services/secrets"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/setting"
)

// actionName is the unique row-level lock name for serverlock.ServerLockService.
const actionName = "alerting migration"

//nolint:stylecheck
var ForceMigrationError = fmt.Errorf("Grafana has already been migrated to Unified Alerting. Any alert rules created while using Unified Alerting will be deleted by rolling back. Set force_migration=true in your grafana.ini and restart Grafana to roll back and delete Unified Alerting configuration data.")

type MigrationService struct {
	lock                 *serverlock.ServerLockService
	store                db.DB
	cfg                  *setting.Cfg
	log                  log.Logger
	info                 InfoStore
	ruleStore            RuleStore
	alertingStore        AlertingStore
	encryptionService    secrets.Service
	dashboardService     dashboards.DashboardService
	folderService        folder.Service
	dataSourceCache      datasources.CacheService
	folderPermissions    accesscontrol.FolderPermissionsService
	dashboardPermissions accesscontrol.DashboardPermissionsService
}

func ProvideService(
	lock *serverlock.ServerLockService,
	cfg *setting.Cfg,
	sqlStore db.DB,
	kv kvstore.KVStore,
	ruleStore *store.DBstore,
	encryptionService secrets.Service,
	dashboardService dashboards.DashboardService,
	folderService folder.Service,
	dataSourceCache datasources.CacheService,
	folderPermissions accesscontrol.FolderPermissionsService,
	dashboardPermissions accesscontrol.DashboardPermissionsService,
) (*MigrationService, error) {
	return &MigrationService{
		lock:                 lock,
		log:                  log.New("ngalert.migration"),
		cfg:                  cfg,
		store:                sqlStore,
		info:                 InfoStore{kv: kvstore.WithNamespace(kv, 0, KVNamespace)},
		ruleStore:            ruleStore,
		alertingStore:        ruleStore,
		encryptionService:    encryptionService,
		dashboardService:     dashboardService,
		folderService:        folderService,
		dataSourceCache:      dataSourceCache,
		folderPermissions:    folderPermissions,
		dashboardPermissions: dashboardPermissions,
	}, nil
}

// Run starts the migration. This will either migrate from legacy alerting to unified alerting or revert the migration.
// If the migration status in the kvstore is not set and unified alerting is enabled, the migration will be executed.
// If the migration status in the kvstore is set and both unified alerting is disabled and ForceMigration is set to true, the migration will be reverted.
func (ms *MigrationService) Run(ctx context.Context) error {
	var errMigration error
	errLock := ms.lock.LockExecuteAndRelease(ctx, actionName, time.Minute*10, func(context.Context) {
		ms.log.Info("Starting")
		errMigration = ms.store.InTransaction(ctx, func(ctx context.Context) error {
			migrated, err := ms.info.IsMigrated(ctx)
			if err != nil {
				return fmt.Errorf("getting migration status: %w", err)
			}
			if migrated == ms.cfg.UnifiedAlerting.IsEnabled() {
				// Nothing to do.
				ms.log.Info("No migrations to run")
				return nil
			}

			if migrated {
				// If legacy alerting is also disabled, there is nothing to do
				if setting.AlertingEnabled != nil && !*setting.AlertingEnabled {
					return nil
				}

				// Safeguard to prevent data loss when reverting from UA to LA.
				if !ms.cfg.ForceMigration {
					return ForceMigrationError
				}

				// Revert migration
				ms.log.Info("Reverting legacy migration")
				err := ms.Revert(ctx)
				if err != nil {
					return fmt.Errorf("reverting migration: %w", err)
				}
				ms.log.Info("Legacy migration reverted")
				return nil
			}

			ms.log.Info("Starting legacy migration")
			mg := newMigration(
				ms.log,
				ms.cfg,
				ms.info,
				ms.store,
				ms.ruleStore,
				ms.alertingStore,
				ms.encryptionService,
				ms.dashboardService,
				ms.folderService,
				ms.dataSourceCache,
				ms.folderPermissions,
				ms.dashboardPermissions,
			)

			err = mg.Exec(ctx)
			if err != nil {
				return fmt.Errorf("executing migration: %w", err)
			}

			err = ms.info.setMigrated(ctx, true)
			if err != nil {
				return fmt.Errorf("setting migration status: %w", err)
			}

			ms.log.Info("Completed legacy migration")
			return nil
		})
	})
	if errLock != nil {
		ms.log.Warn("Server lock for alerting migration already exists")
		return nil
	}
	if errMigration != nil {
		return fmt.Errorf("migration failed: %w", errMigration)
	}
	return nil
}

// IsDisabled returns true if the cfg is nil.
func (ms *MigrationService) IsDisabled() bool {
	return ms.cfg == nil
}

// Revert reverts the migration, deleting all unified alerting resources such as alert rules, alertmanager configurations, and silence files.
// In addition, it will delete all folders and permissions originally created by this migration, these are stored in the kvstore.
func (ms *MigrationService) Revert(ctx context.Context) error {
	return ms.store.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		_, err := sess.Exec("delete from alert_rule")
		if err != nil {
			return err
		}

		_, err = sess.Exec("delete from alert_rule_version")
		if err != nil {
			return err
		}

		createdFolders, err := ms.info.GetCreatedFolders(ctx)
		if err != nil {
			return err
		}
		for orgId, folderUIDs := range createdFolders {
			usr := getRevertUser(orgId)

			for _, folderUID := range folderUIDs {
				cmd := folder.DeleteFolderCommand{
					UID:          folderUID,
					OrgID:        orgId,
					SignedInUser: usr.(*user.SignedInUser),
				}
				err := ms.folderService.Delete(ctx, &cmd) // Also handles permissions and other related entities.
				if err != nil {
					return err
				}
			}
		}

		_, err = sess.Exec("delete from alert_configuration")
		if err != nil {
			return err
		}

		_, err = sess.Exec("delete from ngalert_configuration")
		if err != nil {
			return err
		}

		_, err = sess.Exec("delete from alert_instance")
		if err != nil {
			return err
		}

		exists, err := sess.IsTableExist("kv_store")
		if err != nil {
			return err
		}

		if exists {
			_, err = sess.Exec("delete from kv_store where namespace = ?", notifier.KVNamespace)
			if err != nil {
				return err
			}

			_, err = sess.Exec("delete from kv_store where namespace = ?", KVNamespace)
			if err != nil {
				return err
			}
		}

		files, err := filepath.Glob(filepath.Join(ms.cfg.DataPath, "alerting", "*", "silences"))
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := os.Remove(f); err != nil {
				ms.log.Error("alert migration error: failed to remove silence file", "file", f, "err", err)
			}
		}

		err = ms.info.setMigrated(ctx, false)
		if err != nil {
			return fmt.Errorf("setting migration status: %w", err)
		}

		return nil
	})
}
