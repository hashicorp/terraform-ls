package channel

import (
	"errors"
	"io"
)

// A Framing converts a reader and a writer into a Channel with a particular
// message-framing discipline.
type Framing func(io.Reader, io.WriteCloser) Channel

// WithTrigger returns a Channel that delegates I/O operations to ch, and when
// a Recv operation on ch returns io.EOF it synchronously calls the trigger.
func WithTrigger(ch Channel, trigger func()) Channel {
	return triggered{ch: ch, trigger: trigger}
}

type triggered struct {
	ch      Channel
	trigger func()
}

// Recv implements part of the channel.Channel interface. It delegates to the
// wrapped channel and calls the trigger when the delegate returns io.EOF.
func (c triggered) Recv() ([]byte, error) {
	msg, err := c.ch.Recv()
	if err == io.EOF {
		c.trigger()
	}
	return msg, err
}

func (c triggered) Send(msg []byte) error { return c.ch.Send(msg) }
func (c triggered) Close() error          { return c.ch.Close() }

type direct struct {
	send chan<- []byte
	recv <-chan []byte
}

func (d direct) Send(msg []byte) (err error) {
	cp := make([]byte, len(msg))
	copy(cp, msg)
	defer func() {
		if p := recover(); p != nil {
			err = errors.New("send on closed channel")
		}
	}()
	d.send <- cp
	return nil
}

func (d direct) Recv() ([]byte, error) {
	msg, ok := <-d.recv
	if ok {
		return msg, nil
	}
	return nil, io.EOF
}

func (d direct) Close() error { close(d.send); return nil }

// Direct returns a pair of synchronous connected channels that pass message
// buffers directly in memory without framing or encoding. Sends to client will
// be received by server, and vice versa.
func Direct() (client, server Channel) {
	c2s := make(chan []byte)
	s2c := make(chan []byte)
	client = direct{send: c2s, recv: s2c}
	server = direct{send: s2c, recv: c2s}
	return
}
