package main

import (
	"flag"
	"os"

	"gitee.com/dark.H/ProxyZ/clientcontroll"
	"gitee.com/dark.H/ProxyZ/deploy"
	"gitee.com/dark.H/gs"
)

func main() {
	cl := ""
	dev := ""
	l := 1080
	flag.StringVar(&cl, "s", "https://localhost:55443", "set server addr")
	flag.IntVar(&l, "l", 1080, "set listen port")
	flag.StringVar(&dev, "dev", "", "use ssh to devploy proxy server ; example -dev 'user@host:port/pwd' ")
	flag.Parse()

	if dev != "" {
		user := ""
		host := ""
		pwd := ""
		gs.Str(dev).Split("@").Every(func(no int, i gs.Str) {
			if no == 0 {
				user = i.Str()
			} else {
				i.Split("/").Every(func(no int, i gs.Str) {
					if no == 0 {
						host = i.Str()
					} else {
						pwd = i.Str()
					}
				})
			}
		})
		if user != "" && host != "" && pwd != "" {
			deploy.DepOneHost(user, host, pwd)
		}
		os.Exit(0)
	}
	cli := clientcontroll.NewClientControll(cl, l)
	cli.Socks5Listen()
}
