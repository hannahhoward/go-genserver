package memory_test

import (
	"strconv"
	"testing"

	"github.com/hannahhoward/go-genserver/store/memory"
	"github.com/stretchr/testify/require"
)

type PrintableInt int

func (p PrintableInt) String() string {
	return strconv.Itoa(int(p))
}

type value struct {
	i int
}

func TestThreadSafety(t *testing.T) {
	store := memory.NewStore[PrintableInt, value]()
	has, err := store.Has(PrintableInt(5))
	require.False(t, has)
	require.Nil(t, err)
	readDone := make(chan struct{}, 1)
	writeDone := make(chan struct{}, 1)
	ss := store.Get(PrintableInt(5))
	go func() {
		for i := 0; i < 100; i++ {
			val, err := ss.Get()
			if err == nil {
				t.Logf("read val: %d", val.i)
			}
		}
		readDone <- struct{}{}
	}()
	go func() {
		store.CreateIfNotExist(PrintableInt(5), value{5})
		for i := 0; i < 100; i++ {
			returnValFunc, err := ss.ModifyState(func(a *value) (func(), error) {
				a.i = i + 5
				return func() {
					t.Logf("write val: %d", i+5)
				}, nil
			})
			if err == nil {
				returnValFunc()
			}
		}
		writeDone <- struct{}{}
	}()
	<-writeDone
	<-readDone
}
