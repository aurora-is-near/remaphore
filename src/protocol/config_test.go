package protocol

import (
	"bytes"
	"testing"
)

func TestPeers_Remove(t *testing.T) {
	peers := Peers{
		{PublicKey: []byte("first")},
		{PublicKey: []byte("second")},
		{PublicKey: []byte("third")},
		{PublicKey: []byte("fourth")},
	}
	if peers = peers.Remove([]byte("not existent")); len(peers) != 4 {
		t.Error("Wrong remove")
	}
	if peers = peers.Remove([]byte("third")); len(peers) != 3 {
		t.Error("Not removed")
	}
	if peers = peers.Remove([]byte("first")); len(peers) != 2 {
		t.Error("Not removed")
	}
	if peers = peers.Remove([]byte("fourth")); len(peers) != 1 {
		t.Error("Not removed")
	}
	if !bytes.Equal(peers[0].PublicKey, []byte("second")) {
		t.Error("Missing entry")
	}
	if peers = peers.Remove([]byte("second")); len(peers) != 0 {
		t.Error("Not removed")
	}
	if peers = peers.Remove([]byte("second")); len(peers) != 0 {
		t.Error("Error on empty")
	}
}
