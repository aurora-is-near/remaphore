package protocol

import (
	"encoding/json"
	"reflect"

	"github.com/btcsuite/btcutil/base58"
)

type Base58Bytes []byte

func (b58b *Base58Bytes) MarshalJSON() ([]byte, error) {
	if *b58b == nil {
		return []byte("null"), nil
	}
	return []byte("\"" + base58.Encode(*b58b) + "\""), nil
}

func (b58b *Base58Bytes) UnmarshalJSON(d []byte) error {
	if d == nil || string(d) == "null" {
		return nil
	}
	if len(d) < 2 {
		return &json.InvalidUnmarshalError{Type: reflect.TypeOf(b58b)}
	}
	if d[0] != '"' || d[len(d)-1] != '"' {
		return &json.InvalidUnmarshalError{Type: reflect.TypeOf(b58b)}
	}
	*b58b = base58.Decode(string(d[1 : len(d)-1]))
	return nil
}
