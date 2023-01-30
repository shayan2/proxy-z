package servercontroll

import (
	"time"

	"gitee.com/dark.H/gs"
)

func TestServer(server string) time.Duration {
	st := time.Now()
	ok := true
	f := ""
	if !gs.Str(server).In(":55443") {
		server += ":55443"
	}
	if gs.Str(server).In("://") {
		f = "https://" + gs.Str(server).Split("://")[1].Str()
	} else {
		f = "https://" + gs.Str(server).Str()
	}

	HTTPSGet(f + "/z-info").Json().Every(func(k string, v any) {
		if k == "status" {
			gs.S(v).Color("g").Println(server)
			if v != "ok" {
				gs.Str("server is not alive !").Color("r").Println()
				ok = false
			}
		}
	})
	if !ok {
		return time.Duration(30000) * time.Hour
	}
	return time.Since(st)
}

func SendUpdate(server string) {
	HTTPSPost("https://"+gs.Str(server).Split("://")[1].Str()+"/z11-update", nil).Json().Every(func(k string, v any) {
		gs.S(v).Color("g").Println(server + " > " + k)
	})
}
