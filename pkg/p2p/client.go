package p2p

import "net"

type Dialer struct {
	cfg Config
}

func (d *Dialer) Dial(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, d.cfg.HandshakeTimeout)
}
