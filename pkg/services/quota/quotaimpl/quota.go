package quotaimpl

import (
	"context"
	"fmt"
	"sync"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/events"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/quota"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/setting"
	"golang.org/x/sync/errgroup"
)

type Service struct {
	store  store
	Cfg    *setting.Cfg
	Logger log.Logger

	mutex     sync.RWMutex
	reporters map[quota.TargetSrv]quota.UsageReporterFunc

	defaultLimits quota.Limits
}

type ServiceDisabled struct{}

func (s *ServiceDisabled) QuotaReached(c *models.ReqContext, target string) (bool, error) {
	return false, fmt.Errorf("Quotas not enabled")
}

func (s *ServiceDisabled) Get(ctx context.Context, scope string, id int64) ([]quota.QuotaDTO, error) {
	return nil, fmt.Errorf("Quotas not enabled")
}

func (s *ServiceDisabled) Update(ctx context.Context, cmd *quota.UpdateQuotaCmd) error {
	return fmt.Errorf("Quotas not enabled")
}

// CheckQuotaReached check that quota is reached for a target. If ScopeParameters are not defined, only global scope is checked
func (s *ServiceDisabled) CheckQuotaReached(ctx context.Context, target string, scopeParams *quota.ScopeParameters) (bool, error) {
	return false, fmt.Errorf("Quotas not enabled")
}

func (s *ServiceDisabled) DeleteByUser(ctx context.Context, userID int64) error {
	return fmt.Errorf("Quotas not enabled")
}

func ProvideService(db db.DB, cfg *setting.Cfg, ss *sqlstore.SQLStore, bus bus.Bus) quota.Service {
	s := Service{
		store:     &sqlStore{db: db},
		Cfg:       cfg,
		Logger:    log.New("quota_service"),
		reporters: make(map[quota.TargetSrv]quota.UsageReporterFunc),
	}

	if s.IsDisabled() {
		return &ServiceDisabled{}
	}

	s.loadQuotaSettings()

	bus.AddEventListener(s.addReporter)

	return &s
}

func (s *Service) IsDisabled() bool {
	quotaSection := s.Cfg.Raw.Section("quota")
	return !quotaSection.Key("enabled").MustBool(false)
}

// QuotaReached checks that quota is reached for a target. Runs CheckQuotaReached and take context and scope parameters from the request context
func (s *Service) QuotaReached(c *models.ReqContext, target string) (bool, error) {
	// No request context means this is a background service, like LDAP Background Sync
	if c == nil {
		return false, nil
	}

	var params *quota.ScopeParameters
	if c.IsSignedIn {
		params = &quota.ScopeParameters{
			OrgID:  c.OrgID,
			UserID: c.UserID,
		}
	}
	return s.CheckQuotaReached(c.Req.Context(), target, params)
}

func (s *Service) Get(ctx context.Context, scope string, id int64) ([]quota.QuotaDTO, error) {
	quotaScope := quota.Scope(scope)
	if err := quotaScope.Validate(); err != nil {
		return nil, err
	}

	q := make([]quota.QuotaDTO, 0)

	scopeParams := quota.ScopeParameters{}
	if quotaScope == quota.OrgScope {
		scopeParams.OrgID = id
	} else if quotaScope == quota.UserScope {
		scopeParams.UserID = id
	}

	customLimits, err := s.store.Get(ctx, &scopeParams)
	if err != nil {
		return nil, err
	}

	u, err := s.getUsage(ctx, &scopeParams)
	if err != nil {
		return nil, err
	}

	for target, targetDefaultLimits := range s.defaultLimits {
		scopeTargetDefaultLimit, ok := targetDefaultLimits[quota.Scope(scope)]
		if !ok {
			return []quota.QuotaDTO{}, quota.ErrInvalidQuotaScope
		}
		limit := scopeTargetDefaultLimit

		if targetCustomLimits, ok := customLimits[quota.TargetSrv(target)]; ok {
			if scopeTargetCustomLimit, ok := targetCustomLimits[quota.Scope(scope)]; ok {
				limit = scopeTargetCustomLimit
			}
		}

		q = append(q, quota.QuotaDTO{
			Target: string(target),
			Limit:  limit,
			OrgId:  scopeParams.OrgID,
			UserId: scopeParams.UserID,
			Used:   u.Get(target, quota.Scope(scope)),
		})
	}

	return q, nil
}

func (s *Service) Update(ctx context.Context, cmd *quota.UpdateQuotaCmd) error {
	return s.store.Update(ctx, cmd)
}

// CheckQuotaReached check that quota is reached for a target. If ScopeParameters are not defined, only global scope is checked
func (s *Service) CheckQuotaReached(ctx context.Context, target string, scopeParams *quota.ScopeParameters) (bool, error) {
	targetLimits, err := s.getOverridenLimits(ctx, target, scopeParams)
	if err != nil {
		return false, err
	}

	usageReporterFunc, ok := s.getReporter(quota.TargetSrv(target))
	if !ok {
		return false, quota.ErrInvalidQuotaTarget
	}
	targetUsage, err := usageReporterFunc(ctx, scopeParams)
	if err != nil {
		return false, err
	}

	for scp, limit := range targetLimits {
		switch {
		case limit < 0:
			continue
		case limit == 0:
			return true, nil
		default:
			u, ok := targetUsage[scp]
			if !ok {
				return false, fmt.Errorf("no usage for target:%s scope:%s", target, scp)
			}
			if u >= limit {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *Service) DeleteByUser(ctx context.Context, userID int64) error {
	return s.store.DeleteByUser(ctx, userID)
}

func (s *Service) addReporter(_ context.Context, e *events.NewQuotaReporter) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// TODO check for conflicts
	s.reporters[e.TargetSrv] = e.Reporter

	return nil
}

func (s *Service) getReporter(target quota.TargetSrv) (quota.UsageReporterFunc, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	r, ok := s.reporters[target]
	return r, ok
}

type reporter struct {
	target       quota.TargetSrv
	reporterFunc quota.UsageReporterFunc
}

func (s *Service) getReporters() <-chan reporter {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	ch := make(chan reporter)
	go func() {
		defer close(ch)
		for t, r := range s.reporters {
			ch <- reporter{target: t, reporterFunc: r}
		}
	}()

	return ch
}

func (s *Service) getOverridenLimits(ctx context.Context, target string, scopeParams *quota.ScopeParameters) (map[quota.Scope]int64, error) {
	targetLimits, ok := s.defaultLimits[quota.TargetSrv(target)]
	if !ok {
		return nil, quota.ErrInvalidQuotaTarget
	}

	customLimits, err := s.store.Get(ctx, scopeParams)
	if err != nil {
		return nil, err
	}

	targetCustomLimits, ok := customLimits[quota.TargetSrv(target)]
	if ok {
		for scp := range targetLimits {
			if limit, ok := targetCustomLimits[scp]; ok {
				targetLimits[scp] = limit
			}
		}

	}

	return targetLimits, nil
}

func (s *Service) getUsage(ctx context.Context, scopeParams *quota.ScopeParameters) (*quota.Usage, error) {
	usage := &quota.Usage{}
	g, ctx := errgroup.WithContext(ctx)

	for r := range s.getReporters() {
		targetSrv := r.target
		r := r
		g.Go(func() error {
			u, err := r.reporterFunc(ctx, scopeParams)
			if err != nil {
				return err
			}
			usage.Add(targetSrv, u)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return usage, nil
}
