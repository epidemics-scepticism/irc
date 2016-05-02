package main

import "strings"

type Leek struct {
	findWord map[uint16]string
	findValue map[string]uint16
}

func newLeek() *Leek {
	l := &Leek{
		findWord: make(map[uint16]string),
		findValue: make(map[string]uint16),
	}
	for c := 0; c < 65536; c++ {
		l.findWord[uint16(c)] = leekwords[uint16(c)]
		l.findValue[leekwords[uint16(c)]] = uint16(c)
	}
	return l
}

var leek = newLeek()

func (l *Leek) Encode(d []byte) string {
	var out []string
	var w []uint16
	for k, v := range d {
		if k % 2 > 0 {
			w[k/2] |= uint16(v)
		} else {
			w = append(w, uint16(v<<8 & 0xff))
		}
	}
	for _, v := range w {
		out = append(out, l.findWord[v])
	}
	return strings.Join(out, " ")
}

func (l *Leek) Decode(d string) []byte {
	var out []byte
	words := strings.Split(d, " ")
	for _, v := range words {
		i := l.findValue[v]
		out = append(out, byte(i & 0xff))
		out = append(out, byte(i>>8 & 0xff))
	}
	return out
}
