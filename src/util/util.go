package util

import (
	"bytes"
	"io/ioutil"
	"unicode"
)

func GetLines(filename string) ([]string, error) {
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	l := bytes.Split(d, []byte("\n"))
	r := make([]string, 0, len(l))
	for _, s := range l {
		q := bytes.TrimFunc(s, unicode.IsSpace)
		r = append(r, string(q))
	}
	return r, nil
}
