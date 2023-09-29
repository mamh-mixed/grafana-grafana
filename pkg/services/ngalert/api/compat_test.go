package api

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/util"
)

func TestToModel(t *testing.T) {
	t.Run("if no rules are provided the rule field should be nil", func(t *testing.T) {
		ruleGroup := definitions.AlertRuleGroup{
			Title:     "123",
			FolderUID: "123",
			Interval:  10,
		}
		tm, err := AlertRuleGroupFromApiAlertRuleGroup(ruleGroup)
		require.NoError(t, err)
		require.Nil(t, tm.Rules)
	})
	t.Run("if rules are provided the rule field should be not nil", func(t *testing.T) {
		ruleGroup := definitions.AlertRuleGroup{
			Title:     "123",
			FolderUID: "123",
			Interval:  10,
			Rules: []definitions.ProvisionedAlertRule{
				{
					UID: "1",
				},
			},
		}
		tm, err := AlertRuleGroupFromApiAlertRuleGroup(ruleGroup)
		require.NoError(t, err)
		require.Len(t, tm.Rules, 1)
	})
}

func TestContactPointFromContactPointExports(t *testing.T) {
	cp := definitions.ContactPointExport{
		OrgID: 1,
		Name:  "Test",
		Receivers: []definitions.ReceiverExport{
			{
				Type:                  "email",
				Settings:              definitions.RawMessage(`{"addresses": "test@grafana.com,test2@grafana.com;test3@grafana.com\ntest4@granafa.com"}`),
				DisableResolveMessage: false,
			},
			{
				Type: "pushover",
				Settings: definitions.RawMessage(`{
				"priority": 1,
				"okPriority": 2,
				"retry": "555",
				"expire": "333"
			}`),
				DisableResolveMessage: false,
			},
		},
	}
	res, err := ContactPointFromContactPointExports(cp)
	require.NoError(t, err)
	require.Len(t, res.Email, 1)
	require.EqualValues(t, []string{
		"test@grafana.com",
		"test2@grafana.com",
		"test3@grafana.com",
		"test4@granafa.com",
	}, res.Email[0].Addresses)
	require.Len(t, res.Pushover, 1)
	require.Equal(t, definitions.PushoverIntegration{
		IntegrationBase:  definitions.IntegrationBase{},
		UserKey:          "",
		APIToken:         "",
		AlertingPriority: util.Pointer(1),
		OKPriority:       util.Pointer(2),
		Retry:            util.Pointer(555),
		Expire:           util.Pointer(333),
		Device:           nil,
		AlertingSound:    nil,
		OKSound:          nil,
		Title:            nil,
		Message:          nil,
	}, res.Pushover[0])
}
