package event

import (
	"context"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

var (
	_ ReadWriter = &repository.MongoRepository{}
)

type Reader interface {
	ListEvents(ctx context.Context) ([]*model.Event, error)
}

type Writer interface {
	CreateEvent(ctx context.Context, event *model.Event) error
	UpdateEvent(ctx context.Context, eventId string, displayTime *time.Time, startTime *time.Time) (*model.Event, error)
	DeleteEvent(ctx context.Context, eventId string) error
	GetLiveEvent(ctx context.Context) (*model.Event, error)
	GetEventByID(ctx context.Context, eventId string) (*model.Event, error)
	SetEventPartyID(ctx context.Context, eventId string, partyId primitive.ObjectID) error

	// Both GetEventToDisplay and GetEventToStart find an event where startTime/displayTime is after now
	// and then marks the event as displayed/started.

	GetEventToDisplay(ctx context.Context) (*model.Event, error)
	GetEventToStart(ctx context.Context) (*model.Event, error)
}

type ReadWriter interface {
	Reader
	Writer
}
