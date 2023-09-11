package migration

import (
	"context"
	"fmt"
	"strings"

	pb "github.com/prometheus/alertmanager/silence/silencepb"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/folder"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/services/secrets"
	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
)

// DASHBOARD_FOLDER is the format used to generate the folder name for migrated dashboards with custom permissions.
const DASHBOARD_FOLDER = "%s Alerts - %s"

// MaxFolderName is the maximum length of the folder name generated using DASHBOARD_FOLDER format
const MaxFolderName = 255

// It is defined in pkg/expr/service.go as "DatasourceType"
const expressionDatasourceUID = "__expr__"

type migration struct {
	log     log.Logger
	dialect migrator.Dialect
	cfg     *setting.Cfg

	seenUIDs deduplicator

	info              InfoStore
	store             db.DB
	ruleStore         RuleStore
	alertingStore     AlertingStore
	encryptionService secrets.Service
	dashboardService  dashboards.DashboardService
	folderService     folder.Service
	dsCacheService    datasources.CacheService

	folderPermissions    accesscontrol.FolderPermissionsService
	dashboardPermissions accesscontrol.DashboardPermissionsService
}

func newMigration(
	log log.Logger,
	cfg *setting.Cfg,
	info InfoStore,
	store db.DB,
	ruleStore RuleStore,
	alertingStore AlertingStore,
	encryptionService secrets.Service,
	dashboardService dashboards.DashboardService,
	folderService folder.Service,
	dsCacheService datasources.CacheService,
	folderPermissions accesscontrol.FolderPermissionsService,
	dashboardPermissions accesscontrol.DashboardPermissionsService,
) *migration {
	return &migration{
		// We deduplicate for case-insensitive matching in MySQL-compatible backend flavours because they use case-insensitive collation.
		seenUIDs:             deduplicator{set: make(map[string]struct{}), caseInsensitive: store.GetDialect().SupportEngine()},
		log:                  log,
		dialect:              store.GetDialect(),
		cfg:                  cfg,
		info:                 info,
		store:                store,
		ruleStore:            ruleStore,
		alertingStore:        alertingStore,
		encryptionService:    encryptionService,
		dashboardService:     dashboardService,
		folderService:        folderService,
		dsCacheService:       dsCacheService,
		folderPermissions:    folderPermissions,
		dashboardPermissions: dashboardPermissions,
	}
}

// orgMigration is a helper struct for migrating alerts for a single org. It contains state, services, and caches.
type orgMigration struct {
	orgID int64
	log   log.Logger

	dialect        migrator.Dialect
	dataPath       string
	dsCacheService datasources.CacheService

	folderHelper folderHelper

	silences            []*pb.MeshSilence
	alertRuleTitleDedup map[string]deduplicator // Folder -> deduplicator (Title).
}

// newOrgMigration creates a new orgMigration for the given orgID.
func newOrgMigration(m *migration, orgID int64) *orgMigration {
	return &orgMigration{
		orgID: orgID,
		log:   m.log.New("orgID", orgID),

		dialect:        m.dialect,
		dataPath:       m.cfg.DataPath,
		dsCacheService: m.dsCacheService,

		folderHelper: folderHelper{
			info:                     m.info,
			dialect:                  m.dialect,
			folderService:            m.folderService,
			folderPermissions:        m.folderPermissions,
			dashboardPermissions:     m.dashboardPermissions,
			permissionsMap:           make(map[int64]map[permissionHash]*folder.Folder),
			folderCache:              make(map[int64]*folder.Folder),
			newFolderCache:           make(map[int64]*folder.Folder),
			dashboardPermissionCache: make(map[string][]accesscontrol.ResourcePermission),
			folderPermissionCache:    make(map[string][]accesscontrol.ResourcePermission),
		},

		silences:            make([]*pb.MeshSilence, 0),
		alertRuleTitleDedup: make(map[string]deduplicator),
	}
}

// Exec executes the migration.
func (m *migration) Exec(ctx context.Context) error {
	dashAlerts, err := m.slurpDashAlerts(ctx)
	if err != nil {
		return err
	}
	m.log.Info("Alerts found to migrate", "alerts", len(dashAlerts))

	// Per org map of newly created rules to which notification channels it should send to.
	rulesPerOrg := make(map[int64]map[*models.AlertRule][]uidOrID)

	migrationsCache := make(map[int64]*orgMigration)
	for _, da := range dashAlerts {
		om, ok := migrationsCache[da.OrgId]
		if !ok {
			om = newOrgMigration(m, da.OrgId)
			migrationsCache[da.OrgId] = om
		}
		dash, err := m.dashboardService.GetDashboard(ctx, &dashboards.GetDashboardQuery{ID: da.DashboardId, OrgID: da.OrgId})
		if err != nil {
			return fmt.Errorf("failed to get dashboard [ID: %d] for alert %s [ID: %d]: %w", da.DashboardId, da.Name, da.Id, err)
		}
		l := om.log.New("dashboardTitle", dash.Title, "dashboardUID", dash.UID, "ruleID", da.Id, "ruleName", da.Name)

		f, err := om.folderHelper.getOrCreateMigratedFolder(ctx, l, dash)
		if err != nil {
			return fmt.Errorf("failed to get or create folder for alert %s [ID: %d] on dashboard %s [ID: %d]: %w", da.Name, da.Id, dash.Title, dash.ID, err)
		}
		alertRule, channels, err := om.migrateAlert(ctx, l, da, dash, f)
		if err != nil {
			return fmt.Errorf("failed to migrate alert %s [ID: %d] on dashboard %s [ID: %d]: %w", da.Name, da.Id, dash.Title, dash.ID, err)
		}

		if _, ok := rulesPerOrg[alertRule.OrgID]; !ok {
			rulesPerOrg[alertRule.OrgID] = make(map[*models.AlertRule][]uidOrID)
		}
		if _, ok := rulesPerOrg[alertRule.OrgID][alertRule]; !ok {
			rulesPerOrg[alertRule.OrgID][alertRule] = channels
		}
	}

	orgFolderUids := make(map[int64][]string)
	for _, om := range migrationsCache {
		if len(om.silences) > 0 {
			if err := om.writeSilencesFile(); err != nil {
				m.log.Error("Alert migration error: failed to write silence file", "err", err)
			}
		}
		folderUids := make([]string, 0, len(om.folderHelper.newFolderCache))
		for _, f := range om.folderHelper.newFolderCache {
			folderUids = append(folderUids, f.UID)
		}
		orgFolderUids[om.orgID] = folderUids
	}
	err = m.info.setCreatedFolders(ctx, orgFolderUids)
	if err != nil {
		return err
	}

	amConfigPerOrg, err := m.setupAlertmanagerConfigs(ctx, rulesPerOrg)
	if err != nil {
		return err
	}

	err = m.insertRules(ctx, rulesPerOrg)
	if err != nil {
		return err
	}

	for orgID, amConfig := range amConfigPerOrg {
		m.log.Info("Writing alertmanager config", "orgID", orgID, "receivers", len(amConfig.AlertmanagerConfig.Receivers), "routes", len(amConfig.AlertmanagerConfig.Route.Routes))
		if err := m.writeAlertmanagerConfig(ctx, orgID, amConfig); err != nil {
			return err
		}
	}

	return nil
}

// deduplicator is a wrapper around map[string]struct{} and util.GenerateShortUID() which aims help maintain and generate
// unique strings (such as uids or titles). if caseInsensitive is true, all uniqueness is determined in a
// case-insensitive manner. if maxLen is greater than 0, all strings will be truncated to maxLen before being checked in
// contains and dedup will always return a string of length maxLen or less.
type deduplicator struct {
	set             map[string]struct{}
	caseInsensitive bool
	maxLen          int
}

// contains checks whether the given string has already been seen by this deduplicator.
func (s *deduplicator) contains(u string) bool {
	dedup := u
	if s.caseInsensitive {
		dedup = strings.ToLower(dedup)
	}
	if s.maxLen > 0 && len(dedup) > s.maxLen {
		dedup = dedup[:s.maxLen]
	}
	_, seen := s.set[dedup]
	return seen
}

// deduplicate returns a unique string based on the given string by appending a uuid to it. Will truncate the given string if
// the resulting string would be longer than maxLen.
func (s *deduplicator) deduplicate(dedup string) (string, error) {
	uid := util.GenerateShortUID()
	if s.maxLen > 0 && len(dedup)+1+len(uid) > s.maxLen {
		trunc := s.maxLen - 1 - len(uid)
		dedup = dedup[:trunc]
	}

	return dedup + "_" + uid, nil
}

// add adds the given string to the deduplicator.
func (s *deduplicator) add(uid string) {
	dedup := uid
	if s.caseInsensitive {
		dedup = strings.ToLower(dedup)
	}
	s.set[dedup] = struct{}{}
}
