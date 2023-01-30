package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"gitee.com/dark.H/ProxyZ/clientcontroll"
	"gitee.com/dark.H/ProxyZ/deploy"
	"gitee.com/dark.H/ProxyZ/servercontroll"
	"gitee.com/dark.H/gs"
	"gitee.com/dark.H/gt"
)

func main() {
	server := ""
	dev := false
	update := false
	vultrmode := false
	// configbuild := false
	l := 1080
	flag.StringVar(&server, "H", "https://localhost:55443", "set server addr/set ssh name / set some other ")
	flag.IntVar(&l, "l", 1080, "set local socks5 listen port")

	flag.BoolVar(&dev, "dev", false, "use ssh to devploy proxy server ; example -H 'user@host:port/pwd' -dev ")
	flag.BoolVar(&update, "update", false, "set this server update by git")
	flag.BoolVar(&vultrmode, "vultr", false, "true to use vultr api to search host")
	// flag.BoolVar(&configbuild, "", false, "true to use vultr api to build host group")

	flag.Parse()

	if !gs.Str(server).In(":") {
		server += ":55443"
	}
	if !gs.Str(server).In("://") {
		server = "https://" + server
	}
	if dev {
		deploy.DepBySSH(server)
		os.Exit(0)
	}
	if vultrmode {
		for {
			tag := gt.TypedInput("Search Tag[exit] >")
			if tag == "exit" {
				break
			}
			devs := deploy.SearchFromVultr(tag.Str(), server)
			devs.Every(func(no int, i deploy.Onevps) {
				i.Println()
			})
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("enter to continue / build to build dev for all. / sync to sync all route to git ")
			handler, _ := reader.ReadString('\n')
			switch gs.Str(handler).Trim() {
			case "build":
				waiter := sync.WaitGroup{}
				devs.Every(func(no int, i deploy.Onevps) {
					waiter.Add(1)
					go func() {
						defer waiter.Done()
						i.Build()
					}()
				})
			case "sync":
				fmt.Print("git url:")
				repo, _ := reader.ReadString('\n')
				fmt.Print("git name:")
				gitname, _ := reader.ReadString('\n')
				fmt.Print("git pwd:")
				gitpwd, _ := reader.ReadString('\n')
				fmt.Print("set login name:")
				loginname, _ := reader.ReadString('\n')

				fmt.Print("set login pwd:")
				loginpwd, _ := reader.ReadString('\n')
				deploy.SyncToGit(gs.Str(repo).Trim().Str(), gs.Str(gitname).Trim().Str(), gs.Str(gitpwd).Trim().Str(), gs.Str(loginname).Trim().Str(), gs.Str(loginpwd).Trim().Str(), devs)
				fmt.Print("enter to continue")
				reader.ReadString('\n')
			}

		}

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
