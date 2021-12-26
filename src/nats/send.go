package nats

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aurora-is-near/remaphore/src/protocol"
	"github.com/nats-io/nats.go"
)

var (
	ErrNoReceivers = errors.New("no known receivers")
)

type Request struct {
	Config          *protocol.Config
	SenderPublicKey protocol.Base58Bytes
	Subject         string
	Timeout         time.Duration

	conn *nats.Conn
	done context.CancelFunc
}

func (request *Request) Close() {
	if request.done != nil {
		request.done()
	}
	if request.conn != nil {
		request.conn.Close()
	}
	request.conn = nil
}

func connect(config *protocol.Config) (*nats.Conn, error) {
	url := strings.Join(config.NATSUrl, ", ")
	return nats.Connect(url,
		nats.UserCredentials(config.NATSCredsFile),
		nats.ReconnectWait(time.Second/5),
		nats.PingInterval(time.Second*3),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.MaxPingsOutstanding(3),
		nats.Timeout(time.Second*5),
	)
}

func mkSubject(parts ...string) string {
	o := make([]string, 0, len(parts))
	for _, s := range parts {
		if len(s) > 0 {
			o = append(o, s)
		}
	}
	if len(o) == 1 {
		return strings.Join([]string{o[0], "all"}, ".")
	}
	return strings.Join(o, ".")
}

func exuuid(uuid ...string) []byte {
	if uuid == nil || len(uuid) == 0 || len(uuid[0]) == 0 {
		return nil
	}
	return []byte(uuid[0])
}

func (request *Request) Send(dest, verb, msg string, uuid ...string) error {
	if dest == "" {
		dest = "**"
	}
	conn, err := connect(request.Config)
	if err != nil {
		return err
	}
	request.conn = conn
	msgOut, err := (&protocol.Message{
		SenderPublicKey: request.SenderPublicKey,
		Destination:     dest,
		RequestReply:    false,
		UUID:            exuuid(uuid...),
		Verb:            verb,
		Payload:         msg,
	}).EncodeMessage(request.Config)
	if err != nil {
		return err
	}
	subject := mkSubject(request.Config.Subject, request.Subject)
	if err := conn.Publish(subject, msgOut); err != nil {
		return err
	}
	fmt.Println(subject, string(msgOut))
	return conn.Flush()
}

func (request *Request) SendRequest(handler ReplyHandlerFunc, dest, verb, msg string, uuid ...string) error {
	var ctx context.Context
	if dest == "" {
		dest = "**"
	}
	potentialReceivers := request.Config.PotentialReceivers(dest)
	if len(potentialReceivers) == 0 {
		return ErrNoReceivers
	}
	ctx, request.done = context.WithCancel(context.Background())
	if request.Timeout > 0 {
		ctx, request.done = context.WithTimeout(ctx, request.Timeout)
	}
	defer request.done()
	conn, err := connect(request.Config)
	if err != nil {
		return err
	}
	request.conn = conn
	msgStr := &protocol.Message{
		SenderPublicKey: request.SenderPublicKey,
		Destination:     dest,
		RequestReply:    true,
		UUID:            exuuid(uuid...),
		Verb:            verb,
		Payload:         msg,
	}
	msgOut, err := msgStr.EncodeMessage(request.Config)
	if err != nil {
		return err
	}
	replySubject := mkSubject(request.Config.Subject, hex.EncodeToString(msgStr.Hash))
	sub, err := conn.SubscribeSync(replySubject)
	if err != nil {
		return err
	}
	defer func() { _ = sub.Unsubscribe() }()

	subject := mkSubject(request.Config.Subject, request.Subject)
	if err := conn.Publish(subject, msgOut); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	return request.receiveReplies(ctx, handler, sub, potentialReceivers)
}

func (request *Request) receiveReplies(ctx context.Context, handler ReplyHandlerFunc, sub *nats.Subscription, receivers protocol.Peers) error {
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err == context.DeadlineExceeded || err == context.Canceled {
			return nil
		}
		if msg != nil {
			msgStr, err := protocol.DecodeReply(request.Config, msg.Data)
			if err != nil {
				log.Printf("Message error: %s", err)
				continue
			}
			if msgStr.RequestReply {
				continue
			}
			receivers = receivers.Remove(msgStr.SenderPublicKey)
			handler(ctx, msgStr)
			if len(receivers) == 0 {
				return nil
			}
		}
	}
}
