package protocol

import "bytes"

type MsgMatch func(*Config, *Message) bool

func (msg *Message) Match(c *Config, matches ...MsgMatch) bool {
	for _, m := range matches {
		if !m(c, msg) {
			return false
		}
	}
	return true
}

func MatchVerb(verb ...string) MsgMatch {
	return func(c *Config, m *Message) bool {
		for _, s := range verb {
			if s == m.Verb {
				return true
			}
		}
		return false
	}
}

func MatchUUID(uuid []byte) MsgMatch {
	tuuid := NewUUID(uuid)
	return func(c *Config, m *Message) bool {
		_ = c
		return bytes.Equal(tuuid, m.UUID)
	}
}

func MatchDestination(destination ...string) MsgMatch {
	if destination != nil && len(destination) > 0 && len(destination[0]) > 0 {
		return func(c *Config, m *Message) bool {
			_ = c
			return m.Destination == destination[0]
		}
	}
	return func(c *Config, m *Message) bool {
		return MatchWildcards(c.Destination, m.Destination)
	}
}

func MatchSenderPublicKey(publicKey []byte) MsgMatch {
	return func(c *Config, m *Message) bool {
		return bytes.Equal(m.SenderPublicKey, publicKey)
	}
}
