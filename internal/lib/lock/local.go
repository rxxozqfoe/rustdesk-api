package lock

import (
	"sync"
)

type Local struct {
	Locks *sync.Map
}

func (l *Local) Lock(key Key) {
	lock := l.GetLock(key)
	lock.Lock()
}

func (l *Local) UnLock(key Key) {
	lock, ok := l.Locks.Load(key)
	if ok {
		lock.(*sync.Mutex).Unlock()
	}
}

func (l *Local) GetLock(key Key) *sync.Mutex {
	lock, _ := l.Locks.LoadOrStore(key, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func NewLocal() *Local {
	return &Local{
		Locks: &sync.Map{},
	}
}
