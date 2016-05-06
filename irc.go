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

import (
	"bufio"
	"net"
	"strings"
	"time"
)

type Msg struct {
	timestamp, nick, user, host, cmd, rcpt, content, args string
	enc                                                   bool
}

func split(s, d string) (string, string) {
	arr := strings.SplitN(s, d, 2)
	if len(arr) == 2 {
		return arr[0], arr[1]
	} else if len(arr) == 1 {
		return arr[0], ""
	} else {
		return "", ""
	}
}

func Parse(line string) *Msg {
	line = strings.TrimRight(line, "\r\n")
	m := new(Msg)
	m.timestamp = time.Now().UTC().Format("15:04")
	if strings.HasPrefix(line, ":") {
		line = line[1:]
		m.host, line = split(line, " ")
		m.nick, m.host = split(m.host, "!")
		m.user, m.host = split(m.host, "@")
	}
	line, m.content = split(line, " :")
	m.cmd, line = split(line, " ")
	m.rcpt, line = split(line, " ")
	m.args = line
	return m
}

func parseLoop() {
	i := bufio.NewReader(conn)
	for {
		if s, e := i.ReadString('\n'); e != nil {
			PrintError(e)
			return
		} else {
			m := Parse(s)
			if _, ok := IgnoreMap[m.nick]; ok {
				continue
			}
			if m.cmd == "PRIVMSG" && strings.HasPrefix(m.rcpt, "#") == false {
				OtrRecv(m)
				out <- m
			} else if m.cmd == "PING" {
				Raw("PONG :" + m.content)
			} else {
				out <- m
			}
		}
	}
}

func sendLoop() {
	for {
		s := <-send
		_, e := conn.Write([]byte(s + "\r\n"))
		if e != nil {
			PrintError(e)
			return
		}
	}
}

var (
	out  chan *Msg
	send chan string
	conn net.Conn
)

func Ctcp(rcpt, msg string) {
	send <- "PRIVMSG " + rcpt + " :\x01" + msg + "\x01"
}

func SendTo(rcpt, msg string) {
	if len(msg) < 1 {
		return
	}
	if strings.HasPrefix(rcpt, "#") {
		send <- "PRIVMSG " + rcpt + " :" + msg
	} else {
		OtrSend(rcpt, msg)
	}
}

func Join(channel string) {
	send <- "JOIN " + channel
}

func Register(nick string) {
	send <- "USER " + nick + " * localhost :" + nick
	send <- "NICK " + nick
}

func NewNick(nick string) {
	send <- "NICK " + nick
}

func Part(channel string) {
	send <- "PART " + channel
}

func Quit(reason string) {
	for rcpt := range OTR.conv {
		OtrEnd(rcpt)
	}
	send <- "QUIT :Leaving."
}

func Raw(raw string) {
	send <- raw
}

func Init(server, proxy string, ssl bool) (chan *Msg, error) {
	var e error
	out = make(chan *Msg, 256)
	send = make(chan string, 256)
	conn, e = Connect(server, proxy, ssl)
	if e != nil {
		return nil, e
	}
	go parseLoop()
	go sendLoop()
	return out, nil
}
