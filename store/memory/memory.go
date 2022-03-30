package memory

import (
	"fmt"

	"github.com/hannahhoward/go-genserver/genserver"
	"github.com/hannahhoward/go-genserver/sync"
)

type Store[ID fmt.Stringer, State any] struct {
	store sync.Map[ID, *State]
}

func NewStore[ID fmt.Stringer, State any]() *Store[ID, State] {
	return &Store[ID, State]{}
}

func (s *Store[ID, State]) Has(id ID) (bool, error) {
	_, has := s.store.Load(id)
	return has, nil
}

func (s *Store[ID, State]) List() ([]State, error) {
	var list []State
	s.store.Range(func(_ ID, value *State) bool {
		list = append(list, *value)
		return true
	})
	return list, nil
}

func (s *Store[ID, State]) CreateIfNotExist(id ID, state State) (bool, error) {
	_, exists := s.store.LoadOrStore(id, &state)
	return exists, nil
}

func (s *Store[ID, State]) Get(id ID) (State, error) {
	return &storedState[ID, State]{&s.store, id}
}

type storedState[ID fmt.Stringer, State any] struct {
	store *sync.Map[ID, *State]
	id    ID
}

func (ss *storedState[ID, State]) Get() (State, error) {
	val, exists := ss.store.Load(ss.id)
	if !exists {
		var zeroState State
		return zeroState, fmt.Errorf("Could not load state for ID %s", ss.id)
	}
	return *val, nil
}

func (ss *storedState[ID, State]) ModifyState(modifier genserver.StateMutatorFn[State]) (func(), error) {
	val, exists := ss.store.Load(ss.id)
	if !exists {
		return nil, fmt.Errorf("Could not load state for ID %s", ss.id)
	}
	return modifier(val)
}
