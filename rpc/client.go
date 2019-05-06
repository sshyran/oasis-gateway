package rpc

import (
	"github.com/oasislabs/developer-gateway/mqueue/core"
)

// Client is an interface for any type that sends requests and
// receives responses
type Client interface {
	Request(interface{}) (interface{}, error)
}

// RequestManager handles the client RPC requests. Most requests
// are asynchronous and they are handled by returning an identifier
// that the caller can later on query on to find out the outcome
// of the request.
type RequestManager struct {
	mqueue core.MQueue
	client Client
}

type RequestManagerProperties struct {
	MQueue core.MQueue
	Client Client
}

// NewRequestManager creates a new instance of a request manager
func NewRequestManager(properties RequestManagerProperties) *RequestManager {
	if properties.MQueue == nil {
		panic("MQueue must be set")
	}

	if properties.Client == nil {
		panic("Client must be set")
	}

	return &RequestManager{
		mqueue: properties.MQueue,
		client: properties.Client,
	}
}

// RequestManager starts a request and provides an identifier for the caller to
// find the request later on
func (m *RequestManager) StartRequest(key string, req interface{}) (uint64, error) {
	id, err := m.mqueue.Next(key)
	if err != nil {
		return 0, err
	}

	go m.doRequest(key, id, req)
	return id, nil
}

func (m *RequestManager) doRequest(key string, id uint64, req interface{}) {
	// TODO(stan): we should handle the case in which the request takes too long
	res, err := m.client.Request(req)
	if err != nil {
		m.mqueue.Insert(key, core.Element{
			Value:  Error{ErrorCode: -1, Description: err.Error()},
			Offset: id,
		})
		return

	}

	m.mqueue.Insert(key, core.Element{
		Value:  res,
		Offset: id,
	})
}

// GetResponses retrieves the responses the RequestManager already got
// from the asynchronous requests.
func (m *RequestManager) GetResponses(key string, offset uint64, count uint) ([]interface{}, error) {
	els, err := m.mqueue.Retrieve(key, offset, count)
	if err != nil {
		return nil, err
	}

	var res []interface{}
	for _, el := range els {
		res = append(res, el.Value)
	}

	return res, nil
}

// DiscardResponses discards responses stored by the RequestManager to make space
// for new requests
func (m *RequestManager) DiscardResponses(key string, offset uint64) error {
	return m.mqueue.Discard(key, offset)
}
