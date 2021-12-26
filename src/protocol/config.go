package protocol

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
)

const (
	defaultNATSUrl     = "nats://natsserver:4222"
	defaultCredsFile   = "/path/to/credentials/file"
	defaultSubject     = "remaphore"
	defaultDestination = "all"
	AllowedClockSkew   = time.Second * 5
)

type Peers []Peer
type Identities []Identity

type Config struct {
	NATSUrl          []string
	NATSCredsFile    string
	Subject          string
	DefaultKey       Base58Bytes
	Destination      string
	AllowedClockSkew time.Duration
	Identities       Identities
	Peers            Peers
}

func (config *Config) String() string {
	lines := make([]string, 0, 4+5+len(config.NATSUrl)+len(config.Identities)+len(config.Peers))
	for _, l := range config.NATSUrl {
		lines = append(lines, fmt.Sprintf("server: %s", l))
	}
	lines = append(lines, fmt.Sprintf("credentials: %s", config.NATSCredsFile))
	lines = append(lines, fmt.Sprintf("subject: %s", config.Subject))
	lines = append(lines, fmt.Sprintf("default_identity: %s", base58.Encode(config.DefaultKey)))
	lines = append(lines, fmt.Sprintf("destination: %s", config.Destination))
	lines = append(lines, fmt.Sprintf("allow_skew: %v", config.AllowedClockSkew))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprint("[ Identities ]"))
	for _, i := range config.Identities {
		lines = append(lines, i.String())
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprint("[ Peers ]"))
	for _, i := range config.Peers {
		lines = append(lines, i.String())
	}
	return strings.Join(lines, "\n")
}

type Identity struct {
	PublicKey   Base58Bytes
	PrivateKey  Base58Bytes
	Permissions []string
}

func (identity *Identity) String() string {
	return fmt.Sprintf("%s %s [%s]", base58.Encode(identity.PublicKey), base58.Encode(identity.PrivateKey), strings.Join(identity.Permissions, ", "))
}

type Peer struct {
	PublicKey   Base58Bytes
	Destination string
	Permissions []string
}

func (peer *Peer) String() string {
	return fmt.Sprintf("%s %s [%s]", peer.Destination, base58.Encode(peer.PublicKey), strings.Join(peer.Permissions, ", "))
}

func NewConfig() *Config {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	ret := &Config{
		NATSUrl:          []string{defaultNATSUrl},
		NATSCredsFile:    defaultCredsFile,
		Subject:          defaultSubject,
		AllowedClockSkew: AllowedClockSkew,
		DefaultKey:       Base58Bytes(publicKey),
		Destination:      defaultDestination,
		Identities: Identities{{
			PublicKey:   Base58Bytes(publicKey),
			PrivateKey:  Base58Bytes(privateKey),
			Permissions: []string{"ping"},
		}},
	}
	return ret
}

func (config *Config) IsSelf(publicKey []byte) bool {
	if bytes.Equal(config.DefaultKey, publicKey) {
		return true
	}
	for _, v := range config.Identities {
		if bytes.Equal(v.PublicKey, publicKey) {
			return true
		}
	}
	return false
}

func (config *Config) PrivateKey(publicKey []byte, verb ...string) []byte {
	if publicKey == nil || len(publicKey) != ed25519.PublicKeySize {
		return nil
	}
	for _, v := range config.Identities {
		if bytes.Equal(v.PublicKey, publicKey) {
			if v.HasPermission(verb...) {
				return v.PrivateKey
			}
			return nil
		}
	}
	return nil
}

func (peers Peers) Known(publicKey []byte, verb ...string) bool {
	if publicKey == nil || len(publicKey) != ed25519.PublicKeySize {
		return false
	}
	for _, v := range peers {
		if bytes.Equal(v.PublicKey, publicKey) {
			return v.HasPermission(verb...)
		}
	}
	return false
}

func (config *Config) PotentialReceivers(destination string) Peers {
	ret := make(Peers, 0, len(config.Peers))
	for _, rec := range config.Peers {
		if MatchWildcards(rec.Destination, destination) {
			ret = append(ret, *rec.Copy())
		}
	}
	return ret
}

func testPermission(permissions []string, verb ...string) bool {
	if verb == nil || len(verb) == 0 {
		return true
	}
	tVerb := verb[0]
	for _, v := range permissions {
		if v == "*" {
			return true
		}
		if tVerb == v {
			return true
		}
	}
	return false
}

func (identity *Identity) HasPermission(verb ...string) bool {
	return testPermission(identity.Permissions, verb...)
}

func (identity *Identity) Peer(destination string) *Peer {
	return &Peer{
		PublicKey:   identity.PublicKey,
		Permissions: identity.Permissions,
		Destination: destination,
	}
}

func copyStringSlice(d []string) []string {
	r := make([]string, len(d))
	copy(r, d)
	return r
}

func copySlice(d []byte) []byte {
	r := make([]byte, len(d))
	copy(r, d)
	return r
}

func (peer *Peer) Copy() *Peer {
	return &Peer{
		PublicKey:   copySlice(peer.PublicKey),
		Destination: peer.Destination,
		Permissions: copyStringSlice(peer.Permissions),
	}
}

func (peer *Peer) HasPermission(verb ...string) bool {
	return testPermission(peer.Permissions, verb...)
}

func (peers Peers) Remove(peer []byte) Peers {
	for i, p := range peers {
		if bytes.Equal(p.PublicKey, peer) {
			peers[i] = peers[len(peers)-1]
			return peers[:len(peers)-1]
		}
	}
	return peers
}

func (peers Peers) Destination(pubkey []byte) string {
	for _, p := range peers {
		if bytes.Equal(p.PublicKey, pubkey) {
			return p.Destination
		}
	}
	return ""
}
