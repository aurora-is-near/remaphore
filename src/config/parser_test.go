package config

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFile(t *testing.T) {
	d, err := ioutil.ReadFile("../../tests/config.file")
	if err != nil {
		t.Fatalf("")
	}
	config, err := ParseConfig(d)
	if err != nil {
		t.Errorf("Parse: %s", err)
	}
	config1, err := ParseConfig([]byte(fmt.Sprintf("%s", config)))
	if err != nil {
		t.Errorf("Parse: %s", err)
	}
	assert.Equal(t, config, config1)
}
