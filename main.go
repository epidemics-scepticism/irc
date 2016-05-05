package main

import (
	"flag"
)

var (
	IrcServer = flag.String("server", "irc.oftc.net:6697", "IRC Server as host:port")
	IrcNick   = flag.String("nick", generateNick(), "Nick to use on IRC")
	ircProxy  = flag.String("proxy", "127.0.0.1:9050", "SOCKS5 proxy as host:port")
	ircTls    = flag.Bool("tls", true, "use TLS")
	ircClean  = flag.Bool("clean", true, "Strip join/part/quit/notice")
)

func main() {
	flag.Parse()
	_, e := Init(*IrcServer, *ircProxy, *ircTls)
	if e != nil {
		PrintError(e)
		return
	}
	OtrLoad()
	defer OtrSave()
	Register(*IrcNick)
	InitTty()
}
