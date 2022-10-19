package quotaimpl

import (
	"github.com/grafana/grafana/pkg/services/quota"
)

func (s *Service) loadQuotaSettings() {
	// set global defaults.
	quotaSection := s.Cfg.Raw.Section("quota")

	var alertOrgQuota int64
	var alertGlobalQuota int64
	if s.Cfg.UnifiedAlerting.IsEnabled() {
		alertOrgQuota = quotaSection.Key("org_alert_rule").MustInt64(100)
		alertGlobalQuota = quotaSection.Key("global_alert_rule").MustInt64(-1)
	}

	s.defaultLimits = map[quota.TargetSrv]map[quota.Scope]int64{
		quota.UserTarget: {
			quota.GlobalScope: quotaSection.Key("global_user").MustInt64(-1),
		},
		quota.OrgTarget: {
			quota.GlobalScope: quotaSection.Key("global_org").MustInt64(-1),
			quota.OrgScope:    quotaSection.Key("org_user").MustInt64(10),
			quota.UserScope:   quotaSection.Key("user_org").MustInt64(10),
		},
		quota.DashboardTarget: {
			quota.GlobalScope: quotaSection.Key("global_dashboard").MustInt64(-1),
			quota.OrgScope:    quotaSection.Key("org_dashboard").MustInt64(-1),
		},
		quota.DataSourceTarget: {
			quota.GlobalScope: quotaSection.Key("global_data_source").MustInt64(-1),
			quota.OrgScope:    quotaSection.Key("org_data_source").MustInt64(-1),
		},
		quota.ApiKeyTarget: {
			quota.GlobalScope: quotaSection.Key("global_api_key").MustInt64(-1),
			quota.OrgScope:    quotaSection.Key("org_api_key").MustInt64(-1),
		},
		quota.SessionTarget: {
			quota.GlobalScope: quotaSection.Key("global_session").MustInt64(-1),
		},
		quota.AlertRuleTarget: {
			quota.GlobalScope: alertGlobalQuota,
			quota.OrgScope:    alertOrgQuota,
		},
		quota.FileTarget: {
			quota.GlobalScope: quotaSection.Key("global_file").MustInt64(-1),
		},
	}
}
