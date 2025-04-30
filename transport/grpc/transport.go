package grpc

import (
	"new-milli/transport"
)

var _ transport.Transporter = (*Transport)(nil)

// Transport is a gRPC transport.
type Transport struct {
	operation  string
	reqHeader  transport.Header
	respHeader transport.Header
}

// Kind returns the transport kind.
func (tr *Transport) Kind() transport.Kind {
	return transport.KindGRPC
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
	return tr.respHeader
}

// HeaderCarrier is a carrier for gRPC metadata.
type HeaderCarrier struct {
	metadata map[string]string
}

// Get returns the value associated with the passed key.
func (hc *HeaderCarrier) Get(key string) string {
	if hc.metadata == nil {
		return ""
	}
	return hc.metadata[key]
}

// Set stores the key-value pair.
func (hc *HeaderCarrier) Set(key string, value string) {
	if hc.metadata == nil {
		hc.metadata = make(map[string]string)
	}
	hc.metadata[key] = value
}

// Keys lists the keys stored in this carrier.
func (hc *HeaderCarrier) Keys() []string {
	if hc.metadata == nil {
		return nil
	}
	keys := make([]string, 0, len(hc.metadata))
	for k := range hc.metadata {
		keys = append(keys, k)
	}
	return keys
}
