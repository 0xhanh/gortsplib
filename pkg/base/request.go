// Package base contains the primitives of the RTSP protocol.
package base

import (
	"bufio"
	"fmt"
	"strconv"
)

const (
	rtspProtocol10           = "RTSP/1.0"
	requestMaxMethodLength   = 64
	requestMaxURLLength      = 2048
	requestMaxProtocolLength = 64
	// tunnel:
	httpProtocol10 = "HTTP/1.0"
)

// Method is the method of a RTSP request.
type Method string

// methods.
const (
	// tunnel:
	Get  Method = "GET"
	Post Method = "POST"

	Announce     Method = "ANNOUNCE"
	Describe     Method = "DESCRIBE"
	GetParameter Method = "GET_PARAMETER"
	Options      Method = "OPTIONS"
	Pause        Method = "PAUSE"
	Play         Method = "PLAY"
	Record       Method = "RECORD"
	Setup        Method = "SETUP"
	SetParameter Method = "SET_PARAMETER"
	Teardown     Method = "TEARDOWN"
)

// Request is a RTSP request.
type Request struct {
	// request method
	Method Method

	// request url
	URL *URL

	// tunnel:
	Protocol string

	// map of header values
	Header Header

	// optional body
	Body []byte
}

// Unmarshal reads a request.
func (req *Request) Unmarshal(br *bufio.Reader) error {
	byts, err := readBytesLimited(br, ' ', requestMaxMethodLength)
	if err != nil {
		return err
	}
	req.Method = Method(byts[:len(byts)-1])

	if req.Method == "" {
		return fmt.Errorf("empty method")
	}

	byts, err = readBytesLimited(br, ' ', requestMaxURLLength)
	if err != nil {
		return err
	}
	rawURL := string(byts[:len(byts)-1])

	if rawURL != "*" {
		ur, err := ParseURL(rawURL)
		if err != nil {
			return fmt.Errorf("invalid URL (%v)", rawURL)
		}
		req.URL = ur
	} else {
		req.URL = nil
	}

	byts, err = readBytesLimited(br, '\r', requestMaxProtocolLength)
	if err != nil {
		return err
	}
	proto := byts[:len(byts)-1]

	// tunnel:
	if string(proto) != rtspProtocol10 && string(proto) != httpProtocol10 {
		return fmt.Errorf("expected '%s' or '%s', got %v", rtspProtocol10, httpProtocol10, proto)
	}

	err = readByteEqual(br, '\n')
	if err != nil {
		return err
	}

	err = req.Header.unmarshal(br)
	if err != nil {
		return err
	}

	err = (*body)(&req.Body).unmarshal(req.Header, br)
	if err != nil {
		return err
	}

	return nil
}

// MarshalSize returns the size of a Request.
func (req Request) MarshalSize() int {
	n := len(req.Method) + 1

	if req.URL != nil {
		n += len(req.URL.CloneWithoutCredentials().String())
	} else {
		n++
	}

	n += 1 + len(rtspProtocol10) + 2

	if len(req.Body) != 0 {
		req.Header["Content-Length"] = HeaderValue{strconv.FormatInt(int64(len(req.Body)), 10)}
	}

	n += req.Header.marshalSize()

	n += body(req.Body).marshalSize()

	return n
}

// MarshalTo writes a Request.
func (req Request) MarshalTo(buf []byte) (int, error) {
	pos := 0

	pos += copy(buf[pos:], []byte(req.Method))
	buf[pos] = ' '
	pos++

	if req.URL != nil {
		// tunnel:
		if req.proto() == httpProtocol10 {
			// path: /profile/media.smp
			path := []byte(req.URL.Path)
			pos += copy(buf[pos:], path)
		} else {
			pos += copy(buf[pos:], []byte(req.URL.CloneWithoutCredentials().String()))
		}
	} else {
		pos += copy(buf[pos:], []byte("*"))
	}

	buf[pos] = ' '
	pos++
	// tunnel:
	pos += copy(buf[pos:], req.proto())
	buf[pos] = '\r'
	pos++
	buf[pos] = '\n'
	pos++

	if len(req.Body) != 0 {
		req.Header["Content-Length"] = HeaderValue{strconv.FormatInt(int64(len(req.Body)), 10)}
	}

	pos += req.Header.marshalTo(buf[pos:])

	pos += body(req.Body).marshalTo(buf[pos:])

	return pos, nil
}

// Marshal writes a Request.
func (req Request) Marshal() ([]byte, error) {
	buf := make([]byte, req.MarshalSize())
	_, err := req.MarshalTo(buf)
	return buf, err
}

// String implements fmt.Stringer.
func (req Request) String() string {
	buf, _ := req.Marshal()
	return string(buf)
}

// tunnel:
// get protocol
func (req Request) proto() string {
	proto := req.Protocol
	if proto == "" {
		proto = rtspProtocol10
	}

	return proto
}
