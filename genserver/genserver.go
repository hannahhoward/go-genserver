package genserver

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/hannahhoward/go-genserver/mailbox"
	"github.com/hannahhoward/go-genserver/sync"
)

type CallHandler[State any, Message any, Return any] func(*State, Message) (Return, error)
type CastHandler[State any, Message any] func(*State, Message) error
type ShutdownHandler[State any] func(State, ShutdownReason) error
type ShutdownReason uint64

const (
	Normal ShutdownReason = iota

	BrutalKill
)

type MessageHandler[State any] interface {
	Handle(s *State) (func(), error)
}

type CallMessageHandler[State any, Message any, Return any] struct {
	m          Message
	returnChan chan<- Return
	h          CallHandler[State, Message, Return]
}

type MessageReturner[Return any] struct {
	r          Return
	returnChan chan<- Return
}

func (mr MessageReturner[Return]) Return() {
	if mr.returnChan != nil {
		mr.returnChan <- mr.r
	}
}

func (m CallMessageHandler[State, Message, Return]) Handle(s *State) (func(), error) {
	r, err := m.h(s, m.m)
	return MessageReturner[Return]{r, m.returnChan}.Return, err
}

type CastMessageHandler[State any, Message any] struct {
	m Message
	h CastHandler[State, Message]
}

func (m CastMessageHandler[State, Message]) Handle(s *State) (func(), error) {
	err := m.h(s, m.m)
	return func() {}, err
}

type ShutdownMessageHandler[State any] struct {
	r ShutdownReason
	h ShutdownHandler[State]
}

func (m ShutdownMessageHandler[State]) Handle(s *State) (func(), error) {
	err := m.h(*s, m.r)
	return func() {}, err
}

type StateMutatorFn[State any] func(s *State) (func(), error)

type StateMutator[State any] func(StateMutatorFn[State]) (func(), error)

type genServerConfig[ID fmt.Stringer, State any] struct {
	deadlockTimeout  time.Duration
	deadlockCallback func(server *GenServer[ID, State], trace string)
	messagesPool     mailbox.Pool[MessageHandler[State]]
	logger           *log.Logger
}

// GenServer structure
type GenServer[ID fmt.Stringer, State any] struct {
	kind         string
	id           ID
	stateMutator StateMutator[State]
	messages     *mailbox.Mailbox[MessageHandler[State]]
	terminated   chan struct{}
	config       *genServerConfig[ID, State]
}

const defaultDeadlockTimeout = 30 * time.Second

type CallTimeoutError[ID fmt.Stringer] struct {
	kind  string
	id    ID
	trace string
}

func (cte CallTimeoutError[ID]) Error() string {
	if len(cte.trace) > 0 {
		return fmt.Sprintf("GenServer WARNING timeout in %s, id %s\nGenServer stuck in\n%s\n", cte.kind, cte.id, cte.trace)
	} else {
		buf := make([]byte, 100000)
		length := runtime.Stack(buf, false)
		trace := string(buf[:length])
		return fmt.Sprintf("GenServer WARNING timeout in %s, id %s\nGenServer couldn't find Server stacktrace\nClient Stacktrace:\n%s\n", cte.kind, cte.id, trace)
	}
}

// ExceptionHandler receives a State and an error, and can handle the error and continue
// operation by setting the error to nil
type ExceptionHandler[State any] func(State, error) error

type Option[ID fmt.Stringer, State any] func(gsConfig *genServerConfig[ID, State])

func WithMessagePool[ID fmt.Stringer, State any](messagePool mailbox.Pool[MessageHandler[State]]) Option[ID, State] {
	return func(gsConfig *genServerConfig[ID, State]) {
		gsConfig.messagesPool = messagePool
	}
}

// New creates a new genserver
// Assign the Terminate function to define a callback just before the worker stops
func New[ID fmt.Stringer, State any](kind string, id ID, stateMutator StateMutator[State], options ...Option[ID, State]) *GenServer[ID, State] {
	config := &genServerConfig[ID, State]{
		deadlockTimeout: defaultDeadlockTimeout,
		logger:          log.Default(),
	}
	for _, option := range options {
		option(config)
	}

	if config.messagesPool == nil {
		config.messagesPool = sync.NewPool[mailbox.Message[MessageHandler[State]]]()
	}

	server := &GenServer[ID, State]{
		kind:         kind,
		id:           id,
		stateMutator: stateMutator,
		messages:     mailbox.NewMailbox(config.messagesPool),
		terminated:   make(chan struct{}),
		config:       config,
	}
	return server
}

// Spawn is like new but automatically starts the server
func Spawn[ID fmt.Stringer, State any](kind string, id ID, state StateMutator[State], options ...Option[ID, State]) *GenServer[ID, State] {
	server := New(kind, id, state, options...)
	server.Start()
	return server
}

func (server *GenServer[ID, State]) Start() {
	go server.loop()
}

func (server *GenServer[ID, State]) ID() ID {
	return server.id
}

func (server *GenServer[ID, State]) loop() {
	defer func() {
		close(server.terminated)
	}()
	for {
		more, messageHandler := server.messages.Receive()
		if !more {
			return
		}
		_, isShutdown := messageHandler.(ShutdownMessageHandler[State])
		if isShutdown {
			server.messages.Close()
		}
		returnValue, err := server.stateMutator(messageHandler.Handle)
		if err != nil {
			server.config.logger.Printf("Processing message: %s", err)
			server.messages.Close()
		}
		returnValue()
	}
}

// Shutdown sends a shutdown signal to the server.
func Shutdown[ID fmt.Stringer, State any](server *GenServer[ID, State], reason ShutdownReason, handler ShutdownHandler[State], waitUntil <-chan struct{}) error {
	if !server.messages.Send(ShutdownMessageHandler[State]{reason, handler}) {
		return fmt.Errorf("send to dead genserver")
	}
	select {
	case <-waitUntil:
		return errors.New("did not finish shutting down")
	case <-server.terminated:
		return nil
	}
}

// Send sends a message to the server
func Cast[ID fmt.Stringer, State any, Message any](server *GenServer[ID, State], message Message, handler CastHandler[State, Message]) error {
	if !server.messages.Send(CastMessageHandler[State, Message]{message, handler}) {
		return fmt.Errorf("send to dead genserver")
	}
	return nil
}

func Call[ID fmt.Stringer, State any, Message any, Return any](server *GenServer[ID, State], message Message, handler CallHandler[State, Message, Return]) (Return, error) {
	timer := time.NewTimer(server.config.deadlockTimeout)
	// timer.Stop() see here for details on why
	// https://medium.com/@oboturov/golang-time-after-is-not-garbage-collected-4cbc94740082
	defer timer.Stop()

	returnValChan := make(chan Return, 1)
	var empty Return

	// Step 1 submitting message
	if !server.messages.Send(CallMessageHandler[State, Message, Return]{message, returnValChan, handler}) {
		return empty, fmt.Errorf("call to dead genserver")
	}

	// Step 2 waiting for message to finish
	select {
	case <-timer.C:
		return empty, server.handleTimeout()

	case returnVal := <-returnValChan:
		return returnVal, nil
	}
}

type getMessage struct{}

func (server *GenServer[ID, State]) Get() (State, error) {
	return Call(server, getMessage{}, func(s *State, gm getMessage) (State, error) {
		return *s, nil
	})
}

func (server *GenServer[ID, State]) handleTimeout() error {
	buf := make([]byte, 100000)
	length := len(buf)
	for length == len(buf) {
		buf = make([]byte, len(buf)*2)
		length = runtime.Stack(buf, true)
	}
	traces := strings.Split(string(buf[:length]), "\n\n")
	prefix := fmt.Sprintf("server id %v ", server.id)
	var trace string
	for _, t := range traces {
		if strings.HasPrefix(t, prefix) {
			trace = t
			break
		}
	}

	if cb := server.config.deadlockCallback; cb != nil {
		cb(server, trace)
	}
	return CallTimeoutError[ID]{server.kind, server.id, trace}
}
