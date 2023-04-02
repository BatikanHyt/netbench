package protocols

import (
	"context"
	"net"
)

type trackingConn struct {
	net.Conn
	readSize  *int64
	writeSize *int64
}

func (c *trackingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if err == nil {
		*c.readSize += int64(n)
	}

	return n, err
}

func (c *trackingConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if err == nil {
		*c.writeSize += int64(n)
	}

	return n, err
}

func (c *trackingConn) Close() error {
	// Do any cleanup here
	return c.Conn.Close()
}

func DialContextWithBytesTracked(ctx context.Context, network, address string, readBytes, writeBytes *int64) (net.Conn, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	// Wrap the connection with a tracking Conn
	return &trackingConn{
		Conn:      conn,
		readSize:  readBytes,
		writeSize: writeBytes,
	}, nil
}
