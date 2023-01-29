package main

import (
	"flag"
	"os"
	"time"

	"gitee.com/dark.H/ProxyZ/clientcontroll"
	"gitee.com/dark.H/ProxyZ/deploy"
	"gitee.com/dark.H/ProxyZ/servercontroll"
	"gitee.com/dark.H/gs"
)

func main() {
	server := ""
	dev := ""
	update := false
	l := 1080
	flag.StringVar(&server, "s", "https://localhost:55443", "set server addr")
	flag.BoolVar(&update, "u", false, "set this server update by git")
	flag.IntVar(&l, "l", 1080, "set listen port")
	flag.StringVar(&dev, "dev", "", "use ssh to devploy proxy server ; example -dev 'user@host:port/pwd' ")
	flag.Parse()

	if !gs.Str(server).In(":") {
		server += ":55443"
	}
	if !gs.Str(server).In("://") {
		server = "https://" + server
	}
	if dev != "" {
		deploy.DepBySSH(dev)
		os.Exit(0)
	}
	if r := servercontroll.TestServer(server); r > time.Minute {
		os.Exit(0)
		return
	} else {
		gs.Str("server build time: %s ").F(r).Println("test")
	}
	if update {
		servercontroll.SendUpdate(server)
		os.Exit(0)
	}
	cli := clientcontroll.NewClientControll(server, l)
	cli.Socks5Listen()
}
