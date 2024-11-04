package event

import (
	"context"
	"errors"
	"github.com/emortalmc/mono-services/services/party-manager/internal/app/party"
	"github.com/emortalmc/mono-services/services/party-manager/internal/kafka/writer"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"sync"
	"time"
)

type handler struct {
	log      *zap.SugaredLogger
	repo     *repository.MongoRepository // this is a horrible cross cutting concern but we'll cope for now
	notif    *writer.Notifier
	partySvc *party.Service
}

func NewScheduler(ctx context.Context, wg *sync.WaitGroup, log *zap.SugaredLogger, repo *repository.MongoRepository, notif *writer.Notifier,
	partySvc *party.Service) {
	h := &handler{
		log:      log,
		repo:     repo,
		notif:    notif,
		partySvc: partySvc,
	}

	h.run(ctx, wg)
}

func (h *handler) run(ctx context.Context, wg *sync.WaitGroup) {
	go func() {
		ticker := time.NewTicker(3 * time.Second)

		for {
			select {
			case <-ctx.Done():
				wg.Done()
				return
			case <-ticker.C:
				h.runDisplayEvent(ctx)
				h.runStartEvent(ctx)
			}
		}
	}()
}

func (h *handler) runDisplayEvent(ctx context.Context) {
	e, err := h.repo.GetEventToDisplay(ctx)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return
		}

		h.log.Errorf("failed to get event to display: %v", err)
		return
	}

	h.notif.DisplayEvent(ctx, e)
}

func (h *handler) runStartEvent(ctx context.Context) {
	e, err := h.repo.GetEventToStart(ctx)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return
		}

		h.log.Errorf("failed to get event to start: %v", err)
		return
	}

	// We don't convert a party as it's too complex. Instead, we re-create it.
	p, err := h.repo.GetPartyByMemberID(ctx, e.OwnerID)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		h.log.Errorf("failed to get party by member id: %v", err)
		return
	}

	if err == nil {
		if p.LeaderId == e.OwnerID {
			if err := h.partySvc.EmptyParty(ctx, p, true); err != nil {
				h.log.Errorf("failed to empty party: %v", err)
				return
			}
		} else {
			// note: p gets reassigned to the player's new party
			if p, err = h.partySvc.RemovePlayerFromParty(ctx, e.OwnerID, true); err != nil {
				h.log.Errorf("failed to remove player from party: %v", err)
				return
			}

			if err := h.partySvc.SetOpenParty(ctx, p.ID, true); err != nil {
				h.log.Errorf("failed to set open party: %v", err)
				return
			}
		}
	} else {
		p = model.NewParty(e.OwnerID, e.OwnerUsername)
		p.Open = true
		if err := h.repo.CreateParty(ctx, p); err != nil {
			h.log.Errorf("failed to create party: %v", err)
			return
		}

		// todo what is the impact of notifying this? could it be bad?
		h.notif.PartyCreated(ctx, p)
	}

	// We now have a party that is open with only the owner of the event in it.

	if err := h.repo.SetEventPartyID(ctx, e.ID, p.ID); err != nil {
		h.log.Errorf("failed to set event party id: %v", err)
		return
	}

	if err := h.repo.SetPartyEventID(ctx, p.ID, e.ID); err != nil {
		h.log.Errorf("failed to set party event id: %v", err)
		return
	}

	e.PartyID = &p.ID

	h.notif.StartEvent(ctx, e)
}
