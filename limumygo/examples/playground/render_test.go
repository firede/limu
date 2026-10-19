package main

import (
	"testing"
	"time"
)

func TestPlanner(t *testing.T) {
	p := &planner{delay: editDelay}
	t0 := time.Unix(1000, 0)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	step := func(version uint64, width float64, ms int, start bool, wait time.Duration) {
		t.Helper()
		gotStart, gotWait := p.next(version, width, at(ms))
		if gotStart != start || gotWait != wait {
			t.Fatalf("next(%d, %v) at %d ms = %v, %v; want %v, %v", version, width, ms, gotStart, gotWait, start, wait)
		}
	}
	// A pane not laid out waits for its width; the first text is laid out at once.
	step(1, 0, 0, false, 0)
	step(1, 500, 10, true, 0)
	if p.stamp != 1 {
		t.Fatalf("stamp %d", p.stamp)
	}
	step(1, 500, 20, false, 0)
	// Edits wait for editDelay after the last of them.
	step(2, 500, 100, false, editDelay)
	step(3, 500, 200, false, editDelay)
	step(3, 500, 300, false, 50*time.Millisecond)
	step(3, 500, 350, true, 0)
	// A new width is laid out at once.
	step(3, 420, 360, true, 0)
	// A text loaded is laid out at once.
	p.urgent = true
	step(4, 420, 370, true, 0)
	step(5, 420, 380, false, editDelay)
	// Only the newest job's result is current.
	if p.stamp != 4 || p.current(3) || !p.current(4) {
		t.Errorf("stamp %d", p.stamp)
	}
}

// A result that comes back after a newer job started is dropped.
func TestStaleResult(t *testing.T) {
	a := newApp("flowchart LR\n  A --> B\n", appearanceLight)
	old := job{stamp: 1, version: a.buf.Version(), src: "pie\n  \"a\": 1\n", width: 500}.run()
	a.plan.stamp = 2
	a.apply(old)
	if a.scene != nil {
		t.Error("a stale result was shown")
	}
	a.apply(job{stamp: 2, version: a.buf.Version(), src: a.buf.String(), width: 500}.run())
	if a.scene == nil || a.typ != "flowchart" {
		t.Errorf("the newest result was not shown: %q", a.typ)
	}
}

// Jobs run on other goroutines and hand their results back through post.
func TestAsync(t *testing.T) {
	a := newApp("flowchart LR\n  A --> B\n", appearanceLight)
	posted := make(chan func(), 4)
	a.post = func(fn func()) { posted <- fn }
	tt := testerFor(a)
	if a.scene != nil {
		t.Fatal("laid out in the frame")
	}
	(<-posted)()
	tt.Frame()
	if a.scene == nil {
		t.Fatal("the result was not shown")
	}
}
