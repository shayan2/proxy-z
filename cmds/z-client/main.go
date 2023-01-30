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
	dev := false
	update := false
	vultrmode := false
	gitmode := false
	// configbuild := false
	l := 1080
	flag.StringVar(&server, "H", "https://localhost:55443", "set server addr/set ssh name / set some other ")
	flag.IntVar(&l, "l", 3080, "set local socks5 listen port")

	flag.BoolVar(&dev, "dev", false, "use ssh to devploy proxy server ; example -H 'user@host:port/pwd' -dev ")
	flag.BoolVar(&update, "update", false, "set this server update by git")
	flag.BoolVar(&vultrmode, "vultr", false, "true to use vultr api to search host")
	flag.BoolVar(&gitmode, "git", false, "true to use git to login group proxy")
	// flag.BoolVar(&configbuild, "", false, "true to use vultr api to build host group")

	flag.Parse()

	if dev {
		deploy.DepBySSH(server)
		os.Exit(0)
	}
	if vultrmode {
		deploy.VultrMode(server)
		os.Exit(0)
	}
	if gitmode {
		server = deploy.GitMode(server)
		if server == "" {
			os.Exit(0)
		}

	}

	if !gs.Str(server).In(":") {
		server += ":55443"
	}
	if !gs.Str(server).In("://") {
		server = "https://" + server
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
