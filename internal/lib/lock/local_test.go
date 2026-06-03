package lock

import (
	"fmt"
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

func TestLocal_Lock(t *testing.T) {
	l := NewLocal()
	wg := sync.WaitGroup{}
	m := 10
	wg.Add(m)
	i := 0
	for j := 0; j < m; j++ {
		go func() {
			l.Lock("key")
			//fmt.Println(j, i)
			i++
			fmt.Println(j, i)
			l.UnLock("key")
			wg.Done()
		}()
	}

	wg.Wait()
	fmt.Println(i)

}
func TestSyncMap(t *testing.T) {
	m := sync.Map{}
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		v, ok := m.LoadOrStore("key", 1)
		fmt.Println(1, v, ok)
		wg.Done()
	}()
	go func() {
		v, ok := m.LoadOrStore("key", 2)
		fmt.Println(2, v, ok)
		wg.Done()
	}()
	go func() {
		v, ok := m.LoadOrStore("key", 3)
		fmt.Println(3, v, ok)
		wg.Done()
	}()
	wg.Wait()
}
