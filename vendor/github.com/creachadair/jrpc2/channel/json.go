package channel

import (
	"encoding/json"
	"io"
)

const bufSize = 2048

// RawJSON is a framing that transmits and receives records on r and wc, in which
// each record is defined by being a complete JSON value. No padding or other
// separation is added.
func RawJSON(r io.Reader, wc io.WriteCloser) Channel {
	return jsonc{wc: wc, dec: json.NewDecoder(r), buf: make([]byte, bufSize)}
}

// A jsonc implements channel.Channel. Messages sent on a raw channel are not
// explicitly framed, and messages received are framed by JSON syntax.
type jsonc struct {
	wc  io.WriteCloser
	dec *json.Decoder
	buf json.RawMessage
}

// Send implements part of the Channel interface.
func (c jsonc) Send(msg []byte) error {
	if len(msg) == 0 {
		_, err := io.WriteString(c.wc, "null\n")
		return err
	}
	_, err := c.wc.Write(msg)
	return err
}

// Recv implements part of the Channel interface. It reports an error if the
// message is not a structurally valid JSON value. It is safe for the caller to
// treat any record returned as a json.RawMessage.
func (c jsonc) Recv() ([]byte, error) {
	c.buf = c.buf[:0] // reset
	err := c.dec.Decode(&c.buf)
	if err == nil && isNull(c.buf) {
		return nil, nil
	}
	return c.buf, err
}

// Close implements part of the Channel interface.
func (c jsonc) Close() error { return c.wc.Close() }

func isNull(msg json.RawMessage) bool {
	return len(msg) == 4 && msg[0] == 'n' && msg[1] == 'u' && msg[2] == 'l' && msg[3] == 'l'
}
