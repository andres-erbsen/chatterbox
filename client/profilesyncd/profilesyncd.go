package profilesyncd

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/andres-erbsen/dename/client"
	dename "github.com/andres-erbsen/dename/protocol"
	"io"
	mathrand "math/rand"
	"sync"
	"time"
)

type ProfileSyncd struct {
	stop, force, forceDone chan struct{}
	wg                     sync.WaitGroup

	name     string
	client   *client.Client
	meanRate time.Duration
	onUpdate func(*dename.Profile, *dename.ClientReply, error)
	rand     io.Reader
}

func New(client *client.Client, meanRate time.Duration, name string, onUpdate func(*dename.Profile, *dename.ClientReply, error), rnd io.Reader) (*ProfileSyncd, error) {
	if rnd == nil {
		rnd = rand.Reader
	}
	return &ProfileSyncd{name: name, client: client, meanRate: meanRate, onUpdate: onUpdate, rand: rnd}, nil
}

func (d *ProfileSyncd) Start() {
	d.stop = make(chan struct{})
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.Run()
	}()
}

func (d *ProfileSyncd) Stop() {
	close(d.stop)
	d.wg.Wait()
}

func (d *ProfileSyncd) Force() {
	d.force <- struct{}{}
	<-d.forceDone
}

func (d *ProfileSyncd) Run() {
	delay := time.After(d.pickDelay())
	for {
		select {
		case <-d.stop:
			return
		case <-delay:
			delay = time.After(d.pickDelay())
			d.onUpdate(d.client.LookupReply(d.name))
		case <-d.force:
			delay = time.After(d.pickDelay())
			d.onUpdate(d.client.LookupReply(d.name))
			d.forceDone <- struct{}{}
		}
	}
}

func (d *ProfileSyncd) pickDelay() time.Duration {
	var s [8]byte
	io.ReadFull(d.rand, s[:])
	seed := int64(binary.LittleEndian.Uint64(s[:]))
	return time.Duration(mathrand.New(mathrand.NewSource(seed)).ExpFloat64() * float64(d.meanRate))
}
