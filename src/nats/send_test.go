package nats

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/aurora-is-near/remaphore/src/subprocess"

	"github.com/aurora-is-near/remaphore/src/util"

	"github.com/aurora-is-near/remaphore/src/protocol"
)

func TestSendRequest_Send(t *testing.T) {
	servers, err := util.GetLines("../../tests/servers.txt")
	if err != nil {
		t.Fatalf("GetLines: %s", err)
	}
	req := &Request{
		Config: protocol.NewConfig(),
	}
	req.Config.NATSUrl = servers
	req.Config.NATSCredsFile = "../../tests/nats.creds"

	rec := &Request{
		Config:  protocol.NewConfig(),
		Timeout: time.Second * 2,
	}
	rec.Config.NATSUrl = servers
	rec.Config.NATSCredsFile = "../../tests/nats.creds"
	rec.Config.Peers = append(rec.Config.Peers, *(req.Config.Identities[0].Peer(req.Config.Destination)))
	req.Config.Peers = append(req.Config.Peers, *(rec.Config.Identities[0].Peer(rec.Config.Destination)))
	defer rec.Close()
	defer req.Close()
	c := make(chan struct{})
	handler := func(ctx context.Context, message *protocol.Message, reply ReplyFunc) {
		_ = reply
		log.Printf("%s> %s\n", message.Verb, message.Payload)
	}

	go func() {
		defer func() { close(c) }()
		if err := rec.Receive(handler); err != nil {
			log.Printf("Receive: %s\n", err)
		}
	}()
	time.Sleep(time.Second / 2)
	if err := req.Send("", "update", "12345", ""); err != nil {
		t.Fatalf("Send: %s", err)
	}
	<-c
}

func TestRequest_SendRequest(t *testing.T) {
	servers, err := util.GetLines("../../tests/servers.txt")
	if err != nil {
		t.Fatalf("GetLines: %s", err)
	}
	req := &Request{
		Config:  protocol.NewConfig(),
		Timeout: time.Second * 2,
	}
	req.Config.NATSUrl = servers
	req.Config.NATSCredsFile = "../../tests/nats.creds"
	req.Config.Destination = "net.crypto.internal.us.001"

	rec := &Request{
		Config:  protocol.NewConfig(),
		Timeout: time.Second * 2,
	}
	rec.Config.NATSUrl = servers
	rec.Config.NATSCredsFile = "../../tests/nats.creds"
	rec.Config.Destination = "net.crypto.internal.us.002"
	rec.Config.Peers = append(rec.Config.Peers, *(req.Config.Identities[0].Peer(req.Config.Destination)))
	req.Config.Peers = append(req.Config.Peers, *(rec.Config.Identities[0].Peer(rec.Config.Destination)))
	defer rec.Close()
	defer req.Close()
	c := make(chan struct{})
	handler := func(ctx context.Context, message *protocol.Message, reply ReplyFunc) {
		log.Printf("%s> %s\n", message.Verb, message.Payload)

		if reply != nil {
			resp := new(protocol.Message) // &protocol.Message{Payload: "this is a reply"}
			op, err := subprocess.Exec(ctx, rec.Config, []string{"../../tests/test.sh"}, message)
			if err != nil {
				log.Printf("Exec: %s", err)
			}
			resp.Payload = op
			if err := reply(resp); err != nil {
				log.Printf("Reply error: %s\n", err)
			}
		}
	}
	replyHandler := func(ctx context.Context, message *protocol.Message) {
		log.Printf("REPLY: %s> %s\n", message.Verb, message.Payload)
	}

	go func() {
		defer func() { close(c) }()
		if err := rec.Receive(handler); err != nil {
			log.Printf("Receive: %s\n", err)
		}
	}()
	time.Sleep(time.Second / 2)
	if err := req.SendRequest(replyHandler, "net.crypto.internal.**", "update", "12345", ""); err != nil {
		t.Fatalf("Send: %s", err)
	}
	<-c
}
