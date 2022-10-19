package quota

import (
	"errors"
	"sync"
)

var ErrInvalidQuotaTarget = errors.New("invalid quota target")
var ErrInvalidQuotaScope = errors.New("invalid quota scope")

type ScopeParameters struct {
	OrgID  int64
	UserID int64
}

type Scope string

const (
	GlobalScope Scope = "global"
	OrgScope    Scope = "org"
	UserScope   Scope = "user"
)

func (s Scope) Validate() error {
	switch s {
	case GlobalScope, OrgScope, UserScope:
		return nil
	default:
		return ErrInvalidQuotaScope
	}
}

type QuotaScope struct {
	Name         Scope
	Target       string
	DefaultLimit int64
}

type TargetSrv string

const (
	OrgTarget        TargetSrv = "org"
	UserTarget       TargetSrv = "user"
	DataSourceTarget TargetSrv = "data_source"
	DashboardTarget  TargetSrv = "dashboard"
	ApiKeyTarget     TargetSrv = "api_key"
	SessionTarget    TargetSrv = "session"
	AlertRuleTarget  TargetSrv = "alert_rule"
	FileTarget       TargetSrv = "file"
)

func (s TargetSrv) Validate() error {
	switch s {
	case OrgTarget, UserTarget, DataSourceTarget, DashboardTarget, ApiKeyTarget, SessionTarget, AlertRuleTarget, FileTarget:
		return nil
	default:
		return ErrInvalidQuotaTarget
	}
}

type Limits map[TargetSrv]map[Scope]int64

type Usage struct {
	mutex sync.RWMutex
	m     map[TargetSrv]map[Scope]int64
}

func (u *Usage) Add(k TargetSrv, v map[Scope]int64) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.m[k] = v
}

func (u *Usage) Get(target TargetSrv, scope Scope) int64 {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	var used int64 = 0
	if tm, ok := u.m[target]; ok {
		if sm, ok := tm[scope]; ok {
			used = sm
		}
	}
	return used
}

type QuotaDTO struct {
	OrgId  int64  `json:"org_id,omitempty"`
	UserId int64  `json:"user_id,omitempty"`
	Target string `json:"target"`
	Limit  int64  `json:"limit"`
	Used   int64  `json:"used"`
}

type UpdateQuotaCmd struct {
	Target string `json:"target"`
	Limit  int64  `json:"limit"`
	OrgId  int64  `json:"-"`
	UserId int64  `json:"-"`
}
