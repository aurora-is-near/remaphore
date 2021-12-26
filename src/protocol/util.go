package protocol

import (
	"crypto/sha256"
	"strings"
)

const destSeparator = "."

// NewUUID creates a new UUID. If the given one is too long, it will be hashed.
// If none is given, a random one is created.
func NewUUID(uuid ...[]byte) []byte {
	if uuid == nil || len(uuid) == 0 || uuid[0] == nil || len(uuid) == 0 {
		return RandomBytes(uuidLen)
	}
	if len(uuid[0]) <= uuidLen {
		return uuid[0]
	}
	return sha256Hash(uuid[0])[0:uuidLen]
}

func sha256Hash(d []byte) []byte {
	h := sha256.New()
	h.Write(d)
	return h.Sum(nil)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func subMatch(s, p string) bool {
	if s == p {
		return true
	}
	if p == "*" {
		return true
	}
	if n := strings.Index(p, "*"); n > -1 {
		pre := p[:n]
		post := p[n+1:]
		if len(pre) > 0 && pre != s[:len(pre)] {
			return false
		}
		if len(post) > 0 && post != s[len(s)-len(post):] {
			return false
		}
		return true
	}
	return false
}

// MatchWildcards returns true if s matches pattern
// '*' matches between dots may only occur once between dots.
// '**' matches beyond dots. May only appear at end of pattern.
func MatchWildcards(s, pattern string) bool {
	var lastMatch int
	if n := strings.Index(s, "*"); n > -1 {
		return false
	}
	sf := strings.Split(s, destSeparator)
	pf := strings.Split(pattern, destSeparator)
	if len(sf) < len(pf) {
		return false
	}
	if len(sf) >= len(pf) {
		lastMatch = len(pf) - 1
	}
	for i := 0; i < minInt(len(sf), len(pf)); i++ {
		if pf[i] == "**" {
			return i == lastMatch
		}
		if !subMatch(sf[i], pf[i]) {
			return false
		}
	}
	if len(sf) > len(pf) {
		return false
	}
	return true
}
