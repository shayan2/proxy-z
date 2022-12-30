package main

import (
	"flag"

	"gitee.com/dark.H/ProxyZ/controll"
	"gitee.com/dark.H/gs"
)

var (
	server   = ""
	filePath = ""
	filename = ""
)

func main() {
	flag.StringVar(&server, "u", "https://localhost:55443", "http3 server addr")
	flag.StringVar(&filePath, "f", "", "http3 client upload   file path")
	flag.StringVar(&filename, "d", "", "http3 client download file path")

	flag.Parse()
	args := gs.List[string](flag.Args()).Join(" ")
	if args != "" {
		data := args.ParseKV()
		controll.HTTP3Post(server, toany(data)).Print()
		return
	}
	if filename != "" {
		gs.Str("Downloads").Mkdir()
		controll.HTTP3DownFile(gs.Str(server), gs.Str(filename), gs.Str("Downloads").PathJoin(filename)).Print()
		return
	}
	if filePath != "" {
		controll.HTTP3UploadFile(gs.Str(server), gs.Str(filePath)).Print()
		return
	}
	// gs.Str("->" + server).Println()
	controll.HTTP3Get(server).Print()

}

func toany(r gs.Dict[gs.Str]) (d gs.Dict[any]) {
	d = make(gs.Dict[any])
	r.Every(func(k string, v gs.Str) {
		d[k] = v
	})
	return
}
