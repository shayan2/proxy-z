package main

import (
	"flag"
	"os"

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

	if dev != "" {
		deploy.DepBySSH(dev)
		os.Exit(0)
	}

	servercontroll.HTTPSGet("https://" + gs.Str(server).Split("://")[1].Str() + "/z-info").Json().Every(func(k string, v any) {
		if k == "status" {
			gs.S(v).Color("g").Println(server)
			if v != "ok" {
				gs.Str("server is not alive !").Color("r").Println()
				os.Exit(0)
			}
		}
	})
	if update {
		servercontroll.HTTPSPost("https://"+gs.Str(server).Split("://")[1].Str()+"/z11-update", nil).Json().Every(func(k string, v any) {
			gs.S(v).Color("g").Println(server + " > " + k)
		})
		os.Exit(0)
	}
	cli := clientcontroll.NewClientControll(server, l)
	cli.Socks5Listen()
}
