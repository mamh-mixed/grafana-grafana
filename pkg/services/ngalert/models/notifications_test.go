package models

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/util"
)

func TestValidate(t *testing.T) {
	validNotificationSettings := NotificationSettingsGen(NSMuts.WithGroupBy(model.AlertNameLabel, FolderTitleLabel))

	testCases := []struct {
		name                 string
		notificationSettings NotificationSettings
		expErrorContains     string
	}{
		{
			name:                 "valid notification settings",
			notificationSettings: validNotificationSettings(),
		},
		{
			name:                 "missing receiver is invalid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithReceiver("")),
			expErrorContains:     "receiver",
		},
		{
			name:                 "group by empty is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy()),
		},
		{
			name:                 "group by ... is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy("...")),
		},
		{
			name:                 "group by with alert name and folder name labels is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy(model.AlertNameLabel, FolderTitleLabel)),
		},
		{
			name:                 "group by missing alert name label is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy(FolderTitleLabel)),
		},
		{
			name:                 "group by missing folder name label is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy(model.AlertNameLabel)),
		},
		{
			name:                 "group wait empty is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupWait(nil)),
		},
		{
			name:                 "group wait positive is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupWait(util.Pointer(1*time.Second))),
		},
		{
			name:                 "group wait negative is invalid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupWait(util.Pointer(-1*time.Second))),
			expErrorContains:     "group wait",
		},
		{
			name:                 "group interval empty is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupInterval(nil)),
		},
		{
			name:                 "group interval positive is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupInterval(util.Pointer(1*time.Second))),
		},
		{
			name:                 "group interval negative is invalid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupInterval(util.Pointer(-1*time.Second))),
			expErrorContains:     "group interval",
		},
		{
			name:                 "repeat interval empty is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithRepeatInterval(nil)),
		},
		{
			name:                 "repeat interval positive is valid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithRepeatInterval(util.Pointer(1*time.Second))),
		},
		{
			name:                 "repeat interval negative is invalid",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithRepeatInterval(util.Pointer(-1*time.Second))),
			expErrorContains:     "repeat interval",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.notificationSettings.Validate()
			if tt.expErrorContains != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expErrorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNotificationSettingsLabels(t *testing.T) {
	testCases := []struct {
		name                 string
		notificationSettings NotificationSettings
		labels               data.Labels
	}{
		{
			name:                 "default notification settings",
			notificationSettings: NewDefaultNotificationSettings("receiver name"),
			labels: data.Labels{
				AutogeneratedRouteLabel:             "true",
				AutogeneratedRouteReceiverNameLabel: "receiver name",
			},
		},
		{
			name: "default notification settings with hardcoded default group by",
			notificationSettings: NotificationSettings{
				Receiver: "receiver name",
				GroupBy:  DefaultNotificationSettingsGroupBy,
			},
			labels: data.Labels{
				AutogeneratedRouteLabel:             "true",
				AutogeneratedRouteReceiverNameLabel: "receiver name",
				AutogeneratedRouteSettingsHashLabel: "6027cdeaff62ba3f",
			},
		},
		{
			name: "custom notification settings",
			notificationSettings: NotificationSettings{
				Receiver:          "receiver name",
				GroupBy:           []string{"label1", "label2"},
				GroupWait:         util.Pointer(model.Duration(1 * time.Minute)),
				GroupInterval:     util.Pointer(model.Duration(2 * time.Minute)),
				RepeatInterval:    util.Pointer(model.Duration(3 * time.Minute)),
				MuteTimeIntervals: []string{"maintenance1", "maintenance2"},
			},
			labels: data.Labels{
				AutogeneratedRouteLabel:             "true",
				AutogeneratedRouteReceiverNameLabel: "receiver name",
				AutogeneratedRouteSettingsHashLabel: "47164c92f2986a35",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			labels := tt.notificationSettings.ToLabels()
			require.Equal(t, tt.labels, labels)
		})
	}
}

func TestNormalizedGroupBy(t *testing.T) {
	validNotificationSettings := NotificationSettingsGen()

	testCases := []struct {
		name                      string
		notificationSettings      NotificationSettings
		expectedNormalizedGroupBy []string
	}{
		{
			name:                      "default group by is normal",
			notificationSettings:      CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy(DefaultNotificationSettingsGroupBy...)),
			expectedNormalizedGroupBy: DefaultNotificationSettingsGroupBy,
		},
		{
			name:                      "group by ... is normal",
			notificationSettings:      CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy(GroupByAll)),
			expectedNormalizedGroupBy: []string{GroupByAll},
		},
		{
			name:                 "group by empty is normal",
			notificationSettings: CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy()),
		},
		{
			name:                      "group by with ... and other labels is not normal",
			notificationSettings:      CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy(FolderTitleLabel, model.AlertNameLabel, GroupByAll)),
			expectedNormalizedGroupBy: []string{GroupByAll},
		},
		{
			name:                      "group by missing alert name label is not normal",
			notificationSettings:      CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy(FolderTitleLabel)),
			expectedNormalizedGroupBy: DefaultNotificationSettingsGroupBy,
		},
		{
			name:                      "group by missing folder name label is not normal",
			notificationSettings:      CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy(model.AlertNameLabel)),
			expectedNormalizedGroupBy: DefaultNotificationSettingsGroupBy,
		},
		{
			name:                      "group by with all required labels plus extra is normal",
			notificationSettings:      CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy(FolderTitleLabel, model.AlertNameLabel, "custom")),
			expectedNormalizedGroupBy: []string{FolderTitleLabel, model.AlertNameLabel, "custom"},
		},
		{
			name:                      "ensure consistent ordering, required labels in front followed by sorted custom labels ",
			notificationSettings:      CopyNotificationSettings(validNotificationSettings(), NSMuts.WithGroupBy("custom", model.AlertNameLabel, "something", FolderTitleLabel, "other")),
			expectedNormalizedGroupBy: []string{FolderTitleLabel, model.AlertNameLabel, "custom", "other", "something"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			normalized := tt.notificationSettings.NormalizedGroupBy()
			require.Equal(t, normalized, tt.expectedNormalizedGroupBy)
		})
	}
}
