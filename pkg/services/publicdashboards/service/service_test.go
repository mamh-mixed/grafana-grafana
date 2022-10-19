package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/components/simplejson"
	dashboard2 "github.com/grafana/grafana/pkg/coremodel/dashboard"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/annotations"
	"github.com/grafana/grafana/pkg/services/annotations/annotationsimpl"
	"github.com/grafana/grafana/pkg/services/dashboards"
	dashboardsDB "github.com/grafana/grafana/pkg/services/dashboards/database"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	. "github.com/grafana/grafana/pkg/services/publicdashboards"
	"github.com/grafana/grafana/pkg/services/publicdashboards/database"
	"github.com/grafana/grafana/pkg/services/publicdashboards/internal"
	. "github.com/grafana/grafana/pkg/services/publicdashboards/models"
	"github.com/grafana/grafana/pkg/services/serviceaccounts/tests"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/services/tag/tagimpl"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tsdb/intervalv2"
)

var timeSettings = &TimeSettings{From: "now-12h", To: "now"}
var defaultPubdashTimeSettings = &TimeSettings{}
var dashboardData = simplejson.NewFromAny(map[string]interface{}{"time": map[string]interface{}{"from": "now-8h", "to": "now"}})
var SignedInUser = &user.SignedInUser{UserID: 1234, Login: "user@login.com"}

func TestLogPrefix(t *testing.T) {
	assert.Equal(t, LogPrefix, "publicdashboards.service")
}

func TestGetAnnotations(t *testing.T) {
	t.Run("will build anonymous user with correct permissions to get annotations", func(t *testing.T) {
		sqlStore := sqlstore.InitTestDB(t)
		config := setting.NewCfg()
		tagService := tagimpl.ProvideService(sqlStore, sqlStore.Cfg)
		annotationsRepo := annotationsimpl.ProvideService(sqlStore, config, tagService)
		fakeStore := FakePublicDashboardStore{}
		service := &PublicDashboardServiceImpl{
			log:             log.New("test.logger"),
			store:           &fakeStore,
			AnnotationsRepo: annotationsRepo,
		}
		fakeStore.On("GetPublicDashboard", mock.Anything, mock.AnythingOfType("string")).
			Return(&PublicDashboard{Uid: "uid1", IsEnabled: true}, models.NewDashboard("dash1"), nil)
		reqDTO := AnnotationsQueryDTO{
			From: 1,
			To:   2,
		}
		dash := models.NewDashboard("testDashboard")

		items, _ := service.GetAnnotations(context.Background(), reqDTO, "abc123")
		anonUser := service.BuildAnonymousUser(context.Background(), dash)

		assert.Equal(t, "dashboards:*", anonUser.Permissions[0]["dashboards:read"][0])
		assert.Len(t, items, 0)
	})

	t.Run("Test events from tag queries overwrite built-in annotation queries and duplicate events are not returned", func(t *testing.T) {
		dash := models.NewDashboard("test")
		color := "red"
		name := "annoName"
		grafanaAnnotation := DashAnnotation{
			Datasource: CreateDatasource("grafana", "grafana"),
			Enable:     true,
			Name:       &name,
			IconColor:  &color,
			Target: &dashboard2.AnnotationTarget{
				Limit:    100,
				MatchAny: false,
				Tags:     nil,
				Type:     "dashboard",
			},
			Type: "dashboard",
		}
		grafanaTagAnnotation := DashAnnotation{
			Datasource: CreateDatasource("grafana", "grafana"),
			Enable:     true,
			Name:       &name,
			IconColor:  &color,
			Target: &dashboard2.AnnotationTarget{
				Limit:    100,
				MatchAny: false,
				Tags:     []string{"tag1"},
				Type:     "tags",
			},
		}
		annos := []DashAnnotation{grafanaAnnotation, grafanaTagAnnotation}
		dashboard := AddAnnotationsToDashboard(t, dash, annos)

		annotationsRepo := annotations.FakeAnnotationsRepo{}
		fakeStore := FakePublicDashboardStore{}
		service := &PublicDashboardServiceImpl{
			log:             log.New("test.logger"),
			store:           &fakeStore,
			AnnotationsRepo: &annotationsRepo,
		}
		pubdash := &PublicDashboard{Uid: "uid1", IsEnabled: true, OrgId: 1, DashboardUid: dashboard.Uid}
		fakeStore.On("GetPublicDashboard", mock.Anything, mock.AnythingOfType("string")).Return(pubdash, dashboard, nil)
		annotationsRepo.On("Find", mock.Anything, mock.Anything).Return([]*annotations.ItemDTO{
			{
				Id:          1,
				DashboardId: 1,
				PanelId:     1,
				Tags:        []string{"tag1"},
				TimeEnd:     2,
				Time:        2,
				Text:        "text",
			},
		}, nil).Maybe()

		items, err := service.GetAnnotations(context.Background(), AnnotationsQueryDTO{}, "abc123")

		expected := AnnotationEvent{
			Id:          1,
			DashboardId: 1,
			PanelId:     0,
			Tags:        []string{"tag1"},
			IsRegion:    false,
			Text:        "text",
			Color:       color,
			Time:        2,
			TimeEnd:     2,
			Source:      grafanaTagAnnotation,
		}
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, expected, items[0])
	})

	t.Run("Test panelId set to zero when annotation event is for a tags query", func(t *testing.T) {
		dash := models.NewDashboard("test")
		color := "red"
		name := "annoName"
		grafanaAnnotation := DashAnnotation{
			Datasource: CreateDatasource("grafana", "grafana"),
			Enable:     true,
			Name:       &name,
			IconColor:  &color,
			Target: &dashboard2.AnnotationTarget{
				Limit:    100,
				MatchAny: false,
				Tags:     []string{"tag1"},
				Type:     "tags",
			},
		}
		annos := []DashAnnotation{grafanaAnnotation}
		dashboard := AddAnnotationsToDashboard(t, dash, annos)

		annotationsRepo := annotations.FakeAnnotationsRepo{}
		fakeStore := FakePublicDashboardStore{}
		service := &PublicDashboardServiceImpl{
			log:             log.New("test.logger"),
			store:           &fakeStore,
			AnnotationsRepo: &annotationsRepo,
		}
		pubdash := &PublicDashboard{Uid: "uid1", IsEnabled: true, OrgId: 1, DashboardUid: dashboard.Uid}
		fakeStore.On("GetPublicDashboard", mock.Anything, mock.AnythingOfType("string")).Return(pubdash, dashboard, nil)
		annotationsRepo.On("Find", mock.Anything, mock.Anything).Return([]*annotations.ItemDTO{
			{
				Id:          1,
				DashboardId: 1,
				PanelId:     1,
				Tags:        []string{},
				TimeEnd:     1,
				Time:        2,
				Text:        "text",
			},
		}, nil).Maybe()

		items, err := service.GetAnnotations(context.Background(), AnnotationsQueryDTO{}, "abc123")

		expected := AnnotationEvent{
			Id:          1,
			DashboardId: 1,
			PanelId:     0,
			Tags:        []string{},
			IsRegion:    true,
			Text:        "text",
			Color:       color,
			Time:        2,
			TimeEnd:     1,
			Source:      grafanaAnnotation,
		}
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, expected, items[0])
	})

	t.Run("Test can get grafana annotations and will skip annotation queries and disabled annotations", func(t *testing.T) {
		dash := models.NewDashboard("test")
		color := "red"
		name := "annoName"
		disabledGrafanaAnnotation := DashAnnotation{
			Datasource: CreateDatasource("grafana", "grafana"),
			Enable:     false,
			Name:       &name,
			IconColor:  &color,
		}
		grafanaAnnotation := DashAnnotation{
			Datasource: CreateDatasource("grafana", "grafana"),
			Enable:     true,
			Name:       &name,
			IconColor:  &color,
			Target: &dashboard2.AnnotationTarget{
				Limit:    100,
				MatchAny: true,
				Tags:     nil,
				Type:     "dashboard",
			},
			Type: "dashboard",
		}
		queryAnnotation := DashAnnotation{
			Datasource: CreateDatasource("prometheus", "abc123"),
			Enable:     true,
			Name:       &name,
		}
		annos := []DashAnnotation{grafanaAnnotation, queryAnnotation, disabledGrafanaAnnotation}
		dashboard := AddAnnotationsToDashboard(t, dash, annos)

		annotationsRepo := annotations.FakeAnnotationsRepo{}
		fakeStore := FakePublicDashboardStore{}
		service := &PublicDashboardServiceImpl{
			log:             log.New("test.logger"),
			store:           &fakeStore,
			AnnotationsRepo: &annotationsRepo,
		}
		pubdash := &PublicDashboard{Uid: "uid1", IsEnabled: true, OrgId: 1, DashboardUid: dashboard.Uid}
		fakeStore.On("GetPublicDashboard", mock.Anything, mock.AnythingOfType("string")).Return(pubdash, dashboard, nil)
		annotationsRepo.On("Find", mock.Anything, mock.Anything).Return([]*annotations.ItemDTO{
			{
				Id:          1,
				DashboardId: 1,
				PanelId:     1,
				Tags:        []string{},
				TimeEnd:     1,
				Time:        2,
				Text:        "text",
			},
		}, nil).Maybe()

		items, err := service.GetAnnotations(context.Background(), AnnotationsQueryDTO{}, "abc123")

		expected := AnnotationEvent{
			Id:          1,
			DashboardId: 1,
			PanelId:     1,
			Tags:        []string{},
			IsRegion:    true,
			Text:        "text",
			Color:       color,
			Time:        2,
			TimeEnd:     1,
			Source:      grafanaAnnotation,
		}
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, expected, items[0])
	})

	t.Run("test will return nothing when dashboard has no annotations", func(t *testing.T) {
		annotationsRepo := annotations.FakeAnnotationsRepo{}
		fakeStore := FakePublicDashboardStore{}
		service := &PublicDashboardServiceImpl{
			log:             log.New("test.logger"),
			store:           &fakeStore,
			AnnotationsRepo: &annotationsRepo,
		}
		dashboard := models.NewDashboard("dashWithNoAnnotations")
		pubdash := &PublicDashboard{Uid: "uid1", IsEnabled: true, OrgId: 1, DashboardUid: dashboard.Uid}
		fakeStore.On("GetPublicDashboard", mock.Anything, mock.AnythingOfType("string")).Return(pubdash, dashboard, nil)

		items, err := service.GetAnnotations(context.Background(), AnnotationsQueryDTO{}, "abc123")

		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("test will error when annotations repo returns an error", func(t *testing.T) {
		annotationsRepo := annotations.FakeAnnotationsRepo{}
		fakeStore := FakePublicDashboardStore{}
		service := &PublicDashboardServiceImpl{
			log:             log.New("test.logger"),
			store:           &fakeStore,
			AnnotationsRepo: &annotationsRepo,
		}
		dash := models.NewDashboard("test")
		color := "red"
		name := "annoName"
		grafanaAnnotation := DashAnnotation{
			Datasource: CreateDatasource("grafana", "grafana"),
			Enable:     true,
			Name:       &name,
			IconColor:  &color,
			Target: &dashboard2.AnnotationTarget{
				Limit:    100,
				MatchAny: false,
				Tags:     []string{"tag1"},
				Type:     "tags",
			},
		}
		annos := []DashAnnotation{grafanaAnnotation}
		dash = AddAnnotationsToDashboard(t, dash, annos)
		pubdash := &PublicDashboard{Uid: "uid1", IsEnabled: true, OrgId: 1, DashboardUid: dash.Uid}
		fakeStore.On("GetPublicDashboard", mock.Anything, mock.AnythingOfType("string")).Return(pubdash, dash, nil)
		annotationsRepo.On("Find", mock.Anything, mock.Anything).Return(nil, errors.New("failed")).Maybe()

		items, err := service.GetAnnotations(context.Background(), AnnotationsQueryDTO{}, "abc123")

		require.Error(t, err)
		require.Nil(t, items)
	})
}

func TestGetPublicDashboard(t *testing.T) {
	type storeResp struct {
		pd  *PublicDashboard
		d   *models.Dashboard
		err error
	}

	testCases := []struct {
		Name        string
		AccessToken string
		StoreResp   *storeResp
		ErrResp     error
		DashResp    *models.Dashboard
	}{
		{
			Name:        "returns a dashboard",
			AccessToken: "abc123",
			StoreResp: &storeResp{
				pd:  &PublicDashboard{AccessToken: "abcdToken", IsEnabled: true},
				d:   &models.Dashboard{Uid: "mydashboard", Data: dashboardData},
				err: nil,
			},
			ErrResp:  nil,
			DashResp: &models.Dashboard{Uid: "mydashboard", Data: dashboardData},
		},
		{
			Name:        "returns ErrPublicDashboardNotFound when isEnabled is false",
			AccessToken: "abc123",
			StoreResp: &storeResp{
				pd:  &PublicDashboard{AccessToken: "abcdToken", IsEnabled: false},
				d:   &models.Dashboard{Uid: "mydashboard"},
				err: nil,
			},
			ErrResp:  ErrPublicDashboardNotFound,
			DashResp: nil,
		},
		{
			Name:        "returns ErrPublicDashboardNotFound if PublicDashboard missing",
			AccessToken: "abc123",
			StoreResp:   &storeResp{pd: nil, d: nil, err: nil},
			ErrResp:     ErrPublicDashboardNotFound,
			DashResp:    nil,
		},
		{
			Name:        "returns ErrPublicDashboardNotFound if Dashboard missing",
			AccessToken: "abc123",
			StoreResp:   &storeResp{pd: nil, d: nil, err: nil},
			ErrResp:     ErrPublicDashboardNotFound,
			DashResp:    nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			fakeStore := FakePublicDashboardStore{}
			service := &PublicDashboardServiceImpl{
				log:   log.New("test.logger"),
				store: &fakeStore,
			}

			fakeStore.On("GetPublicDashboard", mock.Anything, mock.Anything).
				Return(test.StoreResp.pd, test.StoreResp.d, test.StoreResp.err)

			pdc, dash, err := service.GetPublicDashboard(context.Background(), test.AccessToken)
			if test.ErrResp != nil {
				assert.Error(t, test.ErrResp, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, test.DashResp, dash)

			if test.DashResp != nil {
				assert.NotNil(t, dash.CreatedBy)
				assert.Equal(t, test.StoreResp.pd, pdc)
			}
		})
	}
}

func TestSavePublicDashboard(t *testing.T) {
	t.Run("Saving public dashboard", func(t *testing.T) {
		sqlStore := db.InitTestDB(t)
		dashboardStore := dashboardsDB.ProvideDashboardStore(sqlStore, sqlStore.Cfg, featuremgmt.WithFeatures(), tagimpl.ProvideService(sqlStore, sqlStore.Cfg))
		publicdashboardStore := database.ProvideStore(sqlStore)
		dashboard := insertTestDashboard(t, dashboardStore, "testDashie", 1, 0, true, []map[string]interface{}{}, nil)

		service := &PublicDashboardServiceImpl{
			log:   log.New("test.logger"),
			store: publicdashboardStore,
		}

		dto := &SavePublicDashboardConfigDTO{
			DashboardUid: dashboard.Uid,
			OrgId:        dashboard.OrgId,
			UserId:       7,
			PublicDashboard: &PublicDashboard{
				IsEnabled:    true,
				DashboardUid: "NOTTHESAME",
				OrgId:        9999999,
				TimeSettings: timeSettings,
			},
		}

		_, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)
		require.NoError(t, err)

		pubdash, err := service.GetPublicDashboardConfig(context.Background(), dashboard.OrgId, dashboard.Uid)
		require.NoError(t, err)

		// DashboardUid/OrgId/CreatedBy set by the command, not parameters
		assert.Equal(t, dashboard.Uid, pubdash.DashboardUid)
		assert.Equal(t, dashboard.OrgId, pubdash.OrgId)
		assert.Equal(t, dto.UserId, pubdash.CreatedBy)
		// IsEnabled set by parameters
		assert.Equal(t, dto.PublicDashboard.IsEnabled, pubdash.IsEnabled)
		// CreatedAt set to non-zero time
		assert.NotEqual(t, &time.Time{}, pubdash.CreatedAt)
		// Time settings set by db
		assert.Equal(t, timeSettings, pubdash.TimeSettings)
		// accessToken is valid uuid
		_, err = uuid.Parse(pubdash.AccessToken)
		require.NoError(t, err, "expected a valid UUID, got %s", pubdash.AccessToken)
	})

	t.Run("Validate pubdash has default time setting value", func(t *testing.T) {
		sqlStore := db.InitTestDB(t)
		dashboardStore := dashboardsDB.ProvideDashboardStore(sqlStore, sqlStore.Cfg, featuremgmt.WithFeatures(), tagimpl.ProvideService(sqlStore, sqlStore.Cfg))
		publicdashboardStore := database.ProvideStore(sqlStore)
		dashboard := insertTestDashboard(t, dashboardStore, "testDashie", 1, 0, true, []map[string]interface{}{}, nil)

		service := &PublicDashboardServiceImpl{
			log:   log.New("test.logger"),
			store: publicdashboardStore,
		}

		dto := &SavePublicDashboardConfigDTO{
			DashboardUid: dashboard.Uid,
			OrgId:        dashboard.OrgId,
			UserId:       7,
			PublicDashboard: &PublicDashboard{
				IsEnabled:    true,
				DashboardUid: "NOTTHESAME",
				OrgId:        9999999,
			},
		}

		_, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)
		require.NoError(t, err)

		pubdash, err := service.GetPublicDashboardConfig(context.Background(), dashboard.OrgId, dashboard.Uid)
		require.NoError(t, err)
		assert.Equal(t, defaultPubdashTimeSettings, pubdash.TimeSettings)
	})

	t.Run("Validate pubdash whose dashboard has template variables returns error", func(t *testing.T) {
		sqlStore := db.InitTestDB(t)
		dashboardStore := dashboardsDB.ProvideDashboardStore(sqlStore, sqlStore.Cfg, featuremgmt.WithFeatures(), tagimpl.ProvideService(sqlStore, sqlStore.Cfg))
		publicdashboardStore := database.ProvideStore(sqlStore)
		templateVars := make([]map[string]interface{}, 1)
		dashboard := insertTestDashboard(t, dashboardStore, "testDashie", 1, 0, true, templateVars, nil)

		service := &PublicDashboardServiceImpl{
			log:   log.New("test.logger"),
			store: publicdashboardStore,
		}

		dto := &SavePublicDashboardConfigDTO{
			DashboardUid: dashboard.Uid,
			OrgId:        dashboard.OrgId,
			UserId:       7,
			PublicDashboard: &PublicDashboard{
				IsEnabled:    true,
				DashboardUid: "NOTTHESAME",
				OrgId:        9999999,
			},
		}

		_, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)
		require.Error(t, err)
	})

	t.Run("Pubdash access token generation throws an error and pubdash is not persisted", func(t *testing.T) {
		dashboard := models.NewDashboard("testDashie")

		publicDashboardStore := &FakePublicDashboardStore{}
		publicDashboardStore.On("GetDashboard", mock.Anything, mock.Anything).Return(dashboard, nil)
		publicDashboardStore.On("GetPublicDashboardByUid", mock.Anything, mock.Anything).Return(nil, nil)
		publicDashboardStore.On("GenerateNewPublicDashboardUid", mock.Anything).Return("an-uid", nil)
		publicDashboardStore.On("GenerateNewPublicDashboardAccessToken", mock.Anything).Return("", ErrPublicDashboardFailedGenerateAccessToken)

		service := &PublicDashboardServiceImpl{
			log:   log.New("test.logger"),
			store: publicDashboardStore,
		}

		dto := &SavePublicDashboardConfigDTO{
			DashboardUid: "an-id",
			OrgId:        8,
			UserId:       7,
			PublicDashboard: &PublicDashboard{
				IsEnabled:    true,
				DashboardUid: "NOTTHESAME",
				OrgId:        9999999,
			},
		}

		_, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)

		require.Error(t, err)
		require.Equal(t, err, ErrPublicDashboardFailedGenerateAccessToken)
		publicDashboardStore.AssertNotCalled(t, "SavePublicDashboardConfig")
	})
}

func TestUpdatePublicDashboard(t *testing.T) {
	t.Run("Updating public dashboard", func(t *testing.T) {
		sqlStore := db.InitTestDB(t)
		dashboardStore := dashboardsDB.ProvideDashboardStore(sqlStore, sqlStore.Cfg, featuremgmt.WithFeatures(), tagimpl.ProvideService(sqlStore, sqlStore.Cfg))
		publicdashboardStore := database.ProvideStore(sqlStore)
		dashboard := insertTestDashboard(t, dashboardStore, "testDashie", 1, 0, true, []map[string]interface{}{}, nil)

		service := &PublicDashboardServiceImpl{
			log:   log.New("test.logger"),
			store: publicdashboardStore,
		}

		dto := &SavePublicDashboardConfigDTO{
			DashboardUid: dashboard.Uid,
			OrgId:        dashboard.OrgId,
			UserId:       7,
			PublicDashboard: &PublicDashboard{
				IsEnabled:    true,
				TimeSettings: timeSettings,
			},
		}

		savedPubdash, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)
		require.NoError(t, err)

		// attempt to overwrite settings
		dto = &SavePublicDashboardConfigDTO{
			DashboardUid: dashboard.Uid,
			OrgId:        dashboard.OrgId,
			UserId:       8,
			PublicDashboard: &PublicDashboard{
				Uid:          savedPubdash.Uid,
				OrgId:        9,
				DashboardUid: "abc1234",
				CreatedBy:    9,
				CreatedAt:    time.Time{},

				IsEnabled:    true,
				TimeSettings: timeSettings,
				AccessToken:  "NOTAREALUUID",
			},
		}

		// Since the dto.PublicDashboard has a uid, this will call
		// service.updatePublicDashboardConfig
		updatedPubdash, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)
		require.NoError(t, err)

		// don't get updated
		assert.Equal(t, savedPubdash.DashboardUid, updatedPubdash.DashboardUid)
		assert.Equal(t, savedPubdash.OrgId, updatedPubdash.OrgId)
		assert.Equal(t, savedPubdash.CreatedAt, updatedPubdash.CreatedAt)
		assert.Equal(t, savedPubdash.CreatedBy, updatedPubdash.CreatedBy)
		assert.Equal(t, savedPubdash.AccessToken, updatedPubdash.AccessToken)

		// gets updated
		assert.Equal(t, dto.PublicDashboard.IsEnabled, updatedPubdash.IsEnabled)
		assert.Equal(t, dto.PublicDashboard.TimeSettings, updatedPubdash.TimeSettings)
		assert.Equal(t, dto.UserId, updatedPubdash.UpdatedBy)
		assert.NotEqual(t, &time.Time{}, updatedPubdash.UpdatedAt)
	})

	t.Run("Updating set empty time settings", func(t *testing.T) {
		sqlStore := db.InitTestDB(t)
		dashboardStore := dashboardsDB.ProvideDashboardStore(sqlStore, sqlStore.Cfg, featuremgmt.WithFeatures(), tagimpl.ProvideService(sqlStore, sqlStore.Cfg))
		publicdashboardStore := database.ProvideStore(sqlStore)
		dashboard := insertTestDashboard(t, dashboardStore, "testDashie", 1, 0, true, []map[string]interface{}{}, nil)

		service := &PublicDashboardServiceImpl{
			log:   log.New("test.logger"),
			store: publicdashboardStore,
		}

		dto := &SavePublicDashboardConfigDTO{
			DashboardUid: dashboard.Uid,
			OrgId:        dashboard.OrgId,
			UserId:       7,
			PublicDashboard: &PublicDashboard{
				IsEnabled:    true,
				TimeSettings: timeSettings,
			},
		}

		// Since the dto.PublicDashboard has a uid, this will call
		// service.updatePublicDashboardConfig
		savedPubdash, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)
		require.NoError(t, err)

		// attempt to overwrite settings
		dto = &SavePublicDashboardConfigDTO{
			DashboardUid: dashboard.Uid,
			OrgId:        dashboard.OrgId,
			UserId:       8,
			PublicDashboard: &PublicDashboard{
				Uid:          savedPubdash.Uid,
				OrgId:        9,
				DashboardUid: "abc1234",
				CreatedBy:    9,
				CreatedAt:    time.Time{},

				IsEnabled:   true,
				AccessToken: "NOTAREALUUID",
			},
		}

		updatedPubdash, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)
		require.NoError(t, err)

		assert.Equal(t, &TimeSettings{}, updatedPubdash.TimeSettings)
	})
}

func TestBuildAnonymousUser(t *testing.T) {
	sqlStore := db.InitTestDB(t)
	dashboardStore := dashboardsDB.ProvideDashboardStore(sqlStore, sqlStore.Cfg, featuremgmt.WithFeatures(), tagimpl.ProvideService(sqlStore, sqlStore.Cfg))
	dashboard := insertTestDashboard(t, dashboardStore, "testDashie", 1, 0, true, []map[string]interface{}{}, nil)
	publicdashboardStore := database.ProvideStore(sqlStore)
	service := &PublicDashboardServiceImpl{
		log:   log.New("test.logger"),
		store: publicdashboardStore,
	}

	t.Run("will add datasource read and query permissions to user for each datasource in dashboard", func(t *testing.T) {
		user := service.BuildAnonymousUser(context.Background(), dashboard)

		require.Equal(t, dashboard.OrgId, user.OrgID)
		require.Equal(t, "datasources:uid:ds1", user.Permissions[user.OrgID]["datasources:query"][0])
		require.Equal(t, "datasources:uid:ds3", user.Permissions[user.OrgID]["datasources:query"][1])
		require.Equal(t, "datasources:uid:ds1", user.Permissions[user.OrgID]["datasources:read"][0])
		require.Equal(t, "datasources:uid:ds3", user.Permissions[user.OrgID]["datasources:read"][1])
	})
	t.Run("will add dashboard and annotation permissions needed for getting annotations", func(t *testing.T) {
		user := service.BuildAnonymousUser(context.Background(), dashboard)

		require.Equal(t, dashboard.OrgId, user.OrgID)
		require.Equal(t, "annotations:type:dashboard", user.Permissions[user.OrgID]["annotations:read"][0])
		require.Equal(t, "dashboards:*", user.Permissions[user.OrgID]["dashboards:read"][0])
	})
}

func TestGetMetricRequest(t *testing.T) {
	sqlStore := db.InitTestDB(t)
	dashboardStore := dashboardsDB.ProvideDashboardStore(sqlStore, sqlStore.Cfg, featuremgmt.WithFeatures(), tagimpl.ProvideService(sqlStore, sqlStore.Cfg))
	publicdashboardStore := database.ProvideStore(sqlStore)
	dashboard := insertTestDashboard(t, dashboardStore, "testDashie", 1, 0, true, []map[string]interface{}{}, nil)
	publicDashboard := &PublicDashboard{
		Uid:          "1",
		DashboardUid: dashboard.Uid,
		IsEnabled:    true,
		AccessToken:  "abc123",
	}
	service := &PublicDashboardServiceImpl{
		log:                log.New("test.logger"),
		store:              publicdashboardStore,
		intervalCalculator: intervalv2.NewCalculator(),
	}

	t.Run("will return an error when validation fails", func(t *testing.T) {
		publicDashboardQueryDTO := PublicDashboardQueryDTO{
			IntervalMs:    int64(-1),
			MaxDataPoints: int64(-1),
		}

		_, err := service.GetMetricRequest(context.Background(), dashboard, publicDashboard, 1, publicDashboardQueryDTO)

		require.Error(t, err)
	})

	t.Run("will not return an error when validation succeeds", func(t *testing.T) {
		publicDashboardQueryDTO := PublicDashboardQueryDTO{
			IntervalMs:    int64(1),
			MaxDataPoints: int64(1),
		}
		from, to := internal.GetTimeRangeFromDashboard(t, dashboard.Data)

		metricReq, err := service.GetMetricRequest(context.Background(), dashboard, publicDashboard, 1, publicDashboardQueryDTO)

		require.NoError(t, err)
		require.Equal(t, from, metricReq.From)
		require.Equal(t, to, metricReq.To)
	})
}

func TestGetQueryDataResponse(t *testing.T) {
	sqlStore := sqlstore.InitTestDB(t)
	dashboardStore := dashboardsDB.ProvideDashboardStore(sqlStore, sqlStore.Cfg, featuremgmt.WithFeatures(), tagimpl.ProvideService(sqlStore, sqlStore.Cfg))
	publicdashboardStore := database.ProvideStore(sqlStore)

	service := &PublicDashboardServiceImpl{
		log:                log.New("test.logger"),
		store:              publicdashboardStore,
		intervalCalculator: intervalv2.NewCalculator(),
	}

	publicDashboardQueryDTO := PublicDashboardQueryDTO{
		IntervalMs:    int64(1),
		MaxDataPoints: int64(1),
	}

	t.Run("Returns nil when query is hidden", func(t *testing.T) {
		hiddenQuery := map[string]interface{}{
			"datasource": map[string]interface{}{
				"type": "mysql",
				"uid":  "ds1",
			},
			"hide":  true,
			"refId": "A",
		}
		customPanels := []interface{}{
			map[string]interface{}{
				"id": 1,
				"datasource": map[string]interface{}{
					"uid": "ds1",
				},
				"targets": []interface{}{hiddenQuery},
			}}

		dashboard := insertTestDashboard(t, dashboardStore, "testDashWithHiddenQuery", 1, 0, true, []map[string]interface{}{}, customPanels)
		dto := &SavePublicDashboardConfigDTO{
			DashboardUid: dashboard.Uid,
			OrgId:        dashboard.OrgId,
			UserId:       7,
			PublicDashboard: &PublicDashboard{
				IsEnabled:    true,
				DashboardUid: "NOTTHESAME",
				OrgId:        9999999,
				TimeSettings: timeSettings,
			},
		}
		pubdashDto, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)
		require.NoError(t, err)

		resp, _ := service.GetQueryDataResponse(context.Background(), true, publicDashboardQueryDTO, 1, pubdashDto.AccessToken)
		require.Nil(t, resp)
	})
}

func TestBuildMetricRequest(t *testing.T) {
	sqlStore := db.InitTestDB(t)
	dashboardStore := dashboardsDB.ProvideDashboardStore(sqlStore, sqlStore.Cfg, featuremgmt.WithFeatures(), tagimpl.ProvideService(sqlStore, sqlStore.Cfg))
	publicdashboardStore := database.ProvideStore(sqlStore)

	publicDashboard := insertTestDashboard(t, dashboardStore, "testDashie", 1, 0, true, []map[string]interface{}{}, nil)
	nonPublicDashboard := insertTestDashboard(t, dashboardStore, "testNonPublicDashie", 1, 0, true, []map[string]interface{}{}, nil)
	from, to := internal.GetTimeRangeFromDashboard(t, publicDashboard.Data)

	service := &PublicDashboardServiceImpl{
		log:                log.New("test.logger"),
		store:              publicdashboardStore,
		intervalCalculator: intervalv2.NewCalculator(),
	}

	publicDashboardQueryDTO := PublicDashboardQueryDTO{
		IntervalMs:    int64(10000000),
		MaxDataPoints: int64(200),
	}

	dto := &SavePublicDashboardConfigDTO{
		DashboardUid: publicDashboard.Uid,
		OrgId:        publicDashboard.OrgId,
		PublicDashboard: &PublicDashboard{
			IsEnabled:    true,
			DashboardUid: "NOTTHESAME",
			OrgId:        9999999,
			TimeSettings: timeSettings,
		},
	}

	publicDashboardPD, err := service.SavePublicDashboardConfig(context.Background(), SignedInUser, dto)
	require.NoError(t, err)

	nonPublicDto := &SavePublicDashboardConfigDTO{
		DashboardUid: nonPublicDashboard.Uid,
		OrgId:        nonPublicDashboard.OrgId,
		PublicDashboard: &PublicDashboard{
			IsEnabled:    false,
			DashboardUid: "NOTTHESAME",
			OrgId:        9999999,
			TimeSettings: defaultPubdashTimeSettings,
		},
	}

	_, err = service.SavePublicDashboardConfig(context.Background(), SignedInUser, nonPublicDto)
	require.NoError(t, err)

	t.Run("extracts queries from provided dashboard", func(t *testing.T) {
		reqDTO, err := service.buildMetricRequest(
			context.Background(),
			publicDashboard,
			publicDashboardPD,
			1,
			publicDashboardQueryDTO,
		)
		require.NoError(t, err)

		require.Equal(t, from, reqDTO.From)
		require.Equal(t, to, reqDTO.To)

		for i := range reqDTO.Queries {
			require.Equal(t, publicDashboardQueryDTO.IntervalMs, reqDTO.Queries[i].Get("intervalMs").MustInt64())
			require.Equal(t, publicDashboardQueryDTO.MaxDataPoints, reqDTO.Queries[i].Get("maxDataPoints").MustInt64())
		}

		require.Len(t, reqDTO.Queries, 2)

		require.Equal(
			t,
			simplejson.NewFromAny(map[string]interface{}{
				"datasource": map[string]interface{}{
					"type": "mysql",
					"uid":  "ds1",
				},
				"intervalMs":    int64(10000000),
				"maxDataPoints": int64(200),
				"refId":         "A",
			}),
			reqDTO.Queries[0],
		)

		require.Equal(
			t,
			simplejson.NewFromAny(map[string]interface{}{
				"datasource": map[string]interface{}{
					"type": "prometheus",
					"uid":  "ds2",
				},
				"intervalMs":    int64(10000000),
				"maxDataPoints": int64(200),
				"refId":         "B",
			}),
			reqDTO.Queries[1],
		)
	})

	t.Run("returns an error when panel missing", func(t *testing.T) {
		_, err := service.buildMetricRequest(
			context.Background(),
			publicDashboard,
			publicDashboardPD,
			49,
			publicDashboardQueryDTO,
		)

		require.ErrorContains(t, err, ErrPublicDashboardPanelNotFound.Reason)
	})

	t.Run("metric request built without hidden query", func(t *testing.T) {
		hiddenQuery := map[string]interface{}{
			"datasource": map[string]interface{}{
				"type": "mysql",
				"uid":  "ds1",
			},
			"hide":  true,
			"refId": "A",
		}
		nonHiddenQuery := map[string]interface{}{
			"datasource": map[string]interface{}{
				"type": "prometheus",
				"uid":  "ds2",
			},
			"refId": "B",
		}

		customPanels := []interface{}{
			map[string]interface{}{
				"id": 1,
				"datasource": map[string]interface{}{
					"uid": "ds1",
				},
				"targets": []interface{}{hiddenQuery, nonHiddenQuery},
			}}

		publicDashboard := insertTestDashboard(t, dashboardStore, "testDashWithHiddenQuery", 1, 0, true, []map[string]interface{}{}, customPanels)

		reqDTO, err := service.buildMetricRequest(
			context.Background(),
			publicDashboard,
			publicDashboardPD,
			1,
			publicDashboardQueryDTO,
		)
		require.NoError(t, err)

		require.Equal(t, from, reqDTO.From)
		require.Equal(t, to, reqDTO.To)

		for i := range reqDTO.Queries {
			require.Equal(t, publicDashboardQueryDTO.IntervalMs, reqDTO.Queries[i].Get("intervalMs").MustInt64())
			require.Equal(t, publicDashboardQueryDTO.MaxDataPoints, reqDTO.Queries[i].Get("maxDataPoints").MustInt64())
		}

		require.Len(t, reqDTO.Queries, 1)

		require.NotEqual(
			t,
			simplejson.NewFromAny(hiddenQuery),
			reqDTO.Queries[0],
		)

		require.Equal(
			t,
			simplejson.NewFromAny(nonHiddenQuery),
			reqDTO.Queries[0],
		)
	})

	t.Run("metric request built with 0 queries len when all queries are hidden", func(t *testing.T) {
		customPanels := []interface{}{
			map[string]interface{}{
				"id": 1,
				"datasource": map[string]interface{}{
					"uid": "ds1",
				},
				"targets": []interface{}{map[string]interface{}{
					"datasource": map[string]interface{}{
						"type": "mysql",
						"uid":  "ds1",
					},
					"hide":  true,
					"refId": "A",
				}, map[string]interface{}{
					"datasource": map[string]interface{}{
						"type": "prometheus",
						"uid":  "ds2",
					},
					"hide":  true,
					"refId": "B",
				}},
			}}

		publicDashboard := insertTestDashboard(t, dashboardStore, "testDashWithAllQueriesHidden", 1, 0, true, []map[string]interface{}{}, customPanels)

		reqDTO, err := service.buildMetricRequest(
			context.Background(),
			publicDashboard,
			publicDashboardPD,
			1,
			publicDashboardQueryDTO,
		)
		require.NoError(t, err)

		require.Equal(t, from, reqDTO.From)
		require.Equal(t, to, reqDTO.To)

		require.Len(t, reqDTO.Queries, 0)
	})
}

func insertTestDashboard(t *testing.T, dashboardStore *dashboardsDB.DashboardStore, title string, orgId int64,
	folderId int64, isFolder bool, templateVars []map[string]interface{}, customPanels []interface{}, tags ...interface{}) *models.Dashboard {
	t.Helper()

	var dashboardPanels []interface{}
	if customPanels != nil {
		dashboardPanels = customPanels
	} else {
		dashboardPanels = []interface{}{
			map[string]interface{}{
				"id": 1,
				"datasource": map[string]interface{}{
					"uid": "ds1",
				},
				"targets": []interface{}{
					map[string]interface{}{
						"datasource": map[string]interface{}{
							"type": "mysql",
							"uid":  "ds1",
						},
						"refId": "A",
					},
					map[string]interface{}{
						"datasource": map[string]interface{}{
							"type": "prometheus",
							"uid":  "ds2",
						},
						"refId": "B",
					},
				},
			},
			map[string]interface{}{
				"id": 2,
				"datasource": map[string]interface{}{
					"uid": "ds3",
				},
				"targets": []interface{}{
					map[string]interface{}{
						"datasource": map[string]interface{}{
							"type": "mysql",
							"uid":  "ds3",
						},
						"refId": "C",
					},
				},
			},
		}
	}

	cmd := models.SaveDashboardCommand{
		OrgId:    orgId,
		FolderId: folderId,
		IsFolder: isFolder,
		Dashboard: simplejson.NewFromAny(map[string]interface{}{
			"id":     nil,
			"title":  title,
			"tags":   tags,
			"panels": dashboardPanels,
			"templating": map[string]interface{}{
				"list": templateVars,
			},
			"time": map[string]interface{}{
				"from": "2022-09-01T00:00:00.000Z",
				"to":   "2022-09-01T12:00:00.000Z",
			},
		}),
	}
	dash, err := dashboardStore.SaveDashboard(context.Background(), cmd)
	require.NoError(t, err)
	require.NotNil(t, dash)
	dash.Data.Set("id", dash.Id)
	dash.Data.Set("uid", dash.Uid)
	return dash
}

func TestPublicDashboardServiceImpl_getSafeIntervalAndMaxDataPoints(t *testing.T) {
	type args struct {
		reqDTO PublicDashboardQueryDTO
		ts     TimeSettings
	}
	tests := []struct {
		name                  string
		args                  args
		wantSafeInterval      int64
		wantSafeMaxDataPoints int64
	}{
		{
			name: "return original interval",
			args: args{
				reqDTO: PublicDashboardQueryDTO{
					IntervalMs:    10000,
					MaxDataPoints: 300,
				},
				ts: TimeSettings{
					From: "now-3h",
					To:   "now",
				},
			},
			wantSafeInterval:      10000,
			wantSafeMaxDataPoints: 300,
		},
		{
			name: "return safe interval because of a small interval",
			args: args{
				reqDTO: PublicDashboardQueryDTO{
					IntervalMs:    1000,
					MaxDataPoints: 300,
				},
				ts: TimeSettings{
					From: "now-6h",
					To:   "now",
				},
			},
			wantSafeInterval:      2000,
			wantSafeMaxDataPoints: 11000,
		},
		{
			name: "return safe interval for long time range",
			args: args{
				reqDTO: PublicDashboardQueryDTO{
					IntervalMs:    100,
					MaxDataPoints: 300,
				},
				ts: TimeSettings{
					From: "now-90d",
					To:   "now",
				},
			},
			wantSafeInterval:      600000,
			wantSafeMaxDataPoints: 11000,
		},
		{
			name: "return safe interval when reqDTO is empty",
			args: args{
				reqDTO: PublicDashboardQueryDTO{},
				ts: TimeSettings{
					From: "now-90d",
					To:   "now",
				},
			},
			wantSafeInterval:      600000,
			wantSafeMaxDataPoints: 11000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PublicDashboardServiceImpl{
				intervalCalculator: intervalv2.NewCalculator(),
			}
			got, got1 := pd.getSafeIntervalAndMaxDataPoints(tt.args.reqDTO, tt.args.ts)
			assert.Equalf(t, tt.wantSafeInterval, got, "getSafeIntervalAndMaxDataPoints(%v, %v)", tt.args.reqDTO, tt.args.ts)
			assert.Equalf(t, tt.wantSafeMaxDataPoints, got1, "getSafeIntervalAndMaxDataPoints(%v, %v)", tt.args.reqDTO, tt.args.ts)
		})
	}
}

func TestDashboardEnabledChanged(t *testing.T) {
	t.Run("created isEnabled: false", func(t *testing.T) {
		assert.False(t, publicDashboardIsEnabledChanged(nil, &PublicDashboard{IsEnabled: false}))
	})

	t.Run("created isEnabled: true", func(t *testing.T) {
		assert.True(t, publicDashboardIsEnabledChanged(nil, &PublicDashboard{IsEnabled: true}))
	})

	t.Run("updated isEnabled same", func(t *testing.T) {
		assert.False(t, publicDashboardIsEnabledChanged(&PublicDashboard{IsEnabled: true}, &PublicDashboard{IsEnabled: true}))
	})

	t.Run("updated isEnabled changed", func(t *testing.T) {
		assert.True(t, publicDashboardIsEnabledChanged(&PublicDashboard{IsEnabled: false}, &PublicDashboard{IsEnabled: true}))
	})
}

func CreateDatasource(dsType string, uid string) struct {
	Type *string `json:"type,omitempty"`
	Uid  *string `json:"uid,omitempty"`
} {
	return struct {
		Type *string `json:"type,omitempty"`
		Uid  *string `json:"uid,omitempty"`
	}{
		Type: &dsType,
		Uid:  &uid,
	}
}

func AddAnnotationsToDashboard(t *testing.T, dash *models.Dashboard, annotations []DashAnnotation) *models.Dashboard {
	type annotationsDto struct {
		List []DashAnnotation `json:"list"`
	}
	annos := annotationsDto{}
	annos.List = annotations
	annoJSON, err := json.Marshal(annos)
	require.NoError(t, err)

	dashAnnos, err := simplejson.NewJson(annoJSON)
	require.NoError(t, err)

	dash.Data.Set("annotations", dashAnnos)

	return dash
}

func TestPublicDashboardServiceImpl_ListPublicDashboards(t *testing.T) {
	type args struct {
		ctx   context.Context
		u     *user.SignedInUser
		orgId int64
	}

	testCases := []struct {
		name         string
		args         args
		evaluateFunc func(c context.Context, u *user.SignedInUser, e accesscontrol.Evaluator) (bool, error)
		want         []PublicDashboardListResponse
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name: "should return empty list when user does not have permissions to read any dashboard",
			args: args{
				ctx:   context.Background(),
				u:     &user.SignedInUser{OrgID: 1},
				orgId: 1,
			},
			want:    []PublicDashboardListResponse{},
			wantErr: assert.NoError,
		},
		{
			name: "should return all dashboards when has permissions",
			args: args{
				ctx: context.Background(),
				u: &user.SignedInUser{OrgID: 1, Permissions: map[int64]map[string][]string{
					1: {"dashboards:read": {
						"dashboards:uid:0S6TmO67z", "dashboards:uid:1S6TmO67z", "dashboards:uid:2S6TmO67z", "dashboards:uid:9S6TmO67z",
					}}},
				},
				orgId: 1,
			},
			want: []PublicDashboardListResponse{
				{
					Uid:          "0GwW7mgVk",
					AccessToken:  "0b458cb7fe7f42c68712078bcacee6e3",
					DashboardUid: "0S6TmO67z",
					Title:        "my zero dashboard",
					IsEnabled:    true,
				},
				{
					Uid:          "1GwW7mgVk",
					AccessToken:  "1b458cb7fe7f42c68712078bcacee6e3",
					DashboardUid: "1S6TmO67z",
					Title:        "my first dashboard",
					IsEnabled:    true,
				},
				{
					Uid:          "2GwW7mgVk",
					AccessToken:  "2b458cb7fe7f42c68712078bcacee6e3",
					DashboardUid: "2S6TmO67z",
					Title:        "my second dashboard",
					IsEnabled:    false,
				},
				{
					Uid:          "9GwW7mgVk",
					AccessToken:  "deletedashboardaccesstoken",
					DashboardUid: "9S6TmO67z",
					Title:        "",
					IsEnabled:    true,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return only dashboards with permissions",
			args: args{
				ctx: context.Background(),
				u: &user.SignedInUser{OrgID: 1, Permissions: map[int64]map[string][]string{
					1: {"dashboards:read": {"dashboards:uid:0S6TmO67z"}}},
				},
				orgId: 1,
			},
			want: []PublicDashboardListResponse{
				{
					Uid:          "0GwW7mgVk",
					AccessToken:  "0b458cb7fe7f42c68712078bcacee6e3",
					DashboardUid: "0S6TmO67z",
					Title:        "my zero dashboard",
					IsEnabled:    true,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return orphaned public dashboards",
			args: args{
				ctx: context.Background(),
				u: &user.SignedInUser{OrgID: 1, Permissions: map[int64]map[string][]string{
					1: {"dashboards:read": {"dashboards:uid:0S6TmO67z"}}},
				},
				orgId: 1,
			},
			evaluateFunc: func(c context.Context, u *user.SignedInUser, e accesscontrol.Evaluator) (bool, error) {
				return false, dashboards.ErrDashboardNotFound
			},
			want: []PublicDashboardListResponse{
				{
					Uid:          "0GwW7mgVk",
					AccessToken:  "0b458cb7fe7f42c68712078bcacee6e3",
					DashboardUid: "0S6TmO67z",
					Title:        "my zero dashboard",
					IsEnabled:    true,
				},
				{
					Uid:          "1GwW7mgVk",
					AccessToken:  "1b458cb7fe7f42c68712078bcacee6e3",
					DashboardUid: "1S6TmO67z",
					Title:        "my first dashboard",
					IsEnabled:    true,
				},
				{
					Uid:          "2GwW7mgVk",
					AccessToken:  "2b458cb7fe7f42c68712078bcacee6e3",
					DashboardUid: "2S6TmO67z",
					Title:        "my second dashboard",
					IsEnabled:    false,
				},
				{
					Uid:          "9GwW7mgVk",
					AccessToken:  "deletedashboardaccesstoken",
					DashboardUid: "9S6TmO67z",
					Title:        "",
					IsEnabled:    true,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "errors different than not data found should be returned",
			args: args{
				ctx: context.Background(),
				u: &user.SignedInUser{OrgID: 1, Permissions: map[int64]map[string][]string{
					1: {"dashboards:read": {"dashboards:uid:0S6TmO67z"}}},
				},
				orgId: 1,
			},
			evaluateFunc: func(c context.Context, u *user.SignedInUser, e accesscontrol.Evaluator) (bool, error) {
				return false, dashboards.ErrDashboardCorrupt
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}

	mockedDashboards := []PublicDashboardListResponse{
		{
			Uid:          "0GwW7mgVk",
			AccessToken:  "0b458cb7fe7f42c68712078bcacee6e3",
			DashboardUid: "0S6TmO67z",
			Title:        "my zero dashboard",
			IsEnabled:    true,
		},
		{
			Uid:          "1GwW7mgVk",
			AccessToken:  "1b458cb7fe7f42c68712078bcacee6e3",
			DashboardUid: "1S6TmO67z",
			Title:        "my first dashboard",
			IsEnabled:    true,
		},
		{
			Uid:          "2GwW7mgVk",
			AccessToken:  "2b458cb7fe7f42c68712078bcacee6e3",
			DashboardUid: "2S6TmO67z",
			Title:        "my second dashboard",
			IsEnabled:    false,
		},
		{
			Uid:          "9GwW7mgVk",
			AccessToken:  "deletedashboardaccesstoken",
			DashboardUid: "9S6TmO67z",
			Title:        "",
			IsEnabled:    true,
		},
	}

	store := NewFakePublicDashboardStore(t)
	store.On("ListPublicDashboards", mock.Anything, mock.Anything).
		Return(mockedDashboards, nil)

	ac := tests.SetupMockAccesscontrol(t,
		func(c context.Context, siu *user.SignedInUser, _ accesscontrol.Options) ([]accesscontrol.Permission, error) {
			return []accesscontrol.Permission{}, nil
		},
		false,
	)

	pd := &PublicDashboardServiceImpl{
		log:   log.New("test.logger"),
		store: store,
		ac:    ac,
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ac.EvaluateFunc = tt.evaluateFunc

			got, err := pd.ListPublicDashboards(tt.args.ctx, tt.args.u, tt.args.orgId)
			if !tt.wantErr(t, err, fmt.Sprintf("ListPublicDashboards(%v, %v, %v)", tt.args.ctx, tt.args.u, tt.args.orgId)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ListPublicDashboards(%v, %v, %v)", tt.args.ctx, tt.args.u, tt.args.orgId)
		})
	}
}
