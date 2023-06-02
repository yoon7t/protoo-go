package transport

import (
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 5 * time.Second //(pongWait * 8) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 131072 //128k
)

type TransportErr struct {
	Code int
	Text string
}

type TransportChans struct {
	OnMsg   chan []byte
	OnErr   chan TransportErr
	OnClose chan TransportErr
	SendCh  chan []byte
}

type WebSocketTransport struct {
	TransportChans
	socket   *websocket.Conn
	closed   bool
	stop     chan bool
	stopLock sync.RWMutex
	shutdown bool
	tErr     *TransportErr
}

func NewWebSocketTransport(socket *websocket.Conn) *WebSocketTransport {
	var transport WebSocketTransport
	transport.socket = socket
	transport.closed = false
	transport.tErr = nil
	transport.stop = make(chan bool, 100)

	transport.socket.SetCloseHandler(func(code int, text string) error {
		transport.tErr = &TransportErr{code, text}
		transport.Stop()
		return nil
	})

	transport.TransportChans = TransportChans{
		OnMsg:   make(chan []byte, 100),
		OnErr:   make(chan TransportErr, 1),
		OnClose: make(chan TransportErr, 1),
		SendCh:  make(chan []byte, 100),
	}
	return &transport
}

func (transport *WebSocketTransport) Start() {
	go transport.ReadLoop()
	go transport.WriteLoop()
}

func (transport *WebSocketTransport) ReadLoop() {
	defer func() {
		// Signal stop if not already in progress
		transport.Stop()
	}()

	transport.socket.SetReadLimit(maxMessageSize)
	transport.socket.SetReadDeadline(time.Now().Add(pongWait))
	transport.socket.SetPongHandler(func(string) error {
		transport.socket.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := transport.socket.ReadMessage()
		if err != nil {
			if wsErr, ok := err.(*websocket.CloseError); ok {
				transport.tErr = &TransportErr{wsErr.Code, wsErr.Text}
			}
			break
		}

		transport.OnMsg <- message

		// Check stop
		transport.stopLock.RLock()
		stop := transport.shutdown
		transport.stopLock.RUnlock()
		if stop {
			return
		}
	}

}

func (transport *WebSocketTransport) WriteLoop() {
	defer func() {
		// Make sure the whole transport is marked for stop if not already
		transport.Stop()
		// Shut down the connection. Will kill reader if blocked on read
		transport.close()
	}()

	pingTicker := time.NewTicker(pingPeriod)

	for {
		select {
		case _ = <-pingTicker.C:
			if err := transport.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				pingTicker.Stop()
				if wsErr, ok := err.(*websocket.CloseError); ok {
					transport.tErr = &TransportErr{wsErr.Code, wsErr.Text}
				}
				return
			}
		case message := <-transport.SendCh:
			{
				err := transport.socket.WriteMessage(websocket.TextMessage, message)
				// TODO handle send error
				if err != nil {
				}
			}
		case <-transport.stop:
			return
		}
	}
}

func (transport *WebSocketTransport) Close() {
	transport.Stop()
}

// Start shutdown
// Set shutdown bool to trigger read loop exit on next loop
// signal write loop to exit and begin socket close

func (transport *WebSocketTransport) Stop() {
	transport.stopLock.Lock()
	defer transport.stopLock.Unlock()
	if !transport.shutdown {
		transport.shutdown = true
		transport.stop <- true
	}
}

/*
* Close connection.
 */

func (transport *WebSocketTransport) close() {
	if transport.closed == false {
		transport.socket.Close()
		transport.closed = true
		if transport.tErr != nil {
			transport.OnClose <- *transport.tErr

		} else {
			transport.OnClose <- TransportErr{100, "Closed"}
		}
	} else {
	}
}
