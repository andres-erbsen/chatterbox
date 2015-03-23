package client

import (
	"crypto/rand"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
	"golang.org/x/net/proxy"
)

type ConnectionCache struct {
	sync.Mutex
	connections map[string]chan *transport.Conn

	dialer proxy.Dialer
}

func NewConnectionCache(dialer proxy.Dialer) *ConnectionCache {
	return &ConnectionCache{
		connections: make(map[string]chan *transport.Conn),
		dialer:      dialer,
	}
}

func (cc *ConnectionCache) Put(k string, conn *transport.Conn) {
	cc.Lock()
	ch, ok := cc.connections[k]
	if !ok {
		ch = make(chan *transport.Conn, 1)
		cc.connections[k] = ch
	}
	cc.Unlock()

	select {
	case ch <- conn:
	default:
		conn.Close()
	}
	return
}

// there must be a conn with the key k and no Puts to k may be issued in
// parallel with Close
func (cc *ConnectionCache) PutClose(k string) {
	cc.Lock()
	defer cc.Unlock()
	ch, ok := cc.connections[k]
	if !ok {
		return
	}
	delete(cc.connections, k)

	close(ch)
}

// Caller MUST call Put or PutClose after this
func (cc *ConnectionCache) DialServer(cacheKey, addr string, port int, serverPK, pk, sk *[32]byte) (conn *transport.Conn, err error) {
	cc.Lock()
	ch, ok := cc.connections[cacheKey]
	if !ok {
		ch = make(chan *transport.Conn, 1)
		cc.connections[cacheKey] = ch
	}
	cc.Unlock()

	if ok {
		conn := <-ch
		if conn != nil {
			return conn, nil
		}
	}
	// ch is empty now

	plainconn, err := cc.dialer.Dial("tcp", net.JoinHostPort(addr, strconv.Itoa(port)))
	if err != nil {
		cc.PutClose(cacheKey)
		return nil, err
	}
	conn, _, err = transport.Handshake(plainconn, pk, sk, serverPK, proto.SERVER_MESSAGE_SIZE)
	if err != nil {
		conn.Close()
		cc.PutClose(cacheKey)
		return nil, err
	}

	return conn, nil
}

type anonDialer struct{ torAddr string }

func NewAnonDialer(torAddr string) proxy.Dialer { return &anonDialer{torAddr} }

func (dl *anonDialer) Dial(network, addr string) (c net.Conn, err error) {
	if dl.torAddr == "DANGEROUS_NO_TOR" {
		return proxy.Direct.Dial(network, addr)
	}
	var identity [16]byte
	if _, err := rand.Read(identity[:]); err != nil {
		panic(err)
	}
	dialer, err := proxy.SOCKS5("tcp", dl.torAddr, &proxy.Auth{
		User:     fmt.Sprintf("%x", identity[:8]),
		Password: fmt.Sprintf("%x", identity[8:]),
	}, proxy.Direct)
	if err != nil {
		return nil, err
	}
	return dialer.Dial(network, addr)
}
