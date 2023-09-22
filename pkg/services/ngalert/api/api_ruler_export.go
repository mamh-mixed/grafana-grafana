package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/folder"
	apimodels "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	ngmodels "github.com/grafana/grafana/pkg/services/ngalert/models"
)

// ExportFromPayload converts the rule groups from the argument `ruleGroupConfig` to export format. All rules are expected to be fully specified. The access to data sources mentioned in the rules is not enforced.
// Can return 403 StatusForbidden if user is not authorized to read folder `namespaceTitle`
func (srv RulerSrv) ExportFromPayload(c *contextmodel.ReqContext, ruleGroupConfig apimodels.PostableRuleGroupConfig, namespaceTitle string) response.Response {
	namespace, err := srv.store.GetNamespaceByTitle(c.Req.Context(), namespaceTitle, c.SignedInUser.OrgID, c.SignedInUser)
	if err != nil {
		return toNamespaceErrorResponse(err)
	}

	rules, err := validateRuleGroup(&ruleGroupConfig, c.SignedInUser.OrgID, namespace, srv.cfg)
	if err != nil {
		return ErrResp(http.StatusBadRequest, err, "")
	}

	r := make([]ngmodels.AlertRule, 0, len(rules))
	for _, rule := range rules {
		r = append(r, rule.AlertRule)
	}

	groupsWithTitle := []ngmodels.AlertRuleGroupWithFolderTitle{
		{
			AlertRuleGroup: &ngmodels.AlertRuleGroup{
				Title:     rules[0].RuleGroup,
				FolderUID: namespace.UID,
				Interval:  rules[0].IntervalSeconds,
				Rules:     r,
			},
			OrgID:       c.OrgID,
			FolderTitle: namespace.Title,
		},
	}

	e, err := AlertingFileExportFromAlertRuleGroupWithFolderTitle(groupsWithTitle)
	if err != nil {
		return ErrResp(http.StatusInternalServerError, err, "failed to create alerting file export")
	}

	return exportResponse(c, e)
}

// ExportRules reads alert rules that user has access to from database according to the filters.
func (srv RulerSrv) ExportRules(c *contextmodel.ReqContext) response.Response {
	hasAccess := accesscontrol.HasAccess(srv.ac, c)
	folderUIDs := c.QueryStrings("folderUid")
	group := c.Query("group")
	uid := c.Query("ruleUid")

	var resultGroup []ngmodels.AlertRuleGroupWithFolderTitle

	if uid != "" || group != "" {
		var rules []*ngmodels.AlertRule
		var namespace *folder.Folder
		if uid != "" {
			if group != "" || len(folderUIDs) > 0 {
				return ErrResp(http.StatusBadRequest, errors.New("group and folder should not be specified when a single rule is requested"), "")
			}
			q := ngmodels.GetAlertRulesGroupByRuleUIDQuery{
				UID:   uid,
				OrgID: c.OrgID,
			}
			var err error
			rules, err = srv.store.GetAlertRulesGroupByRuleUID(c.Req.Context(), &q)
			if err != nil {
				return ErrResp(http.StatusInternalServerError, err, "failed to get rule from database")
			}
			namespace, err = srv.store.GetNamespaceByUID(c.Req.Context(), rules[0].NamespaceUID, c.SignedInUser.OrgID, c.SignedInUser)
			if err != nil {
				return toNamespaceErrorResponse(err)
			}
		}
		if group != "" {
			if len(folderUIDs) != 1 || folderUIDs[0] == "" {
				return ErrResp(http.StatusBadRequest,
					fmt.Errorf("group name must be specified together with a single folder_uid parameter. Got %d", len(folderUIDs)),
					"",
				)
			}
			var err error
			namespace, err = srv.store.GetNamespaceByUID(c.Req.Context(), rules[0].NamespaceUID, c.SignedInUser.OrgID, c.SignedInUser)
			if err != nil {
				return toNamespaceErrorResponse(err)
			}

			q := ngmodels.ListAlertRulesQuery{
				OrgID:         c.OrgID,
				NamespaceUIDs: []string{folderUIDs[0]},
				RuleGroup:     group,
			}
			rules, err = srv.store.ListAlertRules(c.Req.Context(), &q)
			if err != nil {
				return ErrResp(http.StatusInternalServerError, err, "failed to get rule from database")
			}
			ngmodels.RulesGroup(rules).SortByGroupIndex()
		}

		if !authorizeAccessToRuleGroup(rules, hasAccess) {
			return ErrResp(http.StatusUnauthorized, fmt.Errorf("%w to access rules in this group", ErrAuthorization), "")
		}

		result := make([]ngmodels.AlertRule, 0, len(rules))

		for _, r := range rules {
			if uid != "" {
				if r.UID == uid {
					result = append(result, *r)
					break
				}
				continue
			}
			result = append(result, *r)
		}

		if len(result) == 0 {
			return response.Empty(http.StatusNotFound)
		}

		resultGroup = []ngmodels.AlertRuleGroupWithFolderTitle{
			{
				AlertRuleGroup: &ngmodels.AlertRuleGroup{
					Title:     rules[0].RuleGroup,
					FolderUID: rules[0].NamespaceUID,
					Interval:  rules[0].IntervalSeconds,
					Rules:     result,
				},
				OrgID:       c.OrgID,
				FolderTitle: namespace.Title,
			},
		}
	} else {
		folders, err := srv.store.GetUserVisibleNamespaces(c.Req.Context(), c.OrgID, c.SignedInUser)
		if err != nil {
			return ErrResp(http.StatusInternalServerError, err, "failed to get namespaces visible to the user")
		}
		query := ngmodels.ListAlertRulesQuery{
			OrgID:         c.OrgID,
			NamespaceUIDs: nil,
		}
		if len(folderUIDs) > 0 {
			for _, folderUID := range folderUIDs {
				if _, ok := folders[folderUID]; ok {
					query.NamespaceUIDs = append(query.NamespaceUIDs)
				}
			}
			if len(query.NamespaceUIDs) == 0 {
				return ErrResp(http.StatusUnauthorized, fmt.Errorf("%w to access rules in selected folders", ErrAuthorization), "")
			}
		}
		rules, err := srv.store.ListAlertRules(c.Req.Context(), &query)
		if err != nil {
			return ErrResp(http.StatusInternalServerError, err, "failed to get rule from database")
		}
		byGroupKey := ngmodels.GroupByAlertRuleGroupKey(rules)

		for groupKey, rulesGroup := range byGroupKey {
			namespace, ok := folders[groupKey.NamespaceUID]
			if !ok {
				continue // user does not have access
			}
			if !authorizeAccessToRuleGroup(rulesGroup, hasAccess) {
				continue
			}

			derefRules := make([]ngmodels.AlertRule, 0, len(rulesGroup))
			for _, rule := range rulesGroup {
				derefRules = append(derefRules, *rule)
			}

			resultGroup = append(resultGroup, ngmodels.AlertRuleGroupWithFolderTitle{
				AlertRuleGroup: &ngmodels.AlertRuleGroup{
					Title:     groupKey.RuleGroup,
					FolderUID: namespace.UID,
					Interval:  rulesGroup[0].IntervalSeconds,
					Rules:     derefRules,
				},
				OrgID:       c.OrgID,
				FolderTitle: namespace.Title,
			})
		}
	}

	if len(resultGroup) == 0 {
		return response.Empty(http.StatusNotFound)
	}
	
	e, err := AlertingFileExportFromAlertRuleGroupWithFolderTitle(resultGroup)
	if err != nil {
		return ErrResp(http.StatusInternalServerError, err, "failed to create alerting file export")
	}

	return exportResponse(c, e)
}
