/*
   Copyright (C) 2016 cacahuatl < cacahuatl at autistici dot org >

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
package main

import "strings"

type Leek struct {
	findWord  map[uint16]string
	findValue map[string]uint16
}

func newLeek() *Leek {
	l := &Leek{
		findWord:  make(map[uint16]string),
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
		sv := uint16(v)
		if k%2 > 0 {
			w[k/2] |= sv
		} else {
			w = append(w, sv<<8&0xff)
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
		out = append(out, byte(i&0xff))
		out = append(out, byte(i>>8&0xff))
	}
	return out
}
