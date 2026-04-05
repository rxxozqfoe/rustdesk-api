package lock

import "sync"

type Key string

const (
	LockRegisterByOauth Key = "registerByOauth"
)

type Locker interface {
	GetLock(key Key) *sync.Mutex
	Lock(key Key)
	UnLock(key Key)
}
