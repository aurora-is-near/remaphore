package protocol

import "testing"

func TestSubMatch(t *testing.T) {
	if !subMatch("abcde", "*") {
		t.Error("Match 0 failed")
	}
	if !subMatch("abcde", "abcde") {
		t.Error("Match 2 failed")
	}
	if subMatch("abXde", "abcde") {
		t.Error("Match 2.1 succeeded")
	}
	if !subMatch("abcde", "ab*") {
		t.Error("Match 3 failed")
	}
	if subMatch("abcde", "aX*") {
		t.Error("Match 3.1 succeeded")
	}
	if !subMatch("abcde", "*cde") {
		t.Error("Match 4 failed")
	}
	if subMatch("abXde", "*cde") {
		t.Error("Match 4.1 succeeded")
	}
	if !subMatch("abcde", "ab*de") {
		t.Error("Match 5 failed")
	}
	if subMatch("aXcde", "ab*de") {
		t.Error("Match 5.1 succeeded")
	}
	if subMatch("abcXe", "ab*de") {
		t.Error("Match 5.2 succeeded")
	}
	if subMatch("aXcXe", "ab*de") {
		t.Error("Match 5.3 succeeded")
	}
	if !subMatch("abcde", "ab*cde") {
		t.Error("Match 6 failed")
	}
	if subMatch("aXcde", "ab*cde") {
		t.Error("Match 6.1 succeeded")
	}
	if subMatch("abXde", "ab*cde") {
		t.Error("Match 6.2 succeeded")
	}
	if subMatch("aXXde", "ab*cde") {
		t.Error("Match 6.3 succeeded")
	}
	if subMatch("Abcde", "ab*cde") {
		t.Error("Match 6.4 succeeded")
	}
	if subMatch("abcdE", "ab*cde") {
		t.Error("Match 6.5 succeeded")
	}
}

func TestMatchWildcards(t *testing.T) {
	if !MatchWildcards("net.opaque.backends.us.relayer", "net.opaque.backends.us.relayer") {
		t.Error("Match 0 failed")
	}
	if !MatchWildcards("net.opaque.backends.us.relayer", "net.opaque.*.*.relayer") {
		t.Error("Match 1 failed")
	}
	if !MatchWildcards("net.opaque.backends.us.relayer", "net.opaque.**") {
		t.Error("Match 2 failed")
	}
	if MatchWildcards("net.opaque.backends.us.relayer", "net.opaque.*") {
		t.Error("Match 3 succeeded")
	}
	if MatchWildcards("net.opaque.backends.us.relayer", "net.opaque.backends") {
		t.Error("Match 4 succeeded")
	}
	if MatchWildcards("net.opaque.backends.us.relayer", "net.opaque.**.us.relayer") {
		t.Error("Match 5 succeeded")
	}
	if !MatchWildcards("net.opaque.us", "net.opaque.**") {
		t.Error("Match 6 failed")
	}
}
