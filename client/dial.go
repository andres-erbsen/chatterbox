package client

import (
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

	torAddr string
}

func NewConnectionCache(torAddr string) *ConnectionCache {
	return &ConnectionCache{
		connections: make(map[string]chan *transport.Conn),
		torAddr:     torAddr,
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

	dialer := TorAnon(cc.torAddr)
	if cc.torAddr == "DANGEROUS_NO_TOR" {
		dialer = proxy.Direct
	}

	plainconn, err := dialer.Dial("tcp", net.JoinHostPort(addr, strconv.Itoa(port)))
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
