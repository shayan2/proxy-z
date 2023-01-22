package main

import (
	"bytes"
	"log"
	"net"
	"os"
	"time"

	"gitee.com/dark.H/ProxyZ/client"
	"gitee.com/dark.H/ProxyZ/connections/prosocks5"
	"gitee.com/dark.H/gs"
)

func main() {
	t := os.Args[1]
	l, err := net.Listen("tcp", "0.0.0.0:"+gs.S(3080).Str())
	if err != nil {
		log.Fatal(err)
	}
	for {
		socks5con, err := l.Accept()
		if err != nil {
			gs.S(err.Error()).Println("accept err")
			time.Sleep(3 * time.Second)
			continue
		}

		go func(socks5con net.Conn) {
			defer socks5con.Close()
			err := prosocks5.Socks5HandShake(&socks5con)
			if err != nil {
				gs.Str(err.Error()).Println("socks5 handshake")
				return
			}

			raw, host, _, err := prosocks5.GetLocalRequest(&socks5con)
			if err != nil {
				gs.Str(err.Error()).Println("socks5 get host")
				return
			}
			remotecon, err := connect(t)
			if err != nil {
				gs.Str(err.Error()).Println("connect proxy server err")
				return
			}
			defer remotecon.Close()
			_, err = remotecon.Write(raw)
			if err != nil {
				gs.Str(err.Error()).Println("connecting write|" + host)

				return
			}
			// gs.Str(host).Color("g").Println("connect|write")
			_buf := make([]byte, len(prosocks5.Socks5Confirm))
			remotecon.SetReadDeadline(time.Now().Add(10 * time.Second))
			_, err = remotecon.Read(_buf)

			if err != nil {
				gs.Str(err.Error()).Println("connecting read|" + host)
				if err.Error() != "timeout" {
					panic(err)
				}

				return
			}
			if bytes.Equal(_buf, prosocks5.Socks5Confirm) {
				_, err = socks5con.Write(_buf)
				if err != nil {
					gs.Str(err.Error()).Println("connecting reply|" + host)
					return
				}
			}

			gs.Str(host).Color("g").Println("connecting|")
			client.Pipe(socks5con, remotecon)

		}(socks5con)

	}

}

func connect(t string) (c net.Conn, err error) {
	return
}
