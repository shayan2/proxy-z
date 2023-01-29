package main

import (
	"flag"

	"gitee.com/dark.H/ProxyZ/servercontroll"
	"gitee.com/dark.H/gs"
)

var (
	tlsserver  = ""
	quicserver = ""
	www        = ""
)

func main() {
	flag.StringVar(&quicserver, "quic-api", "0.0.0.0:55444", "http3 server addr")
	flag.StringVar(&tlsserver, "tls-api", "0.0.0.0:55443", "http3 server addr")
	flag.StringVar(&www, "www", "/tmp/www", "http3 server www dir path")
	flag.Parse()
	if !gs.Str(www).IsExists() {
		gs.Str(www).Mkdir()
	}
	// gs.Str(quicserver).Println("Server Run")
	go servercontroll.HTTP3Server(quicserver, www, true)
	servercontroll.HTTP3Server(tlsserver, www, false)

}
