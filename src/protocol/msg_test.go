package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage(t *testing.T) {
	peer1 := NewConfig()
	peer2 := NewConfig()
	peer1.Peers = append(peer1.Peers, *(peer2.Identities[0].Peer(peer2.Destination)))
	peer2.Peers = append(peer2.Peers, *(peer1.Identities[0].Peer(peer1.Destination)))
	_ = peer2
	msg := &Message{
		Destination: "remaphore",
		Verb:        "ping",
		Payload:     "No payload",
	}
	d, err := msg.EncodeMessage(peer1)
	if err != nil {
		t.Fatalf("EncodeMessage: %s", err)
	}
	if msg2, err := DecodeMessage(peer2, d); err != nil {
		t.Fatalf("Decode: %s", err)
	} else {
		assert.Equal(t, msg, msg2)
	}
}
