package govern

import (
	"new-milli/transport"
)

var _ transport.Transporter = (*Transport)(nil)

// Transport is a govern transport.
type Transport struct {
	operation   string
	reqHeader   transport.Header
	replyHeader transport.Header
}

// Kind returns the transport kind.
func (tr *Transport) Kind() transport.Kind {
	return transport.KindHTTP // Using HTTP kind since govern server is HTTP-based
}

// Operation returns the operation.
func (tr *Transport) Operation() string {
	return tr.operation
}

// RequestHeader returns the request header.
func (tr *Transport) RequestHeader() transport.Header {
	return tr.reqHeader
}

// ReplyHeader returns the reply header.
func (tr *Transport) ReplyHeader() transport.Header {
	return tr.replyHeader
}

// HeaderCarrier is a carrier for HTTP headers.
type HeaderCarrier struct {
	header map[string]string
}

// Get returns the value associated with the passed key.
func (hc *HeaderCarrier) Get(key string) string {
	if hc.header == nil {
		return ""
	}
	return hc.header[key]
}

// Set stores the key-value pair.
func (hc *HeaderCarrier) Set(key string, value string) {
	if hc.header == nil {
		hc.header = make(map[string]string)
	}
	hc.header[key] = value
}

// Keys lists the keys stored in this carrier.
func (hc *HeaderCarrier) Keys() []string {
	if hc.header == nil {
		return nil
	}
	keys := make([]string, 0, len(hc.header))
	for k := range hc.header {
		keys = append(keys, k)
	}
	return keys
}
