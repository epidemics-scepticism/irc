# irc
privacy-aware tty-based irc client written in go

## features
* uses Christopher Pounds pseudolanguage generator to generate nicks
* otr with green(on)/red(off) and yellow(not applicable) indicators and smp support
* tls with some sane ciphersuites
* uses leekspeak to help provide a second vantage point to verify fingerprints

## notes
* we throw away our otr key after we're done. keeping long term identity isn't wanted behaviour.
* pretty much requires use of a SOCKS5 proxy, defaults to Tor but works with openssh too.
* currently won't gracefully handle bad tls certs (who has an excuse with letsencrypt?)
* ctrl-d (EOF) quits
