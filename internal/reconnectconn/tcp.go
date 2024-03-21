package reconnectconn

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type Conn struct {
	addr          string        // 如果不传的话 必须要用SetConn把net.Conn设置进来
	rwTimeout     time.Duration // ReadFull, WriteAll 和 WriteAndReadResp 用到的timeout, dial的timeout也是这个 凑合一下
	connected     bool
	retryTimes    int
	retryInterval time.Duration
	mutex         sync.Mutex // 所有rw操作都会有锁
	conn          net.Conn
	errFunc       func(error)
}

func New(addr string, timeOut time.Duration, retryTimes int, retryInterval time.Duration, errFunc func(error)) *Conn {
	return &Conn{
		addr:          addr,
		rwTimeout:     timeOut,
		retryTimes:    retryTimes,
		retryInterval: retryInterval,
		errFunc:       errFunc,
	}
}

func (c *Conn) Write(bs []byte) (n int, err error) {
	err = c.wrapRW(func() error {
		n, err = c.conn.Write(bs)
		return err
	})
	return
}
func (c *Conn) Read(buffer []byte) (n int, err error) {
	err = c.wrapRW(func() error {
		n, err = c.conn.Read(buffer)
		return err
	})
	return n, err
}
func (c *Conn) wrapRW(rw func() error) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err = c.prepare(); err != nil {
		return err
	}
	defer func() {
		c.closeForError(err)
	}()
	if err = rw(); err != nil {
		return err
	}
	return nil
}
func (c *Conn) prepare() error {
	if c.connected {
		return nil
	}
	if c.addr == "" {
		return fmt.Errorf("no server addr. wait for SetConn call")
	}
	i := 0
	var conn net.Conn
	var err error
	for i < c.retryTimes {
		i++
		conn, err = net.DialTimeout("tcp", c.addr, c.rwTimeout)
		if err == nil {
			c.connected = true
			c.conn = conn
			return nil
		}
		time.Sleep(c.retryInterval)
	}
	return err
}

func (c *Conn) closeForError(err error) {
	c.errFunc(err)
}

func (c *Conn) Close() {
	c.conn.Close()
}
