package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/otr"
)

type OtrConf struct {
	key     *otr.PrivateKey              `json:"-"`
	Contact map[string][]byte            `json:",omitempty"`
	conv    map[string]*otr.Conversation `json:"-"`
}

var (
	OTR       *OtrConf
	otrFile   = os.Getenv("HOME") + "/.otr-fingerprints"
	otrLoaded = 0
)

func OtrLoad() {
	defer OtrInfo()
	OTR = new(OtrConf)
	OTR.key = new(otr.PrivateKey)
	OTR.key.Generate(rand.Reader)
	OTR.Contact = make(map[string][]byte)
	OTR.conv = make(map[string]*otr.Conversation)
	offset := strings.Index(*IrcServer, ":")
	if offset > 0 {
		suffix := *IrcServer
		suffix = "-" + suffix[:offset]
		otrFile += suffix
	}
	conf, e := ioutil.ReadFile(otrFile)
	if e != nil {
		PrintError(e)
		return
	}
	e = json.Unmarshal(conf, OTR)
	if e != nil {
		PrintError(e)
		return
	}
}

func OtrSave() {
	conf, e := json.Marshal(OTR)
	if e != nil {
		PrintError(e)
		return
	}
	e = ioutil.WriteFile(otrFile, conf, 0600)
	if e != nil {
		PrintError(e)
	}
}

func OtrNew() *otr.Conversation {
	conv := new(otr.Conversation)
	conv.PrivateKey = OTR.key
	conv.FragmentSize = 400
	return conv
}

func OtrStart(rcpt string) {
	rcpt = strings.ToLower(rcpt)
	if _, ok := OTR.conv[rcpt]; ok == false {
		OTR.conv[rcpt] = OtrNew()
	}
	send <- "PRIVMSG " + rcpt + " :" + otr.QueryMessage
}

func OtrEnd(rcpt string) {
	rcpt = strings.ToLower(rcpt)
	if _, ok := OTR.conv[rcpt]; ok {
		msgs := OTR.conv[rcpt].End()
		for _, msg := range msgs {
			send <- "PRIVMSG " + rcpt + " :" + string(msg)
		}
		delete(OTR.conv, rcpt)
	}
}

func OtrStatus(rcpt string) {
	if OtrIsEncrypted(rcpt) {
		PrintLine("OTR: " + ansiColour("Green", "Encrypted with "+rcpt))
	} else {
		PrintLine("OTR: " + ansiColour("Red", "Unencrypted with "+rcpt))
	}
}

func fingerprint(d []byte, prehashed bool) string {
	if !prehashed {
		h := sha256.Sum256(d)
		d = h[:]
	}
	var frag []string
	for _, v := range d {
		f := hex.EncodeToString([]byte{v})
		frag = append(frag, f)
	}
	hexfp := strings.Join(frag, ":")
	leekfp := leek.Encode(d[:10])
	fp := "[" + hexfp + "](" + leekfp + ")"
	return fp
}

func OtrFingerprint(rcpt string) {
	rcpt = strings.ToLower(rcpt)
	current := OTR.conv[rcpt].TheirPublicKey.Fingerprint()
	fpstring := fingerprint(current, true)
	if stored, ok := OTR.Contact[rcpt]; ok {
		if bytes.Equal(stored, current) {
			PrintLine("OTR: Contact " + rcpt + " has good fingerprint: " + ansiColour("Green", fpstring))
		} else {
			PrintLine("OTR: Contact " + rcpt + " has bad fingerprint: " + ansiColour("Red", fpstring))

		}
	} else {
		// hey I just met you
		// and this is crazy
		// but i trust-on-first-use
		// so call me 9d4737bf104973dfc3ad21019e243406c6a55c33
		OTR.Contact[rcpt] = current
		PrintLine("OTR: Contact " + rcpt + " has unknown fingerprint: " + ansiColour("Yellow", fpstring))
	}
}

func OtrIsEncrypted(nick string) bool {
	nick = strings.ToLower(nick)
	if c, ok := OTR.conv[nick]; ok {
		return c.IsEncrypted()
	} else {
		return false
	}
}

func OtrInfo() {
	if OTR.key != nil {
		fpstring := fingerprint(OTR.key.PublicKey.Fingerprint(), true)
		PrintLine("OTR: " + ansiColour("Green", "Loaded with fingerprint: "+fpstring))
	} else {
		PrintLine("OTR: " + ansiColour("Red", "Not loaded"))
		return
	}
	for r := range OTR.conv {
		if OtrIsEncrypted(r) {
			OtrFingerprint(r)
		} else {
			PrintLine("OTR: Contact " + r + " is currently " + ansiColour("Red", "unencrypted"))
		}
	}
}

func OtrSmpQuestion(rcpt, quest, resp string) {
	rcpt = strings.ToLower(rcpt)
	if _, ok := OTR.conv[rcpt]; ok == false {
		OTR.conv[rcpt] = OtrNew()
	}
	msgs, e := OTR.conv[rcpt].Authenticate(quest, []byte(resp))
	if e != nil {
		PrintError(e)
		return
	}
	for _, msg := range msgs {
		send <- "PRIVMSG " + rcpt + " :" + string(msg)
	}
}

func OtrSmpResp(rcpt, resp string) {
	rcpt = strings.ToLower(rcpt)
	if _, ok := OTR.conv[rcpt]; ok == false {
		OTR.conv[rcpt] = OtrNew()
	}
	smpq := OTR.conv[rcpt].SMPQuestion()
	msgs, e := OTR.conv[rcpt].Authenticate(smpq, []byte(resp))
	if e != nil {
		PrintError(e)
		return
	}
	for _, msg := range msgs {
		send <- "PRIVMSG " + rcpt + " :" + string(msg)
	}
}

func OtrRecv(m *Msg) {
	otrnick := strings.ToLower(m.nick)
	if _, ok := OTR.conv[otrnick]; ok == false {
		OTR.conv[otrnick] = OtrNew()
	}
	recv, enc, chg, msgs, e := OTR.conv[otrnick].Receive([]byte(m.content))
	if e != nil {
		PrintError(e)
		return
	}
	switch {
	case chg == otr.NewKeys:
		OtrFingerprint(otrnick)
	case chg == otr.SMPSecretNeeded:
		smpq := OTR.conv[otrnick].SMPQuestion()
		PrintLine("OTR: " + m.nick + " asks '" + smpq + "'. Type '/otr-smpr " + m.nick + " <response>' to answer.")
	case chg == otr.SMPComplete:
		PrintLine("OTR: " + m.nick + " " + ansiColour("Green", "completed") + " authentication.")
		OTR.Contact[otrnick] = OTR.conv[otrnick].TheirPublicKey.Fingerprint()
	case chg == otr.SMPFailed:
		PrintLine("OTR: " + m.nick + " " + ansiColour("Red", "failed") + " authentication.")
	case chg == otr.ConversationEnded:
		PrintLine("OTR: Ended with " + ansiColour("Red", m.nick))
	}
	m.enc = enc
	m.content = string(recv)
	for _, msg := range msgs {
		send <- "PRIVMSG " + m.nick + " :" + string(msg)
	}
	updateTerm()
}

func OtrSend(rcpt, msg string) {
	rcpt = strings.ToLower(rcpt)
	if _, ok := OTR.conv[rcpt]; ok == false {
		OTR.conv[rcpt] = OtrNew()
	}
	outs, e := OTR.conv[rcpt].Send([]byte(msg))
	if e != nil {
		PrintError(e)
		return
	}
	for _, out := range outs {
		send <- "PRIVMSG " + rcpt + " :" + string(out)
	}
	updateTerm()
}
