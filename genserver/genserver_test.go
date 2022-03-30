package genserver_test

import (
	"strconv"
	"testing"

	"github.com/hannahhoward/go-genserver/genserver"
)

type counter struct {
	current uint64
}

func add(c *counter, amt uint64) (uint64, error) {
	c.current = c.current + amt
	return c.current, nil
}

func subtract(c *counter, amt uint64) (uint64, error) {
	c.current = c.current - amt
	return c.current, nil
}

type simpleAccessor struct {
	c *counter
}

func (s *simpleAccessor) ModifyState(modifier genserver.StateMutatorFn[counter]) (func(), error) {
	f, err := modifier(s.c)
	return f, err
}

type PrintableInt int

func (p PrintableInt) String() string {
	return strconv.Itoa(int(p))
}

func TestGenServer(t *testing.T) {
	sa := &simpleAccessor{&counter{0}}
	genServer := genserver.Spawn[PrintableInt]("counter", 1, sa.ModifyState)

	current, err := genserver.Call(genServer, 1, add)
	if err != nil {
		t.Fatalf("should have called genServer successfully")
	}
	if current != 1 {
		t.Fatalf("did not add properly")
	}

	current, err = genserver.Call(genServer, 1, add)
	if err != nil {
		t.Fatalf("should have called genServer successfully")
	}
	if current != 2 {
		t.Fatalf("did not add properly")
	}

	current, err = genserver.Call(genServer, 1, subtract)
	if err != nil {
		t.Fatalf("should have called genServer successfully")
	}
	if current != 1 {
		t.Fatalf("did not subtract properly")
	}

	current, err = genserver.Call(genServer, 1, subtract)
	if err != nil {
		t.Fatalf("should have called genServer successfully")
	}
	if current != 0 {
		t.Fatalf("did not subtract properly")
	}

	current, err = genserver.Call(genServer, 1, add)
	if err != nil {
		t.Fatalf("should have called genServer successfully")
	}
	if current != 1 {
		t.Fatalf("did not add properly")
	}

	var finalValue uint64
	err = genserver.Shutdown(genServer, genserver.Normal, func(c counter, r genserver.ShutdownReason) error {
		finalValue = c.current
		return nil
	}, nil)

	if finalValue != 1 {
		t.Fatal("did not call shutdown handler properly")
	}
}
