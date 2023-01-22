package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"gitee.com/dark.H/ProxyZ/connections/prokcp"
	"gitee.com/dark.H/ProxyZ/connections/prosmux"
	"gitee.com/dark.H/ProxyZ/connections/prosocks5"
	"gitee.com/dark.H/ProxyZ/connections/protls"
	"gitee.com/dark.H/ProxyZ/controll"
	"gitee.com/dark.H/gs"
)

type ClientControl struct {
	SmuxClient *prosmux.SmuxConfig
	nowconf    *baseconnection.ProtocolConfig
	ListenPort int
	ErrCount   int
	AliveCount int

	lock sync.RWMutex
	Addr gs.Str
}

func NewClientControll(addr string, listenport int) *ClientControl {
	c := &ClientControl{
		Addr:       gs.Str(addr),
		ListenPort: listenport,
	}
	return c
}

func RecvMsg(reply gs.Str) (di any, o bool) {
	d := reply.Json()
	if c, ok := d["status"]; ok {
		if c.(string) == "ok" {
			o = true
		}

		di = d["msg"]
		return
	} else {
		o = false
		return
	}
}

func (c *ClientControl) GetAviableProxy() (conf *baseconnection.ProtocolConfig) {
	if c.nowconf != nil {
		return c.nowconf
	}
	var addr string
	useTls := false
	if c.Addr.StartsWith("tls://") {
		addr = c.Addr.Split("://")[1].Str()
		useTls = true
	} else if c.Addr.In("://") {
		addr = c.Addr.Split("://")[1].Str()
	} else {
		addr = c.Addr.Str()
	}
	var reply gs.Str
	if useTls {
		reply = controll.HTTPSPost("https://"+addr+"/proxy-get", nil)
	} else {
		reply = controll.HTTP3Post("https://"+addr+"/proxy-get", nil)
	}

	if reply == "" {
		return nil
	}
	if obj, ok := RecvMsg(reply); ok {
		// fmt.Println(obj)
		buf, err := json.Marshal(obj)
		if err != nil {
			gs.Str(err.Error()).Println("Err Tr")
			return nil
		}
		conf = new(baseconnection.ProtocolConfig)

		if err := json.Unmarshal(buf, conf); err != nil {
			gs.Str("get aviable proxy client err :" + err.Error()).Println("Err")
			return nil
		}
		if conf.Server == "0.0.0.0" {
			conf.Server = gs.Str(addr).Split(":")[0].Trim()
		}
		c.nowconf = conf
	}

	return
}

/*
**************************************************************
**************************************************************
CORE ！！！！！！！！
*/
func (c *ClientControl) Socks5Listen() {
	if c.ListenPort != 0 {
		l, err := net.Listen("tcp", "0.0.0.0:"+gs.S(c.ListenPort).Str())
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
				remotecon, err := c.ConnectRemote()
				if err != nil {
					gs.Str(err.Error()).Println("connect proxy server err")
					return
				}
				defer remotecon.Close()
				_, err = remotecon.Write(raw)
				if err != nil {
					gs.Str(err.Error()).Println("connecting write|" + host)
					c.lock.Lock()
					c.ErrCount += 1
					c.lock.Unlock()
					return
				}
				// gs.Str(host).Color("g").Println("connect|write")
				_buf := make([]byte, len(prosocks5.Socks5Confirm))
				remotecon.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, err = remotecon.Read(_buf)

				if err != nil {
					gs.Str(err.Error()).Println("connecting read|" + host)
					if err.Error() != "timeout" {
						panic(err)
					}

					c.lock.Lock()
					c.ErrCount += 1
					c.lock.Unlock()
					return
				}
				if bytes.Equal(_buf, prosocks5.Socks5Confirm) {
					_, err = socks5con.Write(_buf)
					if err != nil {
						gs.Str(err.Error()).Println("connecting reply|" + host)
						return
					}
				}

				c.lock.Lock()
				c.AliveCount += 1
				if c.ErrCount > 0 {
					c.ErrCount -= 1
				}
				c.lock.Unlock()
				gs.Str(host).Color("g").Println("connecting|" + gs.S(c.AliveCount).Str())
				c.Pipe(socks5con, remotecon)

				c.lock.Lock()
				c.AliveCount -= 1
				c.lock.Unlock()

			}(socks5con)

		}
	}
}

func (c *ClientControl) RebuildSmux() (err error) {
	proxyConfig := c.GetAviableProxy()

	var singleTunnelConn net.Conn
	switch proxyConfig.ProxyType {
	case "tls":

		singleTunnelConn, err = protls.ConnectTls(proxyConfig.RemoteAddr(), proxyConfig)
	case "kcp":
		singleTunnelConn, err = prokcp.ConnectKcp(proxyConfig.RemoteAddr(), proxyConfig)
	default:
		singleTunnelConn, err = prokcp.ConnectKcp(proxyConfig.RemoteAddr(), proxyConfig)
	}
	gs.Str("--> "+proxyConfig.RemoteAddr()).Color("y", "B").Println(proxyConfig.ProxyType)
	if singleTunnelConn != nil {
		c.SmuxClient = prosmux.NewSmuxClient(singleTunnelConn)
	} else {
		if err == nil {
			err = errors.New("tls/kcp only :  now method is :" + proxyConfig.ProxyType)
		}
		return err
	}
	return nil
}

func (c *ClientControl) ConnectRemote() (con net.Conn, err error) {
	if c.SmuxClient == nil {
		err = c.RebuildSmux()
		if err != nil {
			gs.Str("rebuild smux").Println("connect remote")
			return nil, err
		}
	}
	// connted := false
	con, err = c.SmuxClient.NewConnnect()
	if err != nil {
		gs.Str("rebuild smux").Println("connect remote")
		err = c.RebuildSmux()
		if err != nil {
			return nil, err
		}
		con, err = c.SmuxClient.NewConnnect()
	}
	// gs.Str("smxu connect ").Println()
	return
}

func (c *ClientControl) Pipe(p1, p2 net.Conn) {
	var wg sync.WaitGroup
	var wait = 15 * time.Second
	wg.Add(1)
	streamCopy := func(dst net.Conn, src net.Conn, fr, to net.Addr) {
		// startAt := time.Now()
		Copy(dst, src)
		dst.SetReadDeadline(time.Now().Add(wait))
		p1.Close()
		p2.Close()
		// }()
	}

	go func(p1, p2 net.Conn) {
		wg.Done()
		streamCopy(p1, p2, p2.RemoteAddr(), p1.RemoteAddr())
	}(p1, p2)
	streamCopy(p2, p1, p1.RemoteAddr(), p2.RemoteAddr())
	wg.Wait()
}

// Memory optimized io.Copy function specified for this library
func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}

	// fallback to standard io.CopyBuffer
	buf := make([]byte, 4096)
	return io.CopyBuffer(dst, src, buf)
}

func Pipe(p1, p2 net.Conn) {
	var wg sync.WaitGroup
	var wait = 15 * time.Second
	wg.Add(1)
	streamCopy := func(dst net.Conn, src net.Conn, fr, to net.Addr) {
		// startAt := time.Now()
		Copy(dst, src)
		dst.SetReadDeadline(time.Now().Add(wait))
		p1.Close()
		p2.Close()
		// }()
	}

	go func(p1, p2 net.Conn) {
		wg.Done()
		streamCopy(p1, p2, p2.RemoteAddr(), p1.RemoteAddr())
	}(p1, p2)
	streamCopy(p2, p1, p1.RemoteAddr(), p2.RemoteAddr())
	wg.Wait()
}
