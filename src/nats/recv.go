package nats

import (
	"context"
	"encoding/hex"
	"log"

	"github.com/aurora-is-near/remaphore/src/protocol"
)

type ReplyFunc func(message *protocol.Message) error
type HandlerFunc func(ctx context.Context, message *protocol.Message, replyFunc ReplyFunc)
type ReplyHandlerFunc func(ctx context.Context, message *protocol.Message)

func (request *Request) Receive(handler HandlerFunc, matches ...protocol.MsgMatch) error {
	var ctx context.Context
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
	subject := mkSubject(request.Config.Subject, request.Subject)
	sub, err := conn.SubscribeSync(subject)
	if err != nil {
		return err
	}
	defer func() { _ = sub.Unsubscribe() }()
	log.Println("Ready")
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err == context.DeadlineExceeded || err == context.Canceled {
			return nil
		}
		if msg != nil {
			msgStr, err := protocol.DecodeMessage(request.Config, msg.Data)
			if err != nil {
				log.Printf("Message error: %s", err)
				continue
			}
			// if request.Config.IsSelf(msgStr.SenderPublicKey) {
			// 	continue
			// }
			if msgStr.Match(request.Config, matches...) && handler != nil {
				var reply ReplyFunc
				if msgStr.RequestReply {
					replySubject := mkSubject(request.Config.Subject, hex.EncodeToString(msgStr.Hash))
					reply = func(msg *protocol.Message) error {
						msg.Verb = "reply"
						msgO, err := msg.EncodeReply(request.Config)
						if err != nil {
							return err
						}
						if err := conn.Publish(replySubject, msgO); err != nil {
							return err
						}
						return conn.Flush()
					}
				}
				handler(ctx, msgStr, reply)
			}
		}
	}
}
