package mailbox

import (
	"sync"
)

type Mailbox[MessageType comparable] struct {
	head        *Message[MessageType]
	tail        *Message[MessageType]
	messagePool Pool[MessageType]
	open        bool
	lock        *sync.Mutex
	signal      *sync.Cond
}

type Pool[MessageType comparable] interface {
	Put(*Message[MessageType])
	Get() *Message[MessageType]
}

type Message[MessageType comparable] struct {
	message MessageType
	next    *Message[MessageType]
}

func NewMailbox[MessageType comparable](messagePool Pool[MessageType]) *Mailbox[MessageType] {
	lock := &sync.Mutex{}
	return &Mailbox[MessageType]{
		messagePool: messagePool,
		open:        true,
		signal:      sync.NewCond(lock),
		lock:        lock,
	}
}

func (mb *Mailbox[MessageType]) Send(message MessageType) bool {
	var emptyMessage MessageType
	if message == emptyMessage {
		return false
	}

	mb.lock.Lock()
	defer mb.lock.Unlock()

	if !mb.open {
		return false
	}
	newMailboxMessage := mb.messagePool.Get()
	newMailboxMessage.message = message
	if mb.head == nil {
		mb.tail = newMailboxMessage
		mb.head = mb.tail
	} else {
		mb.tail.next = newMailboxMessage
		mb.tail = mb.tail.next
	}
	mb.signal.Signal()
	return true
}

func (mb *Mailbox[MessageType]) Receive() (bool, MessageType) {
	mb.lock.Lock()
	defer mb.lock.Unlock()

	for {
		// Case 1 message is readily waiting
		if mb.head != nil {
			message := mb.head.message
			next := mb.head.next
			mb.messagePool.Put(mb.head)
			mb.head = next
			return true, message
		}

		var itemZero MessageType
		// Case 2 no message and the channel is closed
		if !mb.open {
			return false, itemZero
		}

		// Case 3 message not closed but nothing available yet
		mb.signal.Wait()
	}
}

func (mb *Mailbox[MessageType]) Close() {
	mb.lock.Lock()
	defer mb.lock.Unlock()

	if mb.open {
		mb.open = false
		for {
			if mb.head == nil {
				break
			}
			next := mb.head.next
			mb.messagePool.Put(mb.head)
			mb.head = next
		}
		mb.signal.Signal()
	}
}
