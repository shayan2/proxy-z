package main

import (
	"bytes"
	"crypto/sha1"
	"flag"
	"log"
	"net"
	"time"

	"gitee.com/dark.H/ProxyZ/client"
	"gitee.com/dark.H/ProxyZ/connections/prosocks5"
	"gitee.com/dark.H/gs"
	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

var (
	_key  = "hello world!"
	_salt = "hello Golang World !"
)

func main() {
	t := ""
	S := false
	flag.StringVar(&t, "t", "", "target server")
	flag.BoolVar(&S, "S", false, "server mode")
	flag.Parse()

	if !S {
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

	} else {
		SServer(44806)
	}

}

func connect(t string) (c net.Conn, err error) {
	key := pbkdf2.Key([]byte(_key), []byte(_salt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	DataShard := 10
	ParityShard := 3
	// gs.Str("key:%s | salt: %s | ds:%d | pd: %d | mode:%s ").F(_key, _salt, DataShard, ParityShard, config.Type).Println("kcp config")
	kcpconn, err := kcp.DialWithOptions(t, block, DataShard, ParityShard)
	return kcpconn, err
}

func SServer(p int) {

	key := pbkdf2.Key([]byte(_key), []byte(_salt), 4096, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	// var listener net.Listener
	serverAddr := gs.Str(":%d").F(p)

	DataShard := 10
	ParityShard := 3
	addr := serverAddr.Str()
	gs.Str(addr).Println("listen kcp")
	if listener, err := kcp.ListenWithOptions(addr, block, DataShard, ParityShard); err == nil {
		for {
			con, err := listener.AcceptKCP()
			if err != nil {
				panic(err)
			}
			gs.Str("accept ").Println()
			host, _, _, err := prosocks5.GetServerRequest(con)
			if err != nil {
				panic(err)
			}
			gs.Str(host).Println("server host")
			remoteConn, err := net.Dial("tcp", host)
			if err != nil {
				log.Fatal("dial remote host err:", host)
			}
			gs.Str(host).Println("host|ok")
			// con.SetWriteDeadline(time.Now().Add(2 * time.Minute))
			_, err = con.Write(prosocks5.Socks5Confirm)
			if err != nil {
				// ErrToFile("back con is break", err)
				remoteConn.Close()
				panic(err)

			}
			gs.Str(host).Println("host|build")
			client.Pipe(remoteConn, con)

		}

	} else {

	}
}
