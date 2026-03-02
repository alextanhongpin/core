package lock

import "sync"

type KeyLock struct {
	mu   sync.Locker
	keys map[string]sync.Locker
}

func NewKeyLock() *KeyLock {
	return &KeyLock{
		mu:   new(sync.Mutex),
		keys: make(map[string]sync.Locker),
	}
}

func (kl *KeyLock) Lock(key string) func() {
	kl.mu.Lock()
	l, ok := kl.keys[key]
	if ok {
		kl.mu.Unlock()

		l.Lock()

		return l.Unlock
	}

	l = new(sync.Mutex)
	l.Lock()
	kl.keys[key] = l
	kl.mu.Unlock()

	return func() {
		kl.mu.Lock()
		delete(kl.keys, key)
		kl.mu.Unlock()

		l.Unlock()
	}
}
