package main

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"strings"
)
// http://www.ruf.rice.edu/~pound/lc.py
// https://git-tails.immerda.ch/tails/plain/config/chroot_local-includes/usr/share/amnesia/firstnames.txt
type pseudoLanguage struct {
	parsed bool
	inits  map[string][]byte
	pairs  map[string][]byte
}

func newPseudoLanguage() pseudoLanguage {
	return pseudoLanguage{
		false,
		make(map[string][]byte),
		make(map[string][]byte),
	}
}

func (p pseudoLanguage) parse(i []string) {
	for _, v := range i {
		v += " "
		if len(v) > 3 {
			p.inits[v[0:2]] = append(p.inits[v[0:2]], v[2])
		}
		for pos := 0; pos < len(v)-2; pos++ {
			p.pairs[v[pos:pos+2]] = append(p.pairs[v[pos:pos+2]], v[pos+2])
		}
	}
	p.parsed = true
}

func (p pseudoLanguage) randomInit() string {
	if p.parsed == false {
		p.parse(nicks)
	}
	l := len(p.inits)
	r := properRand(l)
	var s string
	i := 0
	for k := range p.inits {
		if i == r {
			s = k
			break
		} else {
			i += 1
		}
	}
	return s
}

func (p pseudoLanguage) randomPair(k string) byte {
	if p.parsed == false {
		p.parse(nicks)
	}
	l := len(p.pairs[k])
	r := properRand(l)
	return p.pairs[k][r]
}

func (p pseudoLanguage) generate() string {
	if p.parsed == false {
		p.parse(nicks)
	}
	for {
		word := []byte(p.randomInit())
		for bytes.Contains(word, []byte{' '}) == false {
			word = append(word[:], p.randomPair(string(word[len(word)-2:])))
		}
		word = bytes.Trim(word, "\r\n\t\v ")
		if len(word) >= 4 && len(word) <= 10 {
			return string(word)
		}
	}
}

func leetNick(nick string) string {
	leet := func(r rune) rune {
		switch r {
		case 'e':
			return '3'
		case 'i':
			return '1'
		case 'o':
			return '0'
		default:
			return r
		}
	}
	return nick[0:1] + strings.Map(leet, nick[1:])
}

var pl = newPseudoLanguage()

func generateNick() string {
	nick := pl.generate()
	if prob(90) {
		nick = strings.ToLower(nick)
	}
	if prob(5) {
		if prob(50) {
			nick = nick + "_"
		} else {
			nick = nick + "^"
		}
	}
	if prob(5) {
		nick = leetNick(nick)
	}
	return nick
}

func properRand(max int) int {
	bmax := big.NewInt(int64(max))
	s, e := rand.Int(rand.Reader, bmax)
	if e != nil {
		panic("no random")
	}
	return int(s.Int64())
}

func prob(p int) bool {
	return bool(p > properRand(100))
}
