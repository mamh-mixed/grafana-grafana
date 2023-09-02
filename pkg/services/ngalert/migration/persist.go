package migration

import (
	"context"
	"encoding/json"
	"fmt"

	apimodels "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
)

// RuleStore represents the ability to persist and query alert rules.
type RuleStore interface {
	InsertAlertRules(ctx context.Context, rule []models.AlertRule) (map[string]int64, error)
}

// AlertingStore is the database interface used by the Alertmanager service.
type AlertingStore interface {
	SaveAlertmanagerConfiguration(ctx context.Context, cmd *models.SaveAlertmanagerConfigurationCmd) error
}

// insertRules inserts the given rules into the database.
func (m *migration) insertRules(ctx context.Context, rulesPerOrg map[int64]map[*models.AlertRule][]uidOrID) error {
	for orgID, orgRules := range rulesPerOrg {
		m.log.Info("Inserting migrated alert rules", "orgID", orgID, "count", len(orgRules))
		rules := make([]models.AlertRule, 0, len(orgRules))
		for rule := range orgRules {
			rules = append(rules, *rule)
		}
		_, err := m.ruleStore.InsertAlertRules(ctx, rules)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeAlertmanagerConfig writes the given Alertmanager configuration to the database.
func (m *migration) writeAlertmanagerConfig(ctx context.Context, orgID int64, amConfig *apimodels.PostableUserConfig) error {
	rawAmConfig, err := json.Marshal(amConfig)
	if err != nil {
		return err
	}

	cmd := models.SaveAlertmanagerConfigurationCmd{
		AlertmanagerConfiguration: string(rawAmConfig),
		ConfigurationVersion:      fmt.Sprintf("v%d", models.AlertConfigurationVersion),
		Default:                   false,
		OrgID:                     orgID,
		LastApplied:               0,
	}
	return m.alertingStore.SaveAlertmanagerConfiguration(ctx, &cmd)
}
