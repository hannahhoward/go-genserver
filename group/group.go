package group

import (
	"context"
	"fmt"

	"github.com/hannahhoward/go-genserver/genserver"
	"github.com/hannahhoward/go-genserver/mailbox"
	"github.com/hannahhoward/go-genserver/sync"
)

type Store[ID fmt.Stringer, State any] interface {
	Has(id ID) (bool, error)
	List() ([]State, error)
	CreateIfNotExist(id ID, state State) (bool, error)
	Mutator(id ID) genserver.StateMutator[State]
}

type Group[ID fmt.Stringer, State any] struct {
	kind         string
	messagesPool mailbox.Pool[genserver.MessageHandler[State]]
	store        Store[ID, State]
	genServers   sync.Map[ID, *genserver.GenServer[ID, State]]
}

func New[ID fmt.Stringer, State any](kind string, store Store[ID, State]) *Group[ID, State] {
	return &Group[ID, State]{
		messagesPool: sync.NewPool[mailbox.Message[genserver.MessageHandler[State]]](),
		kind:         kind,
		store:        store,
	}
}

// Begin initiates tracking with a specific value for a given identifier
func (g *Group[ID, State]) Begin(id ID, initialState State) error {

	_, exist := g.genServers.Load(id)
	if exist {
		return fmt.Errorf("Begin(%s): already tracking identifier `%s`", g.kind, id)
	}

	exists, err := g.store.CreateIfNotExist(id, initialState)
	if err != nil {
		return fmt.Errorf("Begin(%s): failed to check if state for %s exists: %w", g.kind, id, err)
	}

	if exists {
		return fmt.Errorf("Begin(%s): cannot initiate a state for identifier `%s` that already exists", g.kind, id)
	}

	_, err = g.loadOrCreateGenServer(id)
	if err != nil {
		return fmt.Errorf("Begin(%s): loadOrCreate state: %w", g.kind, err)
	}
	return nil
}

func (g *Group[ID, State]) loadOrCreateGenServer(id ID) (*genserver.GenServer[ID, State], error) {

	res := genserver.New(g.kind, id, g.store.Get(id), genserver.WithMessagePool[ID](g.messagesPool))

	res, loaded := g.genServers.LoadOrStore(id, res)
	if !loaded {
		res.Start()
	}
	return res, nil
}

// Stop stops all state machines in this group
func (g *Group[ID, State]) Stop(ctx context.Context) error {
	var err error
	g.genServers.Range(func(id ID, gs *genserver.GenServer[ID, State]) bool {
		err = genserver.Shutdown(gs, genserver.Normal, func(State, genserver.ShutdownReason) error {
			return nil
		}, ctx.Done())
		if err != nil {
			return false
		}
		return true
	})
	return err
}

// List outputs states of all state machines in this group
func (g *Group[ID, State]) List() ([]State, error) {
	return g.store.List()
}

// Get gets state for a single state machine
func (g *Group[ID, State]) Get(id ID) genserver.StateMutator[State] {
	return g.store.Get(id)
}

// Has indicates whether there is data for the given state machine
func (g *Group[ID, State]) Has(id ID) (bool, error) {
	_, exist := g.genServers.Load(id)
	if exist {
		return true, nil
	}
	return g.store.Has(id)
}

func Call[ID fmt.Stringer, State any, Message any, Return any](g *Group[ID, State], id ID, message Message, handler genserver.CallHandler[State, Message, Return]) (Return, error) {
	gs, exist := g.genServers.Load(id)

	if exist {
		return genserver.Call(gs, message, handler)
	}

	var initialState State
	_, err := g.store.CreateIfNotExist(id, initialState)
	var emptyReturn Return
	if err != nil {
		return emptyReturn, fmt.Errorf("Send(%s): failed to check if state for %s exists: %w", g.kind, id, err)
	}

	gs, err = g.loadOrCreateGenServer(id)
	if err != nil {
		return emptyReturn, fmt.Errorf("loadOrCreate state: %w", err)
	}

	return genserver.Call(gs, message, handler)
}

func Cast[ID fmt.Stringer, State any, Message any](g *Group[ID, State], id ID, message Message, handler genserver.CastHandler[State, Message]) error {
	gs, exist := g.genServers.Load(id)

	if exist {
		return genserver.Cast(gs, message, handler)
	}

	var initialState State
	_, err := g.store.CreateIfNotExist(id, initialState)
	if err != nil {
		return fmt.Errorf("Send(%s): failed to check if state for %s exists: %w", g.kind, id, err)
	}

	gs, err = g.loadOrCreateGenServer(id)
	if err != nil {
		return fmt.Errorf("loadOrCreate state: %w", err)
	}

	return genserver.Cast(gs, message, handler)
}
