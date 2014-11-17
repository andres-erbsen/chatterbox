package main

type Uid [32]byte
type Envelope []byte

//Notifier map which can't be tested for a while
func NewNotifier() *Notifier {
	notifier := &Notifier{
		newClients:      make(chan Client),
		clientsToRemove: make(chan Uid),
		users:           make(map[Uid]chan *Envelope),
		notifications:   make(chan *Notification),
	}

	notifier.Listen()

	return notifier
}

func (notifier *Notifier) Listen() {
	go notifier.addClients()
	go notifier.removeClients()
	go notifier.getEnvelopes()
}

func (notifier *Notifier) addClients() {
	for client := range notifier.newClients {
		notifier.users[client.user] = client.channel
	}
}

func (notifier *Notifier) removeClients() {
	for uid := range notifier.clientsToRemove {
		delete(notifier.users, uid)
	}
}

func (notifier *Notifier) getEnvelopes() {
	for notification := range notifier.notifications {
		if channel, ok := notifier.users[notification.user]; ok == true {
			channel <- notification.envelopes
		}
	}
}

type Client struct {
	user    Uid
	channel chan *Envelope
}

type Notification struct {
	user      Uid
	envelopes *Envelope
}

type Notifier struct {
	newClients      chan Client
	clientsToRemove chan Uid
	users           map[Uid]chan *Envelope
	notifications   chan *Notification
}

func main() {}
