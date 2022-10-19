package sqlstore

import (
	"context"
	"fmt"

	"github.com/grafana/grafana/pkg/models"
)

const (
	alertRuleTarget = "alert_rule"
	dashboardTarget = "dashboard"
	filesTarget     = "file"
)

type targetCount struct {
	Count int64
}

func (ss *SQLStore) GetOrgQuotaByTarget(ctx context.Context, query *models.GetOrgQuotaByTargetQuery) error {
	return ss.WithDbSession(ctx, func(sess *DBSession) error {
		quota := models.Quota{
			Target: query.Target,
			OrgId:  query.OrgId,
		}
		has, err := sess.Get(&quota)
		if err != nil {
			return err
		} else if !has {
			quota.Limit = query.Default
		}

		var used int64
		if query.Target != alertRuleTarget || query.UnifiedAlertingEnabled {
			// get quota used.
			rawSQL := fmt.Sprintf("SELECT COUNT(*) AS count FROM %s WHERE org_id=?",
				dialect.Quote(query.Target))

			if query.Target == dashboardTarget {
				rawSQL += fmt.Sprintf(" AND is_folder=%s", dialect.BooleanStr(false))
			}
			// need to account for removing service accounts from the user table
			if query.Target == "org_user" {
				rawSQL = fmt.Sprintf("SELECT COUNT(*) as count from (select user_id from %s where org_id=? AND user_id IN (SELECT id as user_id FROM %s WHERE is_service_account=%s)) as subq",
					dialect.Quote(query.Target),
					dialect.Quote("user"),
					dialect.BooleanStr(false),
				)
			}
			resp := make([]*targetCount, 0)
			if err := sess.SQL(rawSQL, query.OrgId).Find(&resp); err != nil {
				return err
			}
			used = resp[0].Count
		}

		query.Result = &models.OrgQuotaDTO{
			Target: query.Target,
			Limit:  quota.Limit,
			OrgId:  query.OrgId,
			Used:   used,
		}

		return nil
	})
}

func (ss *SQLStore) GetUserQuotaByTarget(ctx context.Context, query *models.GetUserQuotaByTargetQuery) error {
	return ss.WithDbSession(ctx, func(sess *DBSession) error {
		quota := models.Quota{
			Target: query.Target,
			UserId: query.UserId,
		}
		has, err := sess.Get(&quota)
		if err != nil {
			return err
		} else if !has {
			quota.Limit = query.Default
		}

		var used int64
		if query.Target != alertRuleTarget || query.UnifiedAlertingEnabled {
			// get quota used.
			rawSQL := fmt.Sprintf("SELECT COUNT(*) as count from %s where user_id=?", dialect.Quote(query.Target))
			resp := make([]*targetCount, 0)
			if err := sess.SQL(rawSQL, query.UserId).Find(&resp); err != nil {
				return err
			}
			used = resp[0].Count
		}

		query.Result = &models.UserQuotaDTO{
			Target: query.Target,
			Limit:  quota.Limit,
			UserId: query.UserId,
			Used:   used,
		}

		return nil
	})
}

func (ss *SQLStore) GetGlobalQuotaByTarget(ctx context.Context, query *models.GetGlobalQuotaByTargetQuery) error {
	return ss.WithDbSession(ctx, func(sess *DBSession) error {
		var used int64

		if query.Target == filesTarget {
			// get quota used.
			rawSQL := fmt.Sprintf("SELECT COUNT(*) AS count FROM %s",
				dialect.Quote("file"))

			notFolderCondition := fmt.Sprintf(" WHERE path NOT LIKE '%s'", "%/")
			resp := make([]*targetCount, 0)
			if err := sess.SQL(rawSQL + notFolderCondition).Find(&resp); err != nil {
				return err
			}
			used = resp[0].Count
		} else if query.Target != alertRuleTarget || query.UnifiedAlertingEnabled {
			// get quota used.
			rawSQL := fmt.Sprintf("SELECT COUNT(*) AS count FROM %s",
				dialect.Quote(query.Target))

			if query.Target == dashboardTarget {
				rawSQL += fmt.Sprintf(" WHERE is_folder=%s", dialect.BooleanStr(false))
			}
			// removing service accounts from count
			if query.Target == dialect.Quote("user") {
				rawSQL += fmt.Sprintf(" WHERE is_service_account=%s", dialect.BooleanStr(false))
			}
			resp := make([]*targetCount, 0)
			if err := sess.SQL(rawSQL).Find(&resp); err != nil {
				return err
			}
			used = resp[0].Count
		}

		query.Result = &models.GlobalQuotaDTO{
			Target: query.Target,
			Limit:  query.Default,
			Used:   used,
		}

		return nil
	})
}
