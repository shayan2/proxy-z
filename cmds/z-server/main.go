package main

import (
	"flag"

	"gitee.com/dark.H/ProxyZ/controll"
	"gitee.com/dark.H/gs"
)

var (
	server = ""
	www    = ""
)

func main() {
	flag.StringVar(&server, "server", "0.0.0.0:55443", "http3 server addr")
	flag.StringVar(&www, "www", "/tmp/www", "http3 server www dir path")
	flag.Parse()
	if !gs.Str(www).IsExists() {
		gs.Str(www).Mkdir()
	}
	gs.Str(server).Println("Server Run")
	controll.HTTP3Server(server, www)

}
