package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Base58TT struct {
	Val Base58Bytes
}

func testRemarshal(t *testing.T, td *Base58TT) {
	d, err := json.Marshal(td)
	if err != nil {
		t.Fatalf("Marshal: %s", err)
	}
	td2 := new(Base58TT)
	if err := json.Unmarshal(d, td2); err != nil {
		t.Fatalf("Unmarshal: %s", err)
	}
	assert.Equal(t, td.Val, td2.Val)
}

func TestBase58Bytes_JSON(t *testing.T) {
	td := new(Base58TT)
	testRemarshal(t, td)
	td = &Base58TT{
		Val: []byte(""),
	}
	testRemarshal(t, td)
	td = &Base58TT{
		Val: []byte("testdataofnoconsequence"),
	}
	testRemarshal(t, td)
}
