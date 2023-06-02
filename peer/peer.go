package peer

import (
	"encoding/json"
	"github.com/yoon7t/protoo-go/transport"
)

type PeerErr transport.TransportErr

type Transcation struct {
	id         int
	accept     AcceptFunc
	reject     RejectFunc
	close      func()
	resultChan chan ResultFuture
}

type RequestData struct {
	Accept  RespondFunc
	Reject  RejectFunc
	Request Request
}

type SendRequestData struct {
	*Request
	*Transcation
}

type PeerChans struct {
	OnRequest      chan RequestData
	SendRequest    chan SendRequestData
	OnNotification chan Notification
	OnClose        chan transport.TransportErr
	OnError        chan transport.TransportErr
}

type Peer struct {
	PeerChans
	id           string
	transport    *transport.WebSocketTransport
	transcations map[int]*Transcation
}

func NewPeer(id string, con *transport.WebSocketTransport) *Peer {
	var peer Peer
	peer.id = id
	peer.transport = con
	peer.PeerChans = PeerChans{
		OnRequest:      make(chan RequestData, 100),
		OnNotification: make(chan Notification, 100),
		SendRequest:    make(chan SendRequestData, 100),
		OnClose:        make(chan transport.TransportErr, 1),
	}
	peer.transcations = make(map[int]*Transcation)

	go peer.Run()
	return &peer
}

func (peer *Peer) Run() {
	for {
		select {
		case msg := <-peer.transport.OnMsg:
			peer.handleMessage(msg)
		case err := <-peer.transport.OnErr:
			peer.handleErr(err)
		case err := <-peer.transport.OnClose:
			peer.handleClose(err)
		case data := <-peer.SendRequest:
			peer.sendRequest(data)
		}
	}
}

func (peer *Peer) sendRequest(req SendRequestData) {
	request := req.Request

	str, err := json.Marshal(request)
	if err != nil {
		return
	}

	peer.transcations[request.Id] = req.Transcation
	peer.transport.SendCh <- str
}

func (peer *Peer) handleClose(err transport.TransportErr) {
	peer.OnClose <- err
}

func (peer *Peer) handleErr(err transport.TransportErr) {
	// Transport error triggers close currently
}

func (peer *Peer) Close() {
	peer.transport.Close()
}

func (peer *Peer) ID() string {
	return peer.id
}

type ResultFuture struct {
	Result json.RawMessage
	Err    *PeerErr
}

func (peer *Peer) Request(method string, data interface{}, success AcceptFunc, reject RejectFunc) chan ResultFuture {
	id := GenerateRandomNumber()
	resChan := make(chan ResultFuture, 1)
	dataStr, err := json.Marshal(data)
	if err != nil {
		resChan <- ResultFuture{nil, &PeerErr{10, err.Error()}}
		return resChan
	}

	request := &Request{
		Request: true,
		Id:      id,
		Method:  method,
		Data:    dataStr,
	}

	transcation := &Transcation{
		id:     id,
		accept: success,
		reject: reject,
		close: func() {
		},
		resultChan: resChan,
	}

	peer.SendRequest <- SendRequestData{request, transcation}
	return resChan
}

func (peer *Peer) Notify(method string, data interface{}) {
	dataStr, err := json.Marshal(data)
	if err != nil {
		return
	}
	notification := &Notification{
		Notification: true,
		Method:       method,
		Data:         dataStr,
	}
	str, err := json.Marshal(notification)
	if err != nil {
		return
	}
	peer.transport.SendCh <- str
}

func (peer *Peer) handleMessage(message []byte) {
	var msg PeerMsg
	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}
	if msg.Request {
		var data Request
		if err := json.Unmarshal(message, &data); err != nil {
			return
		}
		peer.handleRequest(data)
	} else if msg.Response {
		if msg.Ok {
			var data Response
			if err := json.Unmarshal(message, &data); err != nil {
				return
			}
			peer.handleResponse(data)
		} else {
			var data ResponseError
			if err := json.Unmarshal(message, &data); err != nil {
				return
			}
			peer.handleResponseError(data)
		}
	} else if msg.Notification {
		var data Notification
		if err := json.Unmarshal(message, &data); err != nil {
			return
		}
		peer.handleNotification(data)
	}
}

func (peer *Peer) handleRequest(request Request) {

	accept := func(data interface{}) {
		dataStr, err := json.Marshal(data)
		if err != nil {
			return
		}
		response := &Response{
			Response: true,
			Ok:       true,
			Id:       request.Id,
			Data:     dataStr,
		}
		str, err := json.Marshal(response)
		if err != nil {
			return
		}
		//send accept
		peer.transport.SendCh <- str
	}

	reject := func(errorCode int, errorReason string) {
		response := &ResponseError{
			Response:    true,
			Ok:          false,
			Id:          request.Id,
			ErrorCode:   errorCode,
			ErrorReason: errorReason,
		}
		str, err := json.Marshal(response)
		if err != nil {
			return
		}
		//send reject
		peer.transport.SendCh <- str
	}
	_, _ = accept, reject
	peer.OnRequest <- RequestData{
		Accept:  accept,
		Reject:  reject,
		Request: request,
	}
}

func (peer *Peer) handleResponse(response Response) {
	id := response.Id
	transcation := peer.transcations[id]
	if transcation == nil {
		return
	}

	if transcation.accept != nil {
		transcation.accept(response.Data)
	} else {
		transcation.resultChan <- ResultFuture{response.Data, nil}
	}

	delete(peer.transcations, id)
}

func (peer *Peer) handleResponseError(response ResponseError) {
	id := response.Id
	transcation := peer.transcations[id]
	if transcation == nil {
		return
	}

	if transcation.reject != nil {
		transcation.reject(response.ErrorCode, response.ErrorReason)
	} else {
		transcation.resultChan <- ResultFuture{nil, &PeerErr{
			Code: response.ErrorCode,
			Text: response.ErrorReason,
		}}
	}

	delete(peer.transcations, id)
}

func (peer *Peer) handleNotification(notification Notification) {
	peer.OnNotification <- notification
}
