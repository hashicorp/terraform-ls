package channel

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// Line is a framing discipline for messages terminated by a Unicode LF
// (10). This framing has the constraint that records may not contain LF.
var Line = Split('\n')

// Split returns a framing in which each message is terminated by the specified
// byte value. The framing has the constraint that outbound records may not
// contain the split byte internally.
func Split(b byte) Framing {
	return func(r io.Reader, wc io.WriteCloser) Channel {
		return split{split: b, wc: wc, buf: bufio.NewReader(r)}
	}
}

// split implements Channel in which messages are terminated by occurrences of
// the specified byte. Outbound messages may not contain the split byte.
type split struct {
	split byte
	wc    io.WriteCloser
	buf   *bufio.Reader
}

// Send implements part of the Channel interface.  It reports an error if msg
// contains a split byte.
func (c split) Send(msg []byte) error {
	if bytes.ContainsAny(msg, string(c.split)) {
		return errors.New("message contains split byte")
	}
	out := make([]byte, len(msg)+1)
	copy(out, msg)
	out[len(msg)] = c.split
	_, err := c.wc.Write(out)
	return err
}

// Recv implements part of the Channel interface.
func (c split) Recv() ([]byte, error) {
	var buf bytes.Buffer
	for {
		chunk, err := c.buf.ReadSlice(c.split)
		buf.Write(chunk)
		if err == bufio.ErrBufferFull {
			continue // incomplete line
		}
		line := buf.Bytes()
		if n := len(line) - 1; n >= 0 {
			return line[:n], err
		}
		return nil, err
	}
}

// Close implements part of the Channel interface.
func (c split) Close() error { return c.wc.Close() }
