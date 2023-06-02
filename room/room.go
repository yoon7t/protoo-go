package room

import (
	"fmt"
	"sync"

	"github.com/yoon7t/protoo-go/peer"
	"github.com/yoon7t/protoo-go/transport"
)

type Room struct {
	*sync.Mutex
	peers  map[string]*peer.Peer
	closed bool
	id     string
}

func NewRoom(roomId string) *Room {
	room := &Room{
		peers:  make(map[string]*peer.Peer),
		closed: false,
		id:     roomId,
	}
	room.Mutex = new(sync.Mutex)
	return room
}

func (room *Room) CreatePeer(peerId string, transport *transport.WebSocketTransport) *peer.Peer {
	newPeer := peer.NewPeer(peerId, transport)
	// Maybe add peer here
	return newPeer
}

func (room *Room) AddPeer(newPeer *peer.Peer) {
	room.Lock()
	defer room.Unlock()
	room.peers[newPeer.ID()] = newPeer

	// TODO auto disconnect from all rooms, but client does this too..
}

func (room *Room) GetPeer(peerId string) *peer.Peer {
	room.Lock()
	defer room.Unlock()
	if peer, ok := room.peers[peerId]; ok {
		return peer
	}
	return nil
}

func (room *Room) Map(fn func(string, *peer.Peer)) {
	room.Lock()
	defer room.Unlock()
	for id, peer := range room.peers {
		fn(id, peer)
	}
}

func (room *Room) GetPeers() map[string]*peer.Peer {
	return room.peers
}

func (room *Room) RemovePeer(peerId string) {
	room.Lock()
	defer room.Unlock()
	delete(room.peers, peerId)
}

func (room *Room) ID() string {
	return room.id
}

func (room *Room) HasPeer(peerId string) bool {
	room.Lock()
	defer room.Unlock()
	_, ok := room.peers[peerId]
	return ok
}

func (room *Room) Notify(from *peer.Peer, method string, data interface{}) {
	room.Lock()
	defer room.Unlock()
	for id, peer := range room.peers {
		//send to other peers
		if id != from.ID() {
			peer.Notify(method, data)
		}
	}
}

func (room *Room) Close() {
	room.Lock()
	defer room.Unlock()
	for id, peer := range room.peers {
		fmt.Println("Close peer:", id)
		peer.Close()
	}
	room.closed = true
}
