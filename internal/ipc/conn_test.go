package ipc_test

import (
	"net"
	"sync"
	"testing"

	"github.com/szdytom/tb/internal/ipc"
)

// newPipe returns a connected pair of Conns. The receiver goroutine
// must be started before the sender to avoid blocking on net.Pipe's
// synchronous writes.
func newPipe(t *testing.T) (server, client *ipc.Conn) {
	t.Helper()
	s, c := net.Pipe()
	t.Cleanup(func() { s.Close() })
	t.Cleanup(func() { c.Close() })
	return ipc.NewConn(s), ipc.NewConn(c)
}

func TestSendReceive(t *testing.T) {
	server, client := newPipe(t)

	var got ipc.Request
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Receive(&got); err != nil {
			t.Errorf("receive: %v", err)
		}
	}()

	req := ipc.NewRequest(1, ipc.OpPing, nil)
	if err := client.Send(req); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if got.ID != 1 || got.Op != ipc.OpPing {
		t.Errorf("got ID=%d Op=%q, want ID=1 Op=Ping", got.ID, got.Op)
	}
}

func TestConsecutiveMessages(t *testing.T) {
	server, client := newPipe(t)

	ch := make(chan ipc.Request, 2)
	go func() {
		for i := 0; i < 2; i++ {
			var got ipc.Request
			if err := server.Receive(&got); err != nil {
				return
			}
			ch <- got
		}
	}()

	if err := client.Send(ipc.NewRequest(1, ipc.OpPing, nil)); err != nil {
		t.Fatal(err)
	}
	if err := client.Send(ipc.NewRequest(2, ipc.OpPing, nil)); err != nil {
		t.Fatal(err)
	}

	r1 := <-ch
	r2 := <-ch

	if r1.ID != 1 || r2.ID != 2 {
		t.Errorf("message order: got IDs %d and %d, want 1 then 2", r1.ID, r2.ID)
	}
}

func TestSendReceiveResponse(t *testing.T) {
	server, client := newPipe(t)

	var got ipc.Response
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := client.Receive(&got); err != nil {
			t.Errorf("receive: %v", err)
		}
	}()

	resp := ipc.OKResponse(1, ipc.PingResponse{Message: "pong"})
	if err := server.Send(resp); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if got.ID != 1 || !got.Ok {
		t.Errorf("got ID=%d Ok=%v, want ID=1 Ok=true", got.ID, got.Ok)
	}
}
