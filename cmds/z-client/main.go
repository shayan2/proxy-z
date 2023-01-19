package main

import (
	"flag"

	"gitee.com/dark.H/ProxyZ/client"
)

func main() {
	cl := ""
	l := 1080
	flag.StringVar(&cl, "s", "https://localhost:55443", "set server addr")
	flag.IntVar(&l, "l", 1080, "set listen port")
	flag.Parse()

	cli := client.NewClientControll(cl, l)
	cli.Socks5Listen()
}
