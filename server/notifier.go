package main

type Uid [32]byte
type Envelope []byte

//Notifier map which can't be tested for a while
func NewNotifier() *Notifier {
	notifier := &Notifier{
		newUsers:      make(chan Uid),
		users:         make(map[Uid]chan *Envelope),
		notifications: make(chan *Notification),
	}

	notifier.Listen()

	return notifier
}

func (notifier *Notifier) Listen() {
	go notifier.GetNewUsers()
	go notifier.GetEnvelopes()
}

func (notifier *Notifier) GetNewUsers() {
	for uid := range notifier.newUsers {
		notifier.users[uid] = make(chan *Envelope)
	}
}

func (notifier *Notifier) GetEnvelopes() {
	for notification := range notifier.notifications {
		notifier.users[notification.user] <- notification.envelope
	}
}

type Notification struct {
	user     Uid
	envelope *Envelope
}

type Notifier struct {
	newUsers      chan Uid
	users         map[Uid]chan *Envelope
	notifications chan *Notification
}

func main() {}
