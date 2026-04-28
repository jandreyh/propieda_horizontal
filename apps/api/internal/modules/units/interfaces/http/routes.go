package http

import (
	"io"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/units/application/usecases"
	"github.com/saas-ph/api/internal/modules/units/domain"
)

// Dependencies agrupa las dependencias que el orquestador (cmd/api)
// inyecta al modulo units al montar sus rutas.
//
// Logger es opcional (si es nil se descarta).
// UnitRepo, OwnerRepo, OccupancyRepo y PeopleRepo son obligatorios.
// Now es opcional; default time.Now.
type Dependencies struct {
	Logger        *slog.Logger
	UnitRepo      domain.UnitRepository
	OwnerRepo     domain.OwnerRepository
	OccupancyRepo domain.OccupancyRepository
	PeopleRepo    domain.PeopleByUnitRepository
	Now           func() time.Time
}

// Mount registra los endpoints del modulo en el router chi recibido.
//
// Endpoints:
//   - POST   /units
//   - GET    /units                          (filtro ?structure_id=...)
//   - GET    /units/{id}
//   - GET    /units/{id}/people              (CRITICO DoD)
//   - POST   /units/{id}/owners
//   - DELETE /units/{id}/owners/{ownerID}
//   - POST   /units/{id}/occupants
//   - DELETE /units/{id}/occupants/{occupancyID}
func Mount(r chi.Router, deps Dependencies) {
	logger := deps.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}

	h := &handlers{
		logger: logger,
		createUC: usecases.NewCreateUnitUseCase(usecases.CreateUnitDeps{
			Units: deps.UnitRepo,
			Now:   now,
		}),
		getUC: usecases.NewGetUnitUseCase(usecases.GetUnitDeps{
			Units: deps.UnitRepo,
		}),
		listUC: usecases.NewListUnitsUseCase(usecases.ListUnitsDeps{
			Units: deps.UnitRepo,
		}),
		addOwnerUC: usecases.NewAddOwnerToUnitUseCase(usecases.AddOwnerToUnitDeps{
			Units:  deps.UnitRepo,
			Owners: deps.OwnerRepo,
			Now:    now,
		}),
		termOwnerUC: usecases.NewTerminateOwnershipUseCase(usecases.TerminateOwnershipDeps{
			Owners: deps.OwnerRepo,
			Now:    now,
		}),
		addOccUC: usecases.NewAddOccupantToUnitUseCase(usecases.AddOccupantToUnitDeps{
			Units:     deps.UnitRepo,
			Occupants: deps.OccupancyRepo,
			Now:       now,
		}),
		moveOutUC: usecases.NewMoveOutOccupantUseCase(usecases.MoveOutOccupantDeps{
			Occupants: deps.OccupancyRepo,
			Now:       now,
		}),
		peopleUC: usecases.NewGetPeopleInUnitUseCase(usecases.GetPeopleInUnitDeps{
			People: deps.PeopleRepo,
		}),
	}

	r.Route("/units", func(ur chi.Router) {
		ur.Post("/", h.createUnit)
		ur.Get("/", h.listUnits)
		ur.Get("/{id}", h.getUnit)
		ur.Get("/{id}/people", h.peopleInUnit)

		ur.Post("/{id}/owners", h.addOwner)
		ur.Delete("/{id}/owners/{ownerID}", h.terminateOwner)

		ur.Post("/{id}/occupants", h.addOccupant)
		ur.Delete("/{id}/occupants/{occupancyID}", h.moveOutOccupant)
	})
}
