package main

import (
	"flag"

	"gitee.com/dark.H/ProxyZ/servercontroll"
	"gitee.com/dark.H/gs"
)

var (
	quicserver = ""
	tlsserver  = ""
	filePath   = ""
	filename   = ""
)

func main() {
	flag.StringVar(&quicserver, "uq", "https://localhost:55444", "http3 server addr")
	flag.StringVar(&tlsserver, "u", "https://localhost:55443", "https server addr")
	flag.StringVar(&filePath, "f", "", "http3 client upload   file path")
	flag.StringVar(&filename, "d", "", "http3 client download file path")

	flag.Parse()
	args := gs.List[string](flag.Args()).Join(" ")
	if args != "" {
		data := args.ParseKV()
		servercontroll.HTTP3Post(quicserver, toany(data)).Print()
		return
	}
	if filename != "" {
		gs.Str("Downloads").Mkdir()
		servercontroll.HTTP3DownFile(gs.Str(quicserver), gs.Str(filename), gs.Str("Downloads").PathJoin(filename)).Print()
		return
	}
	if filePath != "" {
		servercontroll.HTTP3UploadFile(gs.Str(quicserver), gs.Str(filePath)).Print()
		return
	}
	// gs.Str("->" + server).Println()
	servercontroll.HTTPSGet(tlsserver).Print()

}

func toany(r gs.Dict[gs.Str]) (d gs.Dict[any]) {
	d = make(gs.Dict[any])
	r.Every(func(k string, v gs.Str) {
		d[k] = v
	})
	return
}
