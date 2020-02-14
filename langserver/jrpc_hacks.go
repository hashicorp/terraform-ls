package langserver

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/creachadair/jrpc2/channel"
)

func LspFraming(logger *log.Logger) channel.Framing {
	return func(r io.Reader, wc io.WriteCloser) channel.Channel {
		reader := bufio.NewReader(r)

		mimeType := "application/vscode-jsonrpc; charset=utf-8"
		ctype := "Content-Type: " + mimeType + "\r\n"

		return &hdr{
			mtype:  mimeType,
			ctype:  ctype,
			wc:     wc,
			rd:     reader,
			sBuf:   bytes.NewBuffer(nil),
			logger: logger,
		}
	}
}

// An hdr implements Channel. Messages sent on a hdr channel are framed as a
// header/body transaction, similar to HTTP.
type hdr struct {
	mtype string
	ctype string
	wc    io.WriteCloser
	rd    *bufio.Reader
	sBuf  *bytes.Buffer
	rbuf  []byte

	logger *log.Logger
}

// Send implements part of the Channel interface.
func (h *hdr) Send(msg []byte) error {
	h.sBuf.Reset()
	if h.ctype != "" {
		h.sBuf.WriteString(h.ctype)
	}
	h.sBuf.WriteString("Content-Length: ")
	h.sBuf.WriteString(strconv.Itoa(len(msg)))
	h.sBuf.WriteString("\r\n\r\n")
	h.sBuf.Write(msg)
	_, err := h.wc.Write(h.sBuf.Next(h.sBuf.Len()))
	return err
}

// Recv implements part of the Channel interface.
func (h *hdr) Recv() ([]byte, error) {

	var contentType, contentLength string
	for {
		raw, err := h.rd.ReadString('\n')
		if err == io.EOF && raw != "" {
			// handle a partial line at EOF
		} else if err != nil {
			return nil, err
		}

		line := strings.TrimRight(raw, "\r\n")

		if strings.HasPrefix(line, "\"") {
			h.logger.Printf("Invalid header received, but tolerated: %q", line)
			line = strings.TrimLeft(line, "\"")
		}

		if line == "" {
			break
		} else if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			// This implementation ignores unknown header fields.
			clean := strings.TrimSpace(parts[1])
			switch strings.ToLower(parts[0]) {
			case "content-type":
				contentType = clean
			case "content-length":
				contentLength = clean
			}
		} else {
			return nil, errors.New("invalid header line")
		}
	}

	// Verify that the content-type matches what we expect.
	if contentType != h.mtype {
		h.logger.Printf("Invalid Content-Type header received, but tolerated: %q", contentType)
	}

	// Parse out the required content-length field.
	if contentLength == "" {
		return nil, errors.New("missing required content-length")
	}
	size, err := strconv.Atoi(contentLength)
	if err != nil || size < 0 {
		return nil, errors.New("invalid content-length")
	}

	// We need to use ReadFull here because the buffered reader may not have a
	// big enough buffer to deliver the whole message, and will only issue a
	// single read to the underlying source.
	data := h.rbuf
	if len(data) < size || len(data) > (1<<20) && size < len(data)/4 {
		data = make([]byte, size*2)
		h.rbuf = data
	}
	if _, err := io.ReadFull(h.rd, data[:size]); err != nil {
		return nil, err
	}
	msg := data[:size]

	h.logger.Printf("Received data: %s", string(msg))
	return msg, nil
}

// Close implements part of the Channel interface.
func (h *hdr) Close() error { return h.wc.Close() }
