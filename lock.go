package main

import "sync"

// Global lock that will try to ensure that we don't
// screw up the underlying file system.
//
// We do not use file system locks for now, but it
// probably wouldn't be a bad idea to do so. That would
// avoid multiple instanced of the software fighting with
// each other.
var global sync.RWMutex

// Lock the global lock for reading.
func LockRead() {
	global.RLock()
}

// Unlock the global lock for reading. Only
// call this function exactly once after calling
// LockRead.
func UnlockRead() {
	global.RUnlock()
}

// Lock the global lock for writing.
func LockWrite() {
	global.Lock()
}

// Unlock the global lock for writing. Only
// call this function exactly once after calling
// LockWrite.
func UnlockWrite() {
	global.Unlock()
}
