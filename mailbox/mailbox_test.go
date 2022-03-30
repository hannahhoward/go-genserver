package mailbox_test

import (
	"testing"

	"github.com/hannahhoward/go-genserver/mailbox"
	"github.com/hannahhoward/go-genserver/sync"
	"github.com/stretchr/testify/require"
)

type handler struct{}

func (h handler) Handle(s *int) (func(), error) {
	return nil, nil
}

func TestMessageChannel(t *testing.T) {

	channel := mailbox.NewMailbox[int](sync.NewPool[mailbox.Message[int]]())
	counter := 0
	signal := make(chan struct{}, 10)
	closed := false
	data := 5

	go func() {
		for {
			more, _ := channel.Receive()
			if !more {
				break
			}
			counter++
			signal <- struct{}{}
		}
		closed = true
		signal <- struct{}{}
	}()

	require.True(t, channel.Send(data))
	<-signal
	require.Equal(t, counter, 1)
	require.True(t, channel.Send(data))
	<-signal
	require.Equal(t, counter, 2)
	for i := 0; i < 10; i++ {
		require.True(t, channel.Send(data))
	}
	for i := 0; i < 10; i++ {
		<-signal
	}
	require.Equal(t, counter, 12)
	channel.Close()
	<-signal
	require.True(t, closed)
	require.False(t, channel.Send(data))
}
