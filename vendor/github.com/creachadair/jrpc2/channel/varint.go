package channel

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
)

// Varint is a framing that transmits and receives messages on r and wc, with
// each message prefixed by its length encoded in a varint as defined by the
// encoding/binary package.
func Varint(r io.Reader, wc io.WriteCloser) Channel {
	return &varint{
		wc:  wc,
		rd:  bufio.NewReader(r),
		buf: bytes.NewBuffer(nil),
	}
}

// A varint implements Channel for varint-prefixed messages.
type varint struct {
	wc  io.WriteCloser
	rd  *bufio.Reader
	buf *bytes.Buffer
}

// Send implements part of the Channel interface. It encodes len(msg) as a
// varint, concatenates it with the message body, and writes the framed message
// to the underlying writer.
func (v *varint) Send(msg []byte) error {
	var ln [binary.MaxVarintLen64]byte
	nb := binary.PutUvarint(ln[:], uint64(len(msg)))

	v.buf.Reset()
	v.buf.Grow(nb + len(msg))
	v.buf.Write(ln[:nb])
	v.buf.Write(msg)
	_, err := v.wc.Write(v.buf.Next(v.buf.Len()))
	return err
}

// Recv implements part of the Channel interface. It decodes a varint message
// length then reads that many bytes from the underlying reader.
func (v *varint) Recv() ([]byte, error) {
	ln, err := v.decode()
	if err != nil {
		return nil, err
	}
	out := make([]byte, ln)
	nr, err := io.ReadFull(v.rd, out)
	return out[:nr], err
}

// Close implements part of the Channel interface.
func (v *varint) Close() error { return v.wc.Close() }

func (v *varint) decode() (int, error) {
	ln, err := binary.ReadUvarint(v.rd)
	if err != nil {
		return 0, err
	}
	return int(ln), nil
}
