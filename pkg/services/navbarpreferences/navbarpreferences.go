package navbarpreferences

import (
	"context"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/setting"
)

func ProvideService(cfg *setting.Cfg, sqlStore *sqlstore.SQLStore, routeRegister routing.RouteRegister) *NavbarPreferencesService {
	l := &LibraryElementService{
		Cfg:           cfg,
		SQLStore:      sqlStore,
		RouteRegister: routeRegister,
		log:           log.New("navbarpreferences"),
	}
	l.registerAPIEndpoints()
	return l
}

// Service is a service for operating on navbar preferences.
type Service interface {
	GetNavbarPreferences(c context.Context, signedInUser *models.SignedInUser) ([]NavbarPreference, error)
}

// NavbarPreferencesService is the service for the navbar preferences.
type NavbarPreferencesService struct {
	Cfg           *setting.Cfg
	SQLStore      *sqlstore.SQLStore
	RouteRegister routing.RouteRegister
	log           log.Logger
}

// GetNavbarPreferences gets the navbar preferences for a user
func (l *NavbarPreferencesService) GetNavbarPreferences(c context.Context, signedInUser *models.SignedInUser) ([]NavbarPreference, error) {
	return l.GetNavbarPreferences(c, signedInUser)
}
