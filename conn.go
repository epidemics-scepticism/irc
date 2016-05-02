package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/proxy"
)

var saneCipherSuites = []uint16{ // Fuck RC4 and DES && prefer ephemeral KEX.
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384, // no pfs!
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256, // no pfs!
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,    // no pfs!
}

func socksConn(host, socksproxy string) (net.Conn, error) {
	s := fmt.Sprintf("%d", time.Now().UnixNano()) // hacky isolation "good enough"
	d, e := proxy.SOCKS5("tcp", socksproxy, &proxy.Auth{s, s}, new(net.Dialer))
	if e != nil {
		return nil, e
	}
	conn, e := d.Dial("tcp", host)
	if e != nil {
		return nil, e
	}
	return conn, nil
}

func tlsConn(hostname string, conn net.Conn) *tls.Conn {
	cfg := new(tls.Config)
	cfg.ServerName = hostname
	cfg.CipherSuites = saneCipherSuites
	tconn := tls.Client(conn, cfg)
	if e := tconn.Handshake(); e != nil {
		return tconn
	}
	state := tconn.ConnectionState()
	PrintLine("TLS: Cipher '" + ansiColour("Green", tlsCipherSuite(state.CipherSuite)) + "'")
	for k, v := range state.PeerCertificates {
		certLine := fmt.Sprintf("TLS: Cert Chain [%d]\tSubject: %s\tIssuer: %s\tFingerprint: %s",
		k,
		ansiColour("White", v.Subject.CommonName),
		ansiColour("White", v.Issuer.CommonName),
		ansiColour("White", fingerprint(v.Raw, false)))
		PrintLine(certLine)
	}
	return tconn
}

func tlsCipherSuite(c uint16) string {
	cipherSuites := map[uint16]string {
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384: "ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384: "ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA: "ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA: "ECDHE_RSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256: "ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256: "ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA: "ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA: "ECDHE_RSA_WITH_AES_128_CBC_SHA",
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384: "RSA_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256: "RSA_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_RSA_WITH_AES_256_CBC_SHA: "RSA_RSA_WITH_AES_256_CBC_SHA",
	}
	if s, ok := cipherSuites[c]; ok {
		return s
	} else {
		return "UNKNOWN"
	}
}

func Connect(host, proxy string, ssl bool) (net.Conn, error) {
	c, e := socksConn(host, proxy)
	if e != nil {
		return nil, e
	}
	if ssl {
		name, _ := split(host, ":")
		c = tlsConn(name, c)
	}
	return c, nil
}
