package baseconnection

import (
	"errors"
	"net"
	"sync"
	"syscall"
	"time"

	"gitee.com/dark.H/ProxyZ/connections/prosmux"
	"gitee.com/dark.H/ProxyZ/connections/prosocks5"
	"gitee.com/dark.H/gs"
)

type Protocol interface {
	GetListener() net.Listener
	GetConfig() *ProtocolConfig
	DelCon(con net.Conn)
}

type ProxyTunnel struct {
	cons         gs.List[net.Conn]
	alive        int
	lock         sync.RWMutex
	protocl      Protocol
	UseSmux      bool
	On           bool
	ControllFunc func(rawHost string, con net.Conn) (err error)
}

func NewProxyTunnel(procol Protocol) *ProxyTunnel {
	p := new(ProxyTunnel)
	p.protocl = procol
	p.UseSmux = true

	return p
}
func (pt *ProxyTunnel) Start() (err error) {
	pt.On = true

	go pt.Server()
	return
}
func (pt *ProxyTunnel) Server() (err error) {
	defer func() {
		pt.On = false
	}()

	if pt.protocl == nil {
		return errors.New("no protocol set in ProxyTunnel")
	}
	listener := pt.protocl.GetListener()
	if listener == nil {
		return errors.New("protocol.listenre is null !!!")
	}

	if pt.protocl.GetConfig().ProxyType == "kcp" {
		gs.Str(pt.GetConfig().ID + "|" + pt.GetConfig().ProxyType + "| addr:" + pt.GetConfig().RemoteAddr()).Println("Start Single Kcp")
		for {
			con, err := listener.Accept()
			if err != nil {
				gs.Str(err.Error()).Color("r", "B").Println("kcp err")
			}
			smuxc := prosmux.NewSmuxServerNull()
			smuxc.SetHandler(func(con net.Conn) (err error) {
				pt.HandleConnAsync(con)
				return
			})

			go smuxc.AccpetStream(con)
			// if err != nil {
			// 	return err
			// }
			// pt.HandleConnAsync(con)
		}
	} else if pt.UseSmux {
		gs.Str(pt.GetConfig().ID + "|" + pt.GetConfig().ProxyType + "| addr:" + pt.GetConfig().RemoteAddr()).Println("Start Smux Tunnel")
		smux := prosmux.NewSmuxServer(listener, func(con net.Conn) (err error) {
			pt.HandleConnAsync(con)
			return
		})
		return smux.Server()
	} else {
		gs.Str(pt.GetConfig().ID + "|" + pt.GetConfig().ProxyType + "| addr:" + pt.GetConfig().RemoteAddr()).Println("Start Tunnel")
		for {
			con, err := listener.Accept()
			if err != nil {
				return err
			}
			pt.HandleConnAsync(con)
		}
	}
}

func (pt *ProxyTunnel) SetProtocol(procol Protocol) {
	pt.protocl = procol

}

func (pt *ProxyTunnel) GetConfig() *ProtocolConfig {
	if pt.protocl == nil {
		return nil
	}
	return pt.protocl.GetConfig()
}

func (pt *ProxyTunnel) SetControllFunc(l func(rawHost string, con net.Conn) (err error)) {
	pt.ControllFunc = l
}

func (pt *ProxyTunnel) HandleConnAsync(con net.Conn) {
	con.SetReadDeadline(time.Now().Add(time.Minute))
	host, _, _, err := prosocks5.GetServerRequest(con)
	if err != nil {
		gs.Str(err.Error()).Println("GetServerRequest | err")
		ErrToFile("Server HandleConnection", err)
		con.Close()
		return
	} else {
		gs.Str(host).Println("host|ready")
	}

	pt.lock.Lock()
	pt.cons = pt.cons.Add(con)
	pt.alive += 1
	pt.lock.Unlock()
	defer func() {
		pt.lock.Lock()
		pt.alive -= 1
		pt.lock.Unlock()
	}()
	if gs.Str(host).StartsWith("R://") {
		if pt.ControllFunc != nil {
			if err := pt.ControllFunc(host, con); err != nil {
				ErrToFile("server controll func ", err)
			}
		}
	} else {
		pt.TcpNormal(host, con)
	}
}

func (pt *ProxyTunnel) TcpNormal(host string, con net.Conn) (err error) {
	remoteConn, err := net.Dial("tcp", host)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
			// log too many open file error
			// EMFILE is process reaches open file limits, ENFILE is system limit
			ErrToFile("dial error too many file!!:", err)
		} else {
			ErrToFile("tcp normal", err)
		}
		gs.Str(host + "|" + err.Error()).Println("host|failed")
		// log.Println("X connect to ->", host)
		return err
	}
	// gs.Str(host).Println("host|ok")
	// con.SetWriteDeadline(time.Now().Add(2 * time.Minute))
	_, err = con.Write(prosocks5.Socks5Confirm)
	if err != nil {
		ErrToFile("back con is break", err)
		remoteConn.Close()
	}
	gs.Str(host).Println("host|build")
	pt.Pipe(remoteConn, con)
	return
}

func (pt *ProxyTunnel) Pipe(p1, p2 net.Conn) {
	var wg sync.WaitGroup
	var wait = 5 * time.Second
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
