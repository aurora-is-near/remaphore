package protocol

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
)

var (
	ErrDestinationBadChar = errors.New("destination contains forbidden character")
	ErrVerbBadChar        = errors.New("verb contains forbidden character")
	ErrNoPrivateKey       = errors.New("no private key with sender permission found")
	ErrFormat             = errors.New("message format corrupt")
	ErrSignature          = errors.New("signature corrupt")
	ErrPeerPermission     = errors.New("peer key or permission not known")
	ErrClockSkew          = errors.New("message outside of time window")
)

const sepChar = ","
const uuidLen = 12

var requestCode = []byte("Q")

type Message struct {
	SenderPublicKey Base58Bytes
	SenderSignature Base58Bytes
	Destination     string
	RequestReply    bool
	SendTimeNano    int64
	UUID            []byte
	Verb            string
	Payload         string
	Hash            []byte
}

func RandomBytes(l int) []byte {
	d := make([]byte, l)
	if _, err := io.ReadFull(rand.Reader, d); err != nil {
		panic(err)
	}
	return d
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func DecodeMessage(c *Config, msg []byte) (*Message, error) {
	return decodeMessage(c, msg, false)
}

func DecodeReply(c *Config, msg []byte) (*Message, error) {
	return decodeMessage(c, msg, true)
}

func decodeMessage(c *Config, msg []byte, isReply bool) (*Message, error) {
	parts := bytes.SplitN(msg, []byte(sepChar), 3)
	if len(parts) != 3 {
		return nil, ErrFormat
	}
	preMsg := parts[2]
	parts2 := bytes.SplitN(preMsg, []byte(sepChar), 6)
	if len(parts2) != 6 {
		return nil, ErrFormat
	}
	sendTimeNano, err := strconv.ParseInt(string(parts2[1]), 16, 64)
	if err != nil {
		return nil, err
	}
	uuid, err := hex.DecodeString(string(parts2[2]))
	if err != nil {
		return nil, err
	}
	ret := &Message{
		SenderPublicKey: base58.Decode(string(parts[0])),
		SenderSignature: base58.Decode(string(parts[1])),
		Destination:     string(parts2[0]),
		SendTimeNano:    sendTimeNano,
		UUID:            uuid,
		Verb:            string(parts2[3]),
		RequestReply:    bytes.Equal(requestCode, parts2[4]),
		Payload:         string(parts2[5]),
	}
	ret.Hash = sha256Hash(msg)
	if err := ret.verifyPerms(c, isReply); err != nil {
		return ret, err
	}
	// Check clockskew
	if !ret.verifyClockSkew(c) {
		return ret, ErrClockSkew
	}
	return ret, nil
}

func (msg *Message) verifyClockSkew(c *Config) bool {
	now := time.Now().UnixNano()
	if c.AllowedClockSkew < time.Duration(maxInt64(now, msg.SendTimeNano)-minInt64(now, msg.SendTimeNano)) {
		return false
	}
	return true
}

func (msg *Message) verifyPerms(c *Config, isReply bool) error {
	// Check if pubkey known && check if permission
	if isReply {
		if !c.Peers.Known(msg.SenderPublicKey) {
			return ErrPeerPermission
		}
		msg.RequestReply = false
	} else {
		if !c.Peers.Known(msg.SenderPublicKey, msg.Verb) {
			return ErrPeerPermission
		}
	}
	// Check signature
	if !ed25519.Verify(ed25519.PublicKey(msg.SenderPublicKey), msg.preMsg(), msg.SenderSignature) {
		return ErrSignature
	}
	return nil
}

func (msg *Message) requestReplyField() []byte {
	if msg.RequestReply {
		return requestCode
	}
	return []byte("_")
}

func (msg *Message) preMsg() []byte {
	return bytes.Join([][]byte{
		[]byte(msg.Destination),
		[]byte(strconv.FormatInt(msg.SendTimeNano, 16)),
		[]byte(hex.EncodeToString(msg.UUID)),
		[]byte(msg.Verb),
		msg.requestReplyField(),
		[]byte(msg.Payload),
	}, []byte(sepChar))
}

func (msg *Message) EncodeMessage(c *Config) ([]byte, error) {
	return msg.encode(c, false)
}

func (msg *Message) EncodeReply(c *Config) ([]byte, error) {
	return msg.encode(c, true)
}

func (msg *Message) encode(c *Config, isReply bool) ([]byte, error) {
	var privateKey []byte
	if strings.Contains(msg.Destination, sepChar) {
		return nil, ErrDestinationBadChar
	}
	if strings.Contains(msg.Verb, sepChar) {
		return nil, ErrVerbBadChar
	}
	if msg.SenderPublicKey == nil || len(msg.SenderPublicKey) == 0 {
		msg.SenderPublicKey = c.DefaultKey
	}
	if isReply {
		privateKey = c.PrivateKey(msg.SenderPublicKey)
		msg.RequestReply = false
	} else {
		privateKey = c.PrivateKey(msg.SenderPublicKey, msg.Verb)
	}
	if privateKey == nil {
		return nil, ErrNoPrivateKey
	}
	msg.SendTimeNano = time.Now().UnixNano()
	if msg.UUID == nil || len(msg.UUID) == 0 {
		msg.UUID = NewUUID()
	}
	preMsg := msg.preMsg()
	msg.SenderSignature = ed25519.Sign(ed25519.PrivateKey(privateKey), preMsg)
	encodedMsg := bytes.Join([][]byte{
		[]byte(base58.Encode(msg.SenderPublicKey)),
		[]byte(base58.Encode(msg.SenderSignature)),
		preMsg,
	}, []byte(sepChar))
	msg.Hash = sha256Hash(encodedMsg)
	return encodedMsg, nil
}
