package server

import "sync"

type Notifier struct {
	sync.RWMutex
	waiters map[[32]byte][]chan []byte
}

func (n *Notifier) StartWaiting(uid *[32]byte) chan []byte {
	ch := make(chan []byte)
	n.Lock()
	defer n.Unlock()
	n.waiters[*uid] = append(n.waiters[*uid], ch)
	return ch
}

// StopWaitingSync blocks returns after closing removeCh. Calling
// StopWaitingSync while a notification is pending will wait for that
// notification to be handled. Calling StopWaitingSync from the thread that
// should be handling the notification will therefore result in a deadlock.
// When removeCh is not waiting, nothing is done (but the blocking
// considerations still apply).
func (n *Notifier) StopWaitingSync(uid *[32]byte, removeCh chan []byte) {
	n.Lock()
	defer n.Unlock()
	l := n.waiters[*uid]
	i := 0
	for _, ch := range l {
		if ch != removeCh {
			l[i] = ch
			i++
		}
	}
	n.waiters[*uid] = l[:i]
	close(removeCh)
}

func (n *Notifier) Notify(uid *[32]byte, notification []byte) {
	n.Lock()
	defer n.Unlock()
	for _, ch := range n.waiters[*uid] {
		ch <- notification
	}
}
