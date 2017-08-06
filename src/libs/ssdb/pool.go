/*
   实现简单的pool
   代码由redis客户端(github.com/garyburd/redigo/redis)里的pool改造而来
*/
package ssdb

import (
	"container/list"
	"errors"
	"sync"
	"time"
	"utils"
)

var ErrPoolExhausted = errors.New("connection pool exhausted")

type errorConnection struct{ err error }

type Pool struct {
	Dial        func() (*Client, error)
	MaxActive   int
	MaxIdle     int
	IdleTimeout time.Duration
	Wait        bool
	mu          sync.Mutex
	cond        *sync.Cond
	closed      bool
	active      int
	idle        list.List
}

type idleConn struct {
	c *Client
	t time.Time
}

func (p *Pool) Get() *Client {
	c, err := p.get()
	if err != nil {
		return nil
	}
	return c
}

func (p *Pool) Close() error {
	p.mu.Lock()
	idle := p.idle
	p.idle.Init()
	p.closed = true
	p.active -= idle.Len()
	if p.cond != nil {
		p.cond.Broadcast()
	}
	p.mu.Unlock()
	for e := idle.Front(); e != nil; e = e.Next() {
		e.Value.(idleConn).c.Close()
	}
	return nil
}

func (p *Pool) release() {
	p.active -= 1
	if p.cond != nil {
		p.cond.Signal()
	}
}

func (p *Pool) get() (*Client, error) {
	p.mu.Lock()

	// Prune stale connections.

	if timeout := p.IdleTimeout; timeout > 0 {
		for i, n := 0, p.idle.Len(); i < n; i++ {
			e := p.idle.Back()
			if e == nil {
				break
			}
			ic := e.Value.(idleConn)
			if ic.t.Add(timeout).After(time.Now()) {
				break
			}
			p.idle.Remove(e)
			p.release()
			p.mu.Unlock()
			ic.c.Close()
			p.mu.Lock()
		}
	}

	for {
		// Get idle connection.
		for i, n := 0, p.idle.Len(); i < n; i++ {
			e := p.idle.Front()
			if e == nil {
				break
			}
			ic := e.Value.(idleConn)
			p.idle.Remove(e)
			p.mu.Unlock()
			//utils.WriteLog("debug", "ssdb pool size get idle:active=%d idle=%d", p.active, p.idle.Len())
			if ic.c != nil {
				return ic.c, nil
			} else {
				utils.WriteLog("error", "got bad ssdb connection")
			}
		}

		// Check for pool closed before dialing a new connection.
		if p.closed {
			p.mu.Unlock()
			return nil, errors.New("redigo: get on closed pool")
		}

		// Dial new connection if under limit.

		if p.MaxActive == 0 || p.active < p.MaxActive {
			dial := p.Dial
			p.active += 1
			p.mu.Unlock()
			c, err := dial()
			if err != nil {
				p.mu.Lock()
				p.release()
				p.mu.Unlock()
				c = nil
			}
			utils.WriteLog("info", "ssdb pool size create new:active=%d idle=%d", p.active, p.idle.Len())
			return c, err
		}

		if !p.Wait {
			p.mu.Unlock()
			return nil, ErrPoolExhausted
		}

		if p.cond == nil {
			p.cond = sync.NewCond(&p.mu)
		}
		p.cond.Wait()
	}
}

func (p *Pool) put(c *Client, forceClose bool) error {
	p.mu.Lock()
	if !p.closed && !forceClose {
		p.idle.PushFront(idleConn{c: c})
		if p.idle.Len() > p.MaxIdle {
			c = p.idle.Remove(p.idle.Back()).(idleConn).c
		} else {
			c = nil
		}
	}

	if c == nil {
		if p.cond != nil {
			p.cond.Signal()
		}
		p.mu.Unlock()
		return nil
	}

	p.release()
	p.mu.Unlock()
	return c.Close()
}
