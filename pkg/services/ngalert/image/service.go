package image

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/grafana/pkg/components/imguploader"
	"github.com/grafana/grafana/pkg/services/dashboards"
	ngmodels "github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/services/ngalert/store"
	"github.com/grafana/grafana/pkg/services/rendering"
	"github.com/grafana/grafana/pkg/services/screenshot"
	"github.com/grafana/grafana/pkg/setting"
)

//go:generate mockgen -destination=mock.go -package=image github.com/grafana/grafana/pkg/services/ngalert/image ImageService
type ImageService interface {
	// NewImage returns a new image for the alert instance.
	NewImage(ctx context.Context, r *ngmodels.AlertRule, labels data.Labels) (*store.Image, error)
}

var (
	// ErrNoDashboard is returned when the alert rule does not have a dashboard.
	ErrNoDashboard = errors.New("no dashboard")

	// ErrNoPanel is returned when the alert rule does not have a panel in a dashboard.
	ErrNoPanel = errors.New("no panel")
)

const (
	screenshotTimeout  = 10 * time.Second
	screenshotCacheTTL = 15 * time.Second
)

// ScreenshotImageService takes screenshots of the panel for an alert rule and
// saves the image in the store. The image contains a unique token that can be
// passed as an annotation or label to the Alertmanager.
type ScreenshotImageService struct {
	screenshots screenshot.ScreenshotService
	store       store.ImageStore
}

func NewScreenshotImageService(screenshots screenshot.ScreenshotService, store store.ImageStore) ImageService {
	return &ScreenshotImageService{
		screenshots: screenshots,
		store:       store,
	}
}

// NewScreenshotImageServiceFromCfg returns a new ScreenshotImageService
// from the configuration.
func NewScreenshotImageServiceFromCfg(cfg *setting.Cfg, r prometheus.Registerer,
	db *store.DBstore, ds dashboards.DashboardService, rs rendering.Service) (ImageService, error) {
	if !cfg.UnifiedAlerting.Screenshots.Enabled {
		return &ScreenshotImageService{
			screenshots: &screenshot.ScreenshotsUnavailableService{},
		}, nil
	}

	s := screenshot.NewBrowserScreenshotService(r, ds, rs)
	if cfg.UnifiedAlerting.Screenshots.UploadExternalImageStorage {
		u, err := imguploader.NewImageUploader()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize uploading screenshot service: %w", err)
		}
		s = screenshot.NewUploadingScreenshotService(r, s, u)
	}
	s = screenshot.NewRateLimitScreenshotService(s, cfg.UnifiedAlerting.Screenshots.MaxConcurrentScreenshots)
	s = screenshot.NewSingleFlightScreenshotService(s)
	s = screenshot.NewCachableScreenshotService(r, screenshotCacheTTL, s)
	s = screenshot.NewObservableScreenshotService(r, s)

	return &ScreenshotImageService{
		store:       db,
		screenshots: s,
	}, nil
}

// NewImage returns a screenshot of the panel for the alert rule. It returns
// ErrNoDashboard if the alert rule does not have a dashboard and ErrNoPanel
// when the alert rule does not have a panel in a dashboard.
func (s *ScreenshotImageService) NewImage(ctx context.Context, r *ngmodels.AlertRule, _ data.Labels) (*store.Image, error) {
	if r.DashboardUID == nil {
		return nil, ErrNoDashboard
	}
	if r.PanelID == nil {
		return nil, ErrNoPanel
	}

	screenshot, err := s.screenshots.Take(ctx, screenshot.ScreenshotOptions{
		Timeout:      screenshotTimeout,
		DashboardUID: *r.DashboardUID,
		PanelID:      *r.PanelID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to take screenshot: %w", err)
	}

	v := store.Image{
		Path: screenshot.Path,
		URL:  screenshot.URL,
	}
	if err := s.store.Save(ctx, &v); err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	return &v, nil
}

type NoopImageService struct{}

func (s *NoopImageService) NewImage(ctx context.Context, r *ngmodels.AlertRule, _ data.Labels) (*store.Image, error) {
	return &store.Image{}, nil
}
