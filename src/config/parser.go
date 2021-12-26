package config

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/aurora-is-near/remaphore/src/protocol"
	"github.com/btcsuite/btcutil/base58"
)

func cleanLine(s string) string {
	s = strings.TrimFunc(s, unicode.IsSpace)
	if p := strings.Index(s, "#"); p >= 0 {
		return s[:p]
	}
	return s
}

const (
	stGeneral = iota
	stIdentity
	stPeer
)

func splitValue(s string) (key, value string) {
	if p := strings.Index(s, ":"); p > 0 {
		key = strings.ToLower(cleanLine(s[:p]))
		value = cleanLine(s[p+1:])
		return
	}
	return "", ""
}

func setGeneralValue(c *protocol.Config, key, value string) error {
	switch key {
	case "server":
		c.NATSUrl = append(c.NATSUrl, value)
	case "credentials":
		c.NATSCredsFile = value
	case "subject":
		c.Subject = value
	case "destination":
		c.Destination = value
	case "default_identity":
		c.DefaultKey = base58.Decode(value)
		if len(c.DefaultKey) != ed25519.PublicKeySize {
			c.DefaultKey = nil
			return fmt.Errorf("invalid public key: %s", value)
		}
	case "allow_skew":
		v, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		c.AllowedClockSkew = v
	}
	return nil
}

func ParseConfig(d []byte) (*protocol.Config, error) {
	state := stGeneral
	_ = state
	ret := new(protocol.Config)
	buf := bytes.NewBuffer(d)
	for l, _ := buf.ReadString('\n'); len(l) > 0; l, _ = buf.ReadString('\n') {
		l = cleanLine(l)
		if len(l) == 0 {
			continue
		}
		if l[0] == '[' && l[len(l)-1] == ']' {
			sectionName := strings.ToLower(cleanLine(l[1 : len(l)-1]))
			switch sectionName {
			case "identities":
				state = stIdentity
				continue
			case "peers":
				state = stPeer
				continue
			default:
				continue
			}
		}
		switch state {
		case stGeneral:
			k, v := splitValue(l)
			if len(k) == 0 || len(v) == 0 {
				continue
			}
			if err := setGeneralValue(ret, k, v); err != nil {
				return nil, err
			}
		case stIdentity:
			i, err := parseIdentity(l)
			if err != nil {
				return nil, err
			}
			ret.Identities = append(ret.Identities, *i)
		case stPeer:
			p, err := parsePeer(l)
			if err != nil {
				return nil, err
			}
			ret.Peers = append(ret.Peers, *p)
		}
	}
	if err := validateConfig(ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func parsePeer(s string) (*protocol.Peer, error) {
	ret := new(protocol.Peer)
	f := strings.SplitN(s, " ", 3)
	if len(f) != 3 {
		return nil, fmt.Errorf("bad format: \"%s\"", s)
	}
	ret.Destination = f[0]
	pubkey := base58.Decode(f[1])
	if len(pubkey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("bad public key: \"%s\"", f[0])
	}
	ret.PublicKey = pubkey
	permissions, err := parsePermissions(f[2])
	if err != nil {
		return nil, err
	}
	ret.Permissions = permissions
	return ret, nil
}

func validateConfig(c *protocol.Config) error {
	if len(c.NATSUrl) == 0 {
		return fmt.Errorf("no nats servers configured")
	}
	if len(c.NATSCredsFile) == 0 {
		return fmt.Errorf("no nats credentials configured")
	}
	if len(c.Identities) == 0 {
		return fmt.Errorf("no identities configured")
	}
	if len(c.Destination) == 0 {
		return fmt.Errorf("no destination configured")
	}
	if c.AllowedClockSkew == 0 {
		c.AllowedClockSkew = time.Second * 3
	}
	if c.Subject == "" {
		c.Subject = "remaphore"
	}
	if c.DefaultKey == nil {
		c.DefaultKey = c.Identities[0].PublicKey
	}
	return nil
}

func parsePermissions(s string) ([]string, error) {
	if s[0] == '[' && s[len(s)-1] == ']' {
		permissions := strings.ToLower(cleanLine(s[1 : len(s)-1]))
		f := strings.Split(permissions, ",")
		r := make([]string, 0, len(f))
		for _, p := range f {
			p = cleanLine(p)
			if len(p) == 0 {
				continue
			}
			r = append(r, p)
		}
		return r, nil
	}
	return nil, fmt.Errorf("not valid permissions: %s", s)
}

func parseIdentity(s string) (identity *protocol.Identity, err error) {
	ret := new(protocol.Identity)
	f := strings.SplitN(s, " ", 3)
	if len(f) != 3 {
		return nil, fmt.Errorf("bad format: \"%s\"", s)
	}
	pubkey := base58.Decode(f[0])
	if len(pubkey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("bad public key: \"%s\"", f[0])
	}
	privkey := base58.Decode(f[1])
	if len(privkey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("bad private key: \"%s\"", f[1])
	}
	permissions, err := parsePermissions(f[2])
	if err != nil {
		return nil, err
	}
	ret.PublicKey = pubkey
	ret.PrivateKey = privkey
	ret.Permissions = permissions
	return ret, nil
}
