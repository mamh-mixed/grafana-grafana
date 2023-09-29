package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/grafana/alerting/notify"
	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
	"github.com/prometheus/common/model"

	"github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/util"
)

// AlertRuleFromProvisionedAlertRule converts definitions.ProvisionedAlertRule to models.AlertRule
func AlertRuleFromProvisionedAlertRule(a definitions.ProvisionedAlertRule) (models.AlertRule, error) {
	return models.AlertRule{
		ID:           a.ID,
		UID:          a.UID,
		OrgID:        a.OrgID,
		NamespaceUID: a.FolderUID,
		RuleGroup:    a.RuleGroup,
		Title:        a.Title,
		Condition:    a.Condition,
		Data:         AlertQueriesFromApiAlertQueries(a.Data),
		Updated:      a.Updated,
		NoDataState:  models.NoDataState(a.NoDataState),          // TODO there must be a validation
		ExecErrState: models.ExecutionErrorState(a.ExecErrState), // TODO there must be a validation
		For:          time.Duration(a.For),
		Annotations:  a.Annotations,
		Labels:       a.Labels,
		IsPaused:     a.IsPaused,
	}, nil
}

// ProvisionedAlertRuleFromAlertRule converts models.AlertRule to definitions.ProvisionedAlertRule and sets provided provenance status
func ProvisionedAlertRuleFromAlertRule(rule models.AlertRule, provenance models.Provenance) definitions.ProvisionedAlertRule {
	return definitions.ProvisionedAlertRule{
		ID:           rule.ID,
		UID:          rule.UID,
		OrgID:        rule.OrgID,
		FolderUID:    rule.NamespaceUID,
		RuleGroup:    rule.RuleGroup,
		Title:        rule.Title,
		For:          model.Duration(rule.For),
		Condition:    rule.Condition,
		Data:         ApiAlertQueriesFromAlertQueries(rule.Data),
		Updated:      rule.Updated,
		NoDataState:  definitions.NoDataState(rule.NoDataState),          // TODO there may be a validation
		ExecErrState: definitions.ExecutionErrorState(rule.ExecErrState), // TODO there may be a validation
		Annotations:  rule.Annotations,
		Labels:       rule.Labels,
		Provenance:   definitions.Provenance(provenance), // TODO validate enum conversion?
		IsPaused:     rule.IsPaused,
	}
}

// ProvisionedAlertRuleFromAlertRules converts a collection of models.AlertRule to definitions.ProvisionedAlertRules with provenance status models.ProvenanceNone
func ProvisionedAlertRuleFromAlertRules(rules []*models.AlertRule) definitions.ProvisionedAlertRules {
	result := make([]definitions.ProvisionedAlertRule, 0, len(rules))
	for _, r := range rules {
		result = append(result, ProvisionedAlertRuleFromAlertRule(*r, models.ProvenanceNone))
	}
	return result
}

// AlertQueriesFromApiAlertQueries converts a collection of definitions.AlertQuery to collection of models.AlertQuery
func AlertQueriesFromApiAlertQueries(queries []definitions.AlertQuery) []models.AlertQuery {
	result := make([]models.AlertQuery, 0, len(queries))
	for _, q := range queries {
		result = append(result, models.AlertQuery{
			RefID:     q.RefID,
			QueryType: q.QueryType,
			RelativeTimeRange: models.RelativeTimeRange{
				From: models.Duration(q.RelativeTimeRange.From),
				To:   models.Duration(q.RelativeTimeRange.To),
			},
			DatasourceUID: q.DatasourceUID,
			Model:         q.Model,
		})
	}
	return result
}

// ApiAlertQueriesFromAlertQueries converts a collection of models.AlertQuery to collection of definitions.AlertQuery
func ApiAlertQueriesFromAlertQueries(queries []models.AlertQuery) []definitions.AlertQuery {
	result := make([]definitions.AlertQuery, 0, len(queries))
	for _, q := range queries {
		result = append(result, definitions.AlertQuery{
			RefID:     q.RefID,
			QueryType: q.QueryType,
			RelativeTimeRange: definitions.RelativeTimeRange{
				From: definitions.Duration(q.RelativeTimeRange.From),
				To:   definitions.Duration(q.RelativeTimeRange.To),
			},
			DatasourceUID: q.DatasourceUID,
			Model:         q.Model,
		})
	}
	return result
}

func AlertRuleGroupFromApiAlertRuleGroup(a definitions.AlertRuleGroup) (models.AlertRuleGroup, error) {
	ruleGroup := models.AlertRuleGroup{
		Title:     a.Title,
		FolderUID: a.FolderUID,
		Interval:  a.Interval,
	}
	for i := range a.Rules {
		converted, err := AlertRuleFromProvisionedAlertRule(a.Rules[i])
		if err != nil {
			return models.AlertRuleGroup{}, err
		}
		ruleGroup.Rules = append(ruleGroup.Rules, converted)
	}
	return ruleGroup, nil
}

func ApiAlertRuleGroupFromAlertRuleGroup(d models.AlertRuleGroup) definitions.AlertRuleGroup {
	rules := make([]definitions.ProvisionedAlertRule, 0, len(d.Rules))
	for i := range d.Rules {
		rules = append(rules, ProvisionedAlertRuleFromAlertRule(d.Rules[i], d.Provenance))
	}
	return definitions.AlertRuleGroup{
		Title:     d.Title,
		FolderUID: d.FolderUID,
		Interval:  d.Interval,
		Rules:     rules,
	}
}

// AlertingFileExportFromAlertRuleGroupWithFolderTitle creates an definitions.AlertingFileExport DTO from []models.AlertRuleGroupWithFolderTitle.
func AlertingFileExportFromAlertRuleGroupWithFolderTitle(groups []models.AlertRuleGroupWithFolderTitle) (definitions.AlertingFileExport, error) {
	f := definitions.AlertingFileExport{APIVersion: 1}
	for _, group := range groups {
		export, err := AlertRuleGroupExportFromAlertRuleGroupWithFolderTitle(group)
		if err != nil {
			return definitions.AlertingFileExport{}, err
		}
		f.Groups = append(f.Groups, export)
	}
	return f, nil
}

// AlertRuleGroupExportFromAlertRuleGroupWithFolderTitle creates a definitions.AlertRuleGroupExport DTO from models.AlertRuleGroup.
func AlertRuleGroupExportFromAlertRuleGroupWithFolderTitle(d models.AlertRuleGroupWithFolderTitle) (definitions.AlertRuleGroupExport, error) {
	rules := make([]definitions.AlertRuleExport, 0, len(d.Rules))
	for i := range d.Rules {
		alert, err := AlertRuleExportFromAlertRule(d.Rules[i])
		if err != nil {
			return definitions.AlertRuleGroupExport{}, err
		}
		rules = append(rules, alert)
	}
	return definitions.AlertRuleGroupExport{
		OrgID:           d.OrgID,
		Name:            d.Title,
		Folder:          d.FolderTitle,
		FolderUID:       d.FolderUID,
		Interval:        model.Duration(time.Duration(d.Interval) * time.Second),
		IntervalSeconds: d.Interval,
		Rules:           rules,
	}, nil
}

// AlertRuleExportFromAlertRule creates a definitions.AlertRuleExport DTO from models.AlertRule.
func AlertRuleExportFromAlertRule(rule models.AlertRule) (definitions.AlertRuleExport, error) {
	data := make([]definitions.AlertQueryExport, 0, len(rule.Data))
	for i := range rule.Data {
		query, err := AlertQueryExportFromAlertQuery(rule.Data[i])
		if err != nil {
			return definitions.AlertRuleExport{}, err
		}
		data = append(data, query)
	}

	var dashboardUID string
	if rule.DashboardUID != nil {
		dashboardUID = *rule.DashboardUID
	}

	var panelID int64
	if rule.PanelID != nil {
		panelID = *rule.PanelID
	}

	return definitions.AlertRuleExport{
		UID:          rule.UID,
		Title:        rule.Title,
		For:          model.Duration(rule.For),
		ForSeconds:   int64(rule.For.Seconds()),
		Condition:    rule.Condition,
		Data:         data,
		DashboardUID: dashboardUID,
		PanelID:      panelID,
		NoDataState:  definitions.NoDataState(rule.NoDataState),
		ExecErrState: definitions.ExecutionErrorState(rule.ExecErrState),
		Annotations:  rule.Annotations,
		Labels:       rule.Labels,
		IsPaused:     rule.IsPaused,
	}, nil
}

// AlertQueryExportFromAlertQuery creates a definitions.AlertQueryExport DTO from models.AlertQuery.
func AlertQueryExportFromAlertQuery(query models.AlertQuery) (definitions.AlertQueryExport, error) {
	// We unmarshal the json.RawMessage model into a map in order to facilitate yaml marshalling.
	var mdl map[string]any
	err := json.Unmarshal(query.Model, &mdl)
	if err != nil {
		return definitions.AlertQueryExport{}, err
	}
	return definitions.AlertQueryExport{
		RefID:     query.RefID,
		QueryType: query.QueryType,
		RelativeTimeRange: definitions.RelativeTimeRangeExport{
			FromSeconds: int64(time.Duration(query.RelativeTimeRange.From).Seconds()),
			ToSeconds:   int64(time.Duration(query.RelativeTimeRange.To).Seconds()),
		},
		DatasourceUID: query.DatasourceUID,
		Model:         mdl,
		ModelString:   string(query.Model),
	}, nil
}

// AlertingFileExportFromEmbeddedContactPoints creates a definitions.AlertingFileExport DTO from []definitions.EmbeddedContactPoint.
func AlertingFileExportFromEmbeddedContactPoints(orgID int64, ecps []definitions.EmbeddedContactPoint) (definitions.AlertingFileExport, error) {
	f := definitions.AlertingFileExport{APIVersion: 1}

	cache := make(map[string]*definitions.ContactPointExport)
	contactPoints := make([]*definitions.ContactPointExport, 0)
	for _, ecp := range ecps {
		c, ok := cache[ecp.Name]
		if !ok {
			c = &definitions.ContactPointExport{
				OrgID:     orgID,
				Name:      ecp.Name,
				Receivers: make([]definitions.ReceiverExport, 0),
			}
			cache[ecp.Name] = c
			contactPoints = append(contactPoints, c)
		}

		recv, err := ReceiverExportFromEmbeddedContactPoint(ecp)
		if err != nil {
			return definitions.AlertingFileExport{}, err
		}
		c.Receivers = append(c.Receivers, recv)
	}

	for _, c := range contactPoints {
		f.ContactPoints = append(f.ContactPoints, *c)
	}
	return f, nil
}

// ReceiverExportFromEmbeddedContactPoint creates a definitions.ReceiverExport DTO from definitions.EmbeddedContactPoint.
func ReceiverExportFromEmbeddedContactPoint(contact definitions.EmbeddedContactPoint) (definitions.ReceiverExport, error) {
	raw, err := contact.Settings.MarshalJSON()
	if err != nil {
		return definitions.ReceiverExport{}, err
	}
	return definitions.ReceiverExport{
		UID:                   contact.UID,
		Type:                  contact.Type,
		Settings:              raw,
		DisableResolveMessage: contact.DisableResolveMessage,
	}, nil
}

// AlertingFileExportFromRoute creates a definitions.AlertingFileExport DTO from definitions.Route.
func AlertingFileExportFromRoute(orgID int64, route definitions.Route) (definitions.AlertingFileExport, error) {
	f := definitions.AlertingFileExport{
		APIVersion: 1,
		Policies: []definitions.NotificationPolicyExport{{
			OrgID:  orgID,
			Policy: RouteExportFromRoute(&route),
		}},
	}
	return f, nil
}

// RouteExportFromRoute creates a definitions.RouteExport DTO from definitions.Route.
func RouteExportFromRoute(route *definitions.Route) *definitions.RouteExport {
	emptyStringIfNil := func(d *model.Duration) *string {
		if d == nil {
			return nil
		}
		s := d.String()
		return &s
	}

	matchers := make([]*definitions.MatcherExport, 0, len(route.ObjectMatchers))
	for _, matcher := range route.ObjectMatchers {
		matchers = append(matchers, &definitions.MatcherExport{
			Label: matcher.Name,
			Match: matcher.Type.String(),
			Value: matcher.Value,
		})
	}

	export := definitions.RouteExport{
		Receiver:            route.Receiver,
		GroupByStr:          route.GroupByStr,
		Match:               route.Match,
		MatchRE:             route.MatchRE,
		Matchers:            route.Matchers,
		ObjectMatchers:      route.ObjectMatchers,
		ObjectMatchersSlice: matchers,
		MuteTimeIntervals:   route.MuteTimeIntervals,
		Continue:            func(b bool) *bool { return &b }(route.Continue),
		GroupWait:           emptyStringIfNil(route.GroupWait),
		GroupInterval:       emptyStringIfNil(route.GroupInterval),
		RepeatInterval:      emptyStringIfNil(route.RepeatInterval),
	}

	if len(route.Routes) > 0 {
		export.Routes = make([]*definitions.RouteExport, 0, len(route.Routes))
		for _, r := range route.Routes {
			export.Routes = append(export.Routes, RouteExportFromRoute(r))
		}
	}

	return &export
}

func ContactPointFromContactPointExport(rawContactPoint definitions.ContactPointExport) (definitions.ContactPoint, error) {
	j := jsoniter.ConfigCompatibleWithStandardLibrary
	j.RegisterExtension(&contactPointsExtension{})

	contactPoint := definitions.ContactPoint{
		Name: rawContactPoint.Name,
	}
	var errs []error
	for _, rawIntegration := range rawContactPoint.Receivers {
		err := parseIntegration(j, &contactPoint, rawIntegration.Type, rawIntegration.DisableResolveMessage, json.RawMessage(rawIntegration.Settings))
		if err != nil {
			// accumulate errors to report all at once.
			errs = append(errs, fmt.Errorf("failed to parse %s integration (uid:%s): %w", rawIntegration.Type, rawIntegration.UID, err))
		}
	}
	return contactPoint, errors.Join(errs...)
}

// ContactPointToContactPointExport converts definitions.ContactPoint to notify.APIReceiver.
// It uses special extension for jsoniter.API that properly handles marshalling of some specific fields
//nolint:gocyclo
func ContactPointToContactPointExport(cp definitions.ContactPoint) (notify.APIReceiver, error) {
	j := jsoniter.ConfigCompatibleWithStandardLibrary
	j.RegisterExtension(&contactPointsExtension{})

	var integration []*notify.GrafanaIntegrationConfig

	var errs []error
	for _, i := range cp.Alertmanager {
		el, err := marshallIntegration(j, "prometheus-alertmanager", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Dingding {
		el, err := marshallIntegration(j, "dingding", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Discord {
		el, err := marshallIntegration(j, "discord", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Email {
		el, err := marshallIntegration(j, "email", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Googlechat {
		el, err := marshallIntegration(j, "googlechat", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Kafka {
		el, err := marshallIntegration(j, "kafka", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Line {
		el, err := marshallIntegration(j, "line", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Opsgenie {
		el, err := marshallIntegration(j, "opsgenie", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Pagerduty {
		el, err := marshallIntegration(j, "pagerduty", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.OnCall {
		el, err := marshallIntegration(j, "oncall", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Pushover {
		el, err := marshallIntegration(j, "pushover", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Sensugo {
		el, err := marshallIntegration(j, "sensugo", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Slack {
		el, err := marshallIntegration(j, "slack", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Teams {
		el, err := marshallIntegration(j, "teams", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Telegram {
		el, err := marshallIntegration(j, "telegram", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Threema {
		el, err := marshallIntegration(j, "threema", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Victorops {
		el, err := marshallIntegration(j, "victorops", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Webhook {
		el, err := marshallIntegration(j, "webhook", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Wecom {
		el, err := marshallIntegration(j, "wecom", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, i := range cp.Webex {
		el, err := marshallIntegration(j, "webex", i, i.IntegrationBase)
		integration = append(integration, el)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return notify.APIReceiver{}, errors.Join(errs...)
	}
	contactPoint := notify.APIReceiver{
		ConfigReceiver:      notify.ConfigReceiver{Name: cp.Name},
		GrafanaIntegrations: notify.GrafanaIntegrations{Integrations: integration},
	}
	return contactPoint, nil
}

func marshallIntegration(json jsoniter.API, integrationType string, integration interface{}, base definitions.IntegrationBase) (*notify.GrafanaIntegrationConfig, error) {
	data, err := json.Marshal(integration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall integration '%s' to JSON: %w", integrationType, err)
	}
	e := &notify.GrafanaIntegrationConfig{
		Type:     integrationType,
		Settings: data,
	}
	if base.DisableResolveMessage != nil {
		e.DisableResolveMessage = *base.DisableResolveMessage
	}
	return e, nil
}

//nolint:gocyclo
func parseIntegration(json jsoniter.API, result *definitions.ContactPoint, receiverType string, disableResolveMessage bool, data json.RawMessage) error {
	base := definitions.IntegrationBase{}
	if disableResolveMessage { // omit the value if false
		base.DisableResolveMessage = util.Pointer(disableResolveMessage)
	}
	var err error
	switch strings.ToLower(receiverType) {
	case "prometheus-alertmanager":
		integration := definitions.AlertmanagerIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Alertmanager = append(result.Alertmanager, integration)
		}
	case "dingding":
		integration := definitions.DingdingIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Dingding = append(result.Dingding, integration)
		}
	case "discord":
		integration := definitions.DiscordIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Discord = append(result.Discord, integration)
		}
	case "email":
		integration := definitions.EmailIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Email = append(result.Email, integration)
		}
	case "googlechat":
		integration := definitions.GooglechatIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Googlechat = append(result.Googlechat, integration)
		}
	case "kafka":
		integration := definitions.KafkaIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Kafka = append(result.Kafka, integration)
		}
	case "line":
		integration := definitions.LineIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Line = append(result.Line, integration)
		}
	case "opsgenie":
		integration := definitions.OpsgenieIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Opsgenie = append(result.Opsgenie, integration)
		}
	case "pagerduty":
		integration := definitions.PagerdutyIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Pagerduty = append(result.Pagerduty, integration)
		}
	case "oncall":
		integration := definitions.OnCallIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.OnCall = append(result.OnCall, integration)
		}
	case "pushover":
		integration := definitions.PushoverIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Pushover = append(result.Pushover, integration)
		}
	case "sensugo":
		integration := definitions.SensugoIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Sensugo = append(result.Sensugo, integration)
		}
	case "slack":
		integration := definitions.SlackIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Slack = append(result.Slack, integration)
		}
	case "teams":
		integration := definitions.TeamsIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Teams = append(result.Teams, integration)
		}
	case "telegram":
		integration := definitions.TelegramIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Telegram = append(result.Telegram, integration)
		}
	case "threema":
		integration := definitions.ThreemaIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Threema = append(result.Threema, integration)
		}
	case "victorops":
		integration := definitions.VictoropsIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Victorops = append(result.Victorops, integration)
		}
	case "webhook":
		integration := definitions.WebhookIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Webhook = append(result.Webhook, integration)
		}
	case "wecom":
		integration := definitions.WecomIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Wecom = append(result.Wecom, integration)
		}
	case "webex":
		integration := definitions.WebexIntegration{IntegrationBase: base}
		if err = json.Unmarshal(data, &integration); err == nil {
			result.Webex = append(result.Webex, integration)
		}
	default:
		err = fmt.Errorf("notifier %s is not supported", receiverType)
	}
	return err
}

// contactPointsExtension extends jsoniter with special decoders for some fields that are encoded differently in the legacy configuration.
type contactPointsExtension struct {
	jsoniter.DummyExtension
}

func (c contactPointsExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	if structDescriptor.Type == reflect2.TypeOf(definitions.EmailIntegration{}) {
		bind := structDescriptor.GetField("Addresses")
		codec := &emailAddressCodec{}
		bind.Decoder = codec
		bind.Encoder = codec
	}
	if structDescriptor.Type == reflect2.TypeOf(definitions.PushoverIntegration{}) {
		codec := &numberAsStringCodec{}
		for _, field := range []string{"AlertingPriority", "OKPriority", "Retry", "Expire"} {
			desc := structDescriptor.GetField(field)
			desc.Decoder = codec
			desc.Encoder = codec
		}
	}
	if structDescriptor.Type == reflect2.TypeOf(definitions.WebhookIntegration{}) {
		codec := &numberAsStringCodec{}
		desc := structDescriptor.GetField("MaxAlerts")
		desc.Decoder = codec
		desc.Encoder = codec
	}
}

type emailAddressCodec struct{}

func (d *emailAddressCodec) IsEmpty(ptr unsafe.Pointer) bool {
	f := *(*[]string)(ptr)
	return len(f) == 0
}

func (d *emailAddressCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	f := *(*[]string)(ptr)
	addresses := strings.Join(f, ";")
	stream.WriteString(addresses)
}

func (d *emailAddressCodec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	s := iter.ReadString()
	emails := strings.FieldsFunc(strings.Trim(s, "\""), func(r rune) bool {
		switch r {
		case ',', ';', '\n':
			return true
		}
		return false
	})
	*((*[]string)(ptr)) = emails
}

type numberAsStringCodec struct{}

func (d *numberAsStringCodec) IsEmpty(ptr unsafe.Pointer) bool {
	return *((*(*int))(ptr)) == nil
}

func (d *numberAsStringCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	val := *((*(*int))(ptr))
	if val == nil {
		stream.WriteNil()
		return
	}
	stream.WriteInt(*val)
}

func (d *numberAsStringCodec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	valueType := iter.WhatIsNext()
	var value int
	switch valueType {
	case jsoniter.NumberValue:
		value = iter.ReadInt()
	case jsoniter.StringValue:
		str := iter.ReadString()
		num, err := strconv.ParseInt(str, 10, 32)
		if err != nil {
			iter.ReportError("numberAsStringCodec", fmt.Sprintf("string does not represent an 32-bit integer number: %s", err.Error()))
		}
		value = int(num)
	default:
		iter.ReportError("numberAsStringCodec", "not number or string")
	}
	*((*(*int))(ptr)) = &value
}
