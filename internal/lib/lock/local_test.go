package lock

import (
	"sync"
	"testing"
)

func TestLocal_GetLock(t *testing.T) {
	l := NewLocal()
	wg := sync.WaitGroup{}
	const n = 3
	wg.Add(n)
	locks := make([]*sync.Mutex, n)
	i := 0
	for j := 0; j < n; j++ {
		go func(idx int) {
			defer wg.Done()
			lk := l.GetLock("key")
			locks[idx] = lk
			// The shared mutex returned by GetLock must serialize i++.
			lk.Lock()
			i++
			lk.Unlock()
		}(j)
	}
	wg.Wait()

	// GetLock must return the same mutex for the same key.
	for _, lk := range locks {
		if lk != locks[0] {
			t.Fatalf("GetLock returned different mutexes for the same key")
		}
	}
	// Every increment ran under that shared mutex, so none were lost.
	if i != n {
		t.Fatalf("expected i == %d, got %d", n, i)
	}
}

// TestLocal_Lock verifies that Lock/UnLock on the same key serialize their
// critical sections: m goroutines each increment a shared counter under the
// lock, and every increment must survive.
func TestLocal_Lock(t *testing.T) {
	l := NewLocal()
	wg := sync.WaitGroup{}
	const m = 10
	wg.Add(m)
	i := 0
	for j := 0; j < m; j++ {
		go func() {
			defer wg.Done()
			l.Lock("key")
			i++
			l.UnLock("key")
		}()
	}
	wg.Wait()

	if i != m {
		t.Fatalf("expected i == %d after serialized increments, got %d", m, i)
	}
}
