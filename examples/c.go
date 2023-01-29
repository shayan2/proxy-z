package main

import (
	"bytes"
	"crypto/sha1"
	"flag"
	"log"
	"net"
	"time"

	"gitee.com/dark.H/ProxyZ/clientcontroll"
	"gitee.com/dark.H/ProxyZ/connections/prosocks5"
	"gitee.com/dark.H/gs"
	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

var (
	_key  = "hello world!"
	_salt = "hello Golang World !"
	KEY   = "demo passdemo pass!"
	SALT  = "demo saltfasdfss"
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
				} else {
					gs.Str(t).Println("++")
				}
				defer remotecon.Close()
				// time.Sleep(1 * time.Second)
				gs.S(raw).Println("Write")
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
				clientcontroll.Pipe(socks5con, remotecon)

			}(socks5con)

		}

	} else {
		SServer(t)
	}

}

func connect(t string) (c net.Conn, err error) {
	key := pbkdf2.Key([]byte(_key), []byte(_salt), 4096, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	DataShard := 10
	ParityShard := 3
	// gs.Str("key:%s | salt: %s | ds:%d | pd: %d | mode:%s ").F(_key, _salt, DataShard, ParityShard, config.Type).Println("kcp config")
	kcpconn, err := kcp.DialWithOptions(t, block, DataShard, ParityShard)
	return kcpconn, err
}

func SServer(addr string) {
	key := pbkdf2.Key([]byte(_key), []byte(_salt), 4096, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	if listener, err := kcp.ListenWithOptions(addr, block, 10, 3); err == nil {
		for {
			con, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			go func(c net.Conn) {
				gs.Str("accept ").Println()
				host, _, _, err := prosocks5.GetServerRequest(c)
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
				_, err = c.Write(prosocks5.Socks5Confirm)
				if err != nil {
					// ErrToFile("back con is break", err)
					remoteConn.Close()
					panic(err)

				}
				defer c.Close()
				gs.Str(host).Println("host|build")
				clientcontroll.Pipe(remoteConn, c)

			}(con)

		}

	} else {
		log.Fatal(err)
	}
}
