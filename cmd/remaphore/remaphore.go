package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/aurora-is-near/remaphore/src/subprocess"

	"github.com/aurora-is-near/remaphore/src/protocol"

	"github.com/btcsuite/btcutil/base58"

	"github.com/aurora-is-near/remaphore/cmd/remaphore/util"
	"github.com/aurora-is-near/remaphore/src/nats"
)

// remaphore [-c configfile] [-S subject] [-m verb,...] [-o] [-u uuid] [-t duration] [-d] [-D dst] [parse.sh]
// remaphore [-c configfile] [-r|-s] [-S subject] [-m verb] [-u uuid] [-p pubkey] [-D dst] message....

var (
	clConfigFile    = "/etc/remaphore/remaphore.conf"
	clGenConfigFile bool
	clSubject       string
	clVerb          = ""
	clVerbParsed    []string
	clOnce          bool
	clUUID          = ""
	clTimeout       time.Duration
	clRequestReply  bool
	clSendOnly      bool
	clPubkey        string
	clPubkeyParsed  []byte
	clNoFilterDest  bool
	clMatchDest     string
	clRemainder     []string
	clMessage       string
)

func init() {
	flag.BoolVar(&clGenConfigFile, "C", clGenConfigFile, "Print example config file")
	flag.StringVar(&clConfigFile, "c", clConfigFile, "Path to config file")
	flag.StringVar(&clSubject, "S", clSubject, "Subject to communicate on")
	flag.StringVar(&clVerb, "v", clVerb, "-v <verb>[,verb...]: Verb to send or match filter for")
	flag.BoolVar(&clOnce, "o", clOnce, "Exit after one matching message received")
	flag.StringVar(&clUUID, "u", clUUID, "UUID to send/filter for")
	flag.DurationVar(&clTimeout, "t", clTimeout, "Timeout for operation")
	flag.BoolVar(&clRequestReply, "r", clRequestReply, "Request reply to message")
	flag.BoolVar(&clSendOnly, "s", clSendOnly, "Send and forget")
	flag.StringVar(&clPubkey, "p", clPubkey, "Use public key for sending or match for it")
	flag.BoolVar(&clNoFilterDest, "d", clNoFilterDest, "Do not match for destination")
	flag.StringVar(&clMatchDest, "D", clMatchDest, "Specify destination to match")
	_ = clRemainder
	_ = clVerbParsed
}

func parseArgs() {
	flag.Parse()
	if clGenConfigFile {
		util.PrintConfig()
	}
	clRemainder = flag.Args()
	clVerbParsed = util.CleanStrings(strings.Split(clVerb, ",")...)
	clMessage = strings.Join(clRemainder, " ")
	if clRequestReply && clSendOnly {
		util.ExitError(2, "-r and -s are mutually exclusive")
	}
	if clRequestReply || clSendOnly {
		if len(clMessage) == 0 {
			util.ExitError(2, "-r and -s require a message to send")
		}
		if len(clVerbParsed) == 0 {
			util.ExitError(2, "-r and -s require a verb to send")
		}
	}
	if clNoFilterDest && len(clMatchDest) > 0 {
		util.ExitError(2, "-d and -D are mutually exclusive")
	}
	if len(clPubkey) > 0 {
		clPubkeyParsed = base58.Decode(clPubkey)
		if clPubkeyParsed == nil || len(clPubkeyParsed) != ed25519.PublicKeySize {
			util.ExitError(2, "ERROR: Given public key does not parse")
		}
	}
}

func main() {
	var received bool
	var err error
	parseArgs()
	request := &nats.Request{
		Config:          util.GetConfig(clConfigFile),
		SenderPublicKey: clPubkeyParsed,
		Subject:         clSubject,
		Timeout:         clTimeout,
	}
	switch {
	case clSendOnly:
		received = true
		err = request.Send(clMatchDest, clVerbParsed[0], strings.Join(clRemainder, " "), clUUID)
	case clRequestReply:
		printChan := make(chan *protocol.Message, 10)
		closeChan := make(chan struct{}, 1)
		go func() {
			sep := hex.EncodeToString(protocol.RandomBytes(16))
			for m := range printChan {
				received = true
				payload := strings.TrimFunc(m.Payload, unicode.IsSpace)

				if strings.Contains(payload, "\n") {
					util.StdOut("--> %s\n%s,%s\n--< %s\n", sep, request.Config.Peers.Destination(m.SenderPublicKey), payload, sep)
				} else {
					util.StdOut("%s,%s\n", request.Config.Peers.Destination(m.SenderPublicKey), payload)
				}
			}
			close(closeChan)
		}()
		handler := func(ctx context.Context, message *protocol.Message) {
			printChan <- message
		}
		err = request.SendRequest(handler, clMatchDest, clVerbParsed[0], strings.Join(clRemainder, " "), clUUID)
		close(printChan)
		<-closeChan
	default:
		var matches []protocol.MsgMatch
		if len(clUUID) > 0 {
			matches = append(matches, protocol.MatchUUID([]byte(clUUID)))
			clOnce = true
		}
		if len(clVerbParsed) > 0 {
			matches = append(matches, protocol.MatchVerb(clVerbParsed...))
		}
		if clPubkeyParsed != nil && len(clPubkeyParsed) > 0 {
			matches = append(matches, protocol.MatchSenderPublicKey(clPubkeyParsed))
		}
		if !clNoFilterDest {
			matches = append(matches, protocol.MatchDestination(clMatchDest))
		}
		handler := func(ctx context.Context, message *protocol.Message, replyFunc nats.ReplyFunc) {
			log.Println("Incoming message")
			var out string
			var err error
			received = true
			if len(clRemainder) > 0 {
				out, err = subprocess.Exec(ctx, request.Config, clRemainder, message)
				if err != nil {
					log.Printf("ERROR: %s", err)
				}
			} else {
				out = "NO_DATA"
			}
			if replyFunc != nil {
				resp := new(protocol.Message)
				resp.Payload = out
				if err := replyFunc(resp); err != nil {
					log.Printf("Reply error: %s\n", err)
				}
			}
			if clOnce {
				request.Close()
			}
		}
		err = request.Receive(handler, matches...)
	}
	if err != nil {
		util.ExitError(3, "ERROR: %s", err)
	}
	if !received {
		os.Exit(1)
	}
}
