package main

import "sync"

// Global lock that will try to ensure that we don't
// screw up the underlying directory structure on the
// file system.
//
// We do not use file system locks for now, but it
// probably wouldn't be a bad idea to do so. That would
// avoid multiple instanced of the software fighting with
// each other.
var global sync.RWMutex

// Returned by LockRead and LockWrite, call the Unlock
// method of the Unlocker to release the aquired resource.
//
// You can call the Unlock method as often as you want,
// the underlying lock will only be unlocked exactly once.
// In particular, it is safe to defer Unlock but also
// manually call Unlock in a branch.
type Unlocker struct {
	once       sync.Once
	unlockfunc func()
}

// Unlock the underlying lock. You can call this method
// as often as you want, the underlying lock will only be
// unlocked exactly once.
func (u *Unlocker) Unlock() {
	u.once.Do(u.unlockfunc)
}

// Lock the global lock for reading. Returns an Unlocker
// you should use for unlocking once you are done.
func LockRead() *Unlocker {
	global.RLock()

	return &Unlocker{
		unlockfunc: func() {
			global.RUnlock()
		},
	}
}

// Lock the global lock for writing. Returns an Unlocker
// you should use for unlocking once you are done.
func LockWrite() *Unlocker {
	global.Lock()

	return &Unlocker{
		unlockfunc: func() {
			global.Unlock()
		},
	}
}
