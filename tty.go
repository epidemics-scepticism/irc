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
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

func PrintHelp() {
	PrintLine("/join <channel> - Join a channel.")
	PrintLine("/msg <rcpt> <msg> - Message a channel or user with msg")
	PrintLine("/part <channel> [reason] - Part from a channel [for reason]")
	PrintLine("/quit [reason] - Quit [for reason]")
	PrintLine("/nick <nick> - Change your nick")
	PrintLine("/ctcp <rcpt> <msg> - CTCP a channel or user with msg")
	PrintLine("/ignore <rcpt> - Ignore messages from rcpt")
	PrintLine("/unignore <rcpt> - Unignore rcpt")
	PrintLine("/otr-start <rcpt> - Request an OTR session with a user")
	PrintLine("/otr-end <rcpt> - End an OTR session with a user")
	PrintLine("/otr-status <rcpt> - Check the status of OTR with a user")
	PrintLine("/otr-info - Print your OTR fingerprint, if loaded")
	PrintLine("/otr-smpr <rcpt> <response> - Response to an SMP question")
	PrintLine("/otr-smpq <rcpt> <question>? <response> - Pose an SMP question (question must end with a ?)")
	PrintLine("/raw <request> - Send a raw input line to the server")
	PrintLine("/help - this screen!")
	PrintLine("by default, message are sent to the previous user or channel")
}

var (
	t         *terminal.Terminal
	promptEnd string = "> "
	curRcpt   string
	tw, th    int
	escapes   = map[string]string{}
	printMap  = map[string]func(m *Msg){
		"NOTICE":  noticeMsg,
		"ERROR":   errorMsg,
		"PRIVMSG": privMsg,
		"PART":    partMsg,
		"JOIN":    joinMsg,
		"QUIT":    quitMsg,
		"NICK":    nickMsg,
		"KICK":    kickMsg,
		"MODE":    modeMsg,
	}
	inputMap = map[string]func(args string){
		"join":       inputJoin,
		"part":       inputPart,
		"quit":       inputQuit,
		"msg":        inputMsg,
		"nick":       inputNick,
		"ctcp":       inputCtcp,
		"ignore":     inputIgnore,
		"unignore":   inputUnignore,
		"otr-start":  inputOtrInit,
		"otr-end":    inputOtrEnd,
		"otr-status": inputOtrStatus,
		"otr-info":   inputOtrInfo,
		"otr-smpr":   inputOtrSmpr,
		"otr-smpq":   inputOtrSmpq,
		"raw":        inputRaw,
		"help":       inputHelp,
		"shrug":      inputShrug,
	}
	IgnoreMap = make(map[string]bool)
)

func inputHelp(args string) {
	PrintHelp()
}

func inputRaw(args string) {
	Raw(args)
}

func inputIgnore(args string) {
	if len(args) == 0 {
		for ignore := range IgnoreMap {
			PrintLine("Ignore: " + ignore)
		}
	} else if _, ok := IgnoreMap[args]; !ok {
		IgnoreMap[args] = true
	}
}

func inputUnignore(args string) {
	if _, ok := IgnoreMap[args]; ok {
		delete(IgnoreMap, args)
	}
}

func inputOtrSmpq(args string) {
	rcpt, msg := split(args, " ")
	quest, resp := split(msg, "? ")
	if len(resp) < 1 {
		PrintLine("Invalid question format, it must end with a ? character")
		return
	}
	quest += "?"
	OtrSmpQuestion(rcpt, quest, resp)
}

func inputOtrSmpr(args string) {
	rcpt, msg := split(args, " ")
	OtrSmpResp(rcpt, msg)
}

func inputOtrInfo(args string) {
	OtrInfo()
}

func inputOtrStatus(args string) {
	OtrStatus(args)
}

func inputOtrInit(args string) {
	curRcpt = args
	OtrStart(args)
	updateTerm()
}

func inputOtrEnd(args string) {
	OtrEnd(args)
}

func inputCtcp(args string) {
	rcpt, msg := split(args, " ")
	Ctcp(rcpt, msg)
}

func inputNick(args string) {
	NewNick(args)
}

func inputJoin(args string) {
	curRcpt = args
	Join(args)
	updateTerm()
}

func inputPart(args string) {
	Part(args)
}

func inputQuit(args string) {
	Quit(args)
}

func inputShrug(args string) {
	SendTo(curRcpt, "¯\\_(ツ)_/¯")
}

func inputMsg(args string) {
	rcpt, msg := split(args, " ")
	curRcpt = rcpt
	SendTo(rcpt, msg)
	updateTerm()
}

func setEscapeCodes() {
	escapes["Black"] = string(t.Escape.Black)
	escapes["Red"] = string(t.Escape.Red)
	escapes["Green"] = string(t.Escape.Green)
	escapes["Yellow"] = string(t.Escape.Yellow)
	escapes["Blue"] = string(t.Escape.Blue)
	escapes["Magenta"] = string(t.Escape.Magenta)
	escapes["Cyan"] = string(t.Escape.Cyan)
	escapes["White"] = string(t.Escape.White)
	escapes["Reset"] = string(t.Escape.Reset)
}

func ansiColour(c, s string) string {
	return escapes[c] + s + escapes["Reset"]
}

func modeMsg(m *Msg) {
	s := "[" + m.timestamp + "]"
	s += " " + m.nick + " set mode "
	s += "[" + m.args + m.content + "] "
	s += "for " + m.rcpt
	PrintLine(s)
}

func kickMsg(m *Msg) {
	s := "[" + m.timestamp + "]"
	s += " " + m.nick + " kicked " + m.args + " from " + m.rcpt + " [" + m.content + "]"
	PrintLine(s)
}

func privMsg(m *Msg) {
	var colour string

	if len(m.content) < 1 {
		return
	}
	s := "[" + m.timestamp + "]"
	if m.rcpt[0] == '#' {
		colour = "Yellow"
	} else if m.enc {
		colour = "Green"
	} else {
		colour = "Red"
	}
	s += " [" + ansiColour(colour, m.nick) + "@" + ansiColour(colour, m.rcpt) + "]"
	if strings.HasPrefix(m.content, "\x01") && strings.HasSuffix(m.content, "\x01") {
		m.content = strings.Trim(m.content, "\x01")
		ctcp, args := split(m.content, " ")
		switch {
		case ctcp == "ACTION":
			s += " *" + args + "*"
		default:
			s += " " + m.content
		}
	} else {
		s += " " + m.content
	}
	PrintLine(s)
}

func nickMsg(m *Msg) {
	s := "[" + m.timestamp + "]"
	s += " " + m.nick + " is now known as " + m.content
	if _, ok := IgnoreMap[m.content]; ok {
		delete(IgnoreMap, m.content)
	}
	if _, ok := IgnoreMap[m.nick]; ok {
		IgnoreMap[m.content] = true
		delete(IgnoreMap, m.nick)
	}
	PrintLine(s)
}

func noticeMsg(m *Msg) {
	s := "[" + m.timestamp + "]"
	s += " [" + ansiColour("Yellow", m.nick) + "@" + ansiColour("Yellow", m.rcpt) + "]"
	s += " " + ansiColour("Yellow", m.cmd) + ":"
	if len(m.args) > 0 {
		s += " [" + ansiColour("Yellow", m.args) + "]"
	}
	s += " " + ansiColour("Yellow", m.content)
	PrintLine(s)
}

func errorMsg(m *Msg) {
	s := "[" + m.timestamp + "]"
	s += " [" + ansiColour("Magenta", m.nick) + "@" + ansiColour("Magenta", m.rcpt) + "]"
	s += " " + ansiColour("Magenta", m.cmd) + ":"
	if len(m.args) > 0 {
		s += " [" + ansiColour("Magenta", m.args) + "]"
	}
	s += " " + ansiColour("Magenta", m.content)
	PrintLine(s)
}

func partMsg(m *Msg) {
	if !*ircClean {
		s := "[" + m.timestamp + "]"
		s += " " + m.nick + " [" + m.user + "@" + m.host + "]" + " has left " + m.rcpt + " [" + m.content + "]"
		PrintLine(s)
	}
}

func joinMsg(m *Msg) {
	if !*ircClean {
		s := "[" + m.timestamp + "]"
		s += " " + m.nick + " [" + m.user + "@" + m.host + "]" + " has joined "
		if len(m.rcpt) > 0 {
			s += m.rcpt
		} else {
			s += m.content
		}
		PrintLine(s)
	}
}

func quitMsg(m *Msg) {
	if !*ircClean {
		s := "[" + m.timestamp + "]"
		s += " " + m.nick + " [" + m.user + "@" + m.host + "]" + " has quit [" + m.content + "]"
		PrintLine(s)
	}
}

func updateTerm() {
	var colour string
	if strings.HasPrefix(curRcpt, "#") {
		colour = "Yellow"
	} else if OtrIsEncrypted(curRcpt) {
		colour = "Green"
	} else {
		colour = "Red"
	}
	t.SetPrompt(ansiColour(colour, curRcpt) + promptEnd)
	cw, ch, e := terminal.GetSize(0)
	if e != nil {
		PrintError(e)
		return
	}
	tw = cw
	th = ch
	t.SetSize(tw, th)
}

func PrintLine(line string) {
	if t != nil {
		t.Write([]byte(line + "\r\n"))
		updateTerm()
	} else {
		fmt.Fprintln(os.Stdout, line)
	}
}

func PrintError(e error) {
	line := ansiColour("Magenta", e.Error())
	PrintLine(line)
}

func InitTty() {
	state, e := terminal.MakeRaw(0)
	if e != nil {
		PrintError(e)
		return
	}
	defer terminal.Restore(0, state)
	t = terminal.NewTerminal(os.Stdin, promptEnd)
	setEscapeCodes()
	go func() {
		for {
			m := <-out
			if f, ok := printMap[m.cmd]; ok {
				f(m)
			} else if strings.HasPrefix(m.cmd, "4") || strings.HasPrefix(m.cmd, "5") {
				errorMsg(m)
			} else if strings.HasPrefix(m.cmd, "2") || strings.HasPrefix(m.cmd, "3") {
				noticeMsg(m)
			}
		}
	}()
	for {
		s, e := t.ReadLine()
		if e != nil {
			PrintError(e)
			Quit("Leaving.")
			return
		}
		if strings.HasPrefix(s, "/") {
			s = s[1:]
			cmd, args := split(s, " ")
			if f, ok := inputMap[cmd]; ok {
				f(args)
			} else {
				PrintLine("Unknown command '" + cmd + "'. Try /help.")
			}
		} else {
			SendTo(curRcpt, s)
		}
	}
}
