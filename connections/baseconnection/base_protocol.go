package baseconnection

import (
	"net"
	"sync"
	"syscall"
	"time"

	"gitee.com/dark.H/gs"
)

type Protocol interface {
	GetListener() net.Listener
	GetConfig() ProtocolConfig
}

type ProxyTunnel struct {
	cons         gs.List[net.Conn]
	alive        int
	lock         sync.RWMutex
	protocl      Protocol
	ControllFunc func(rawHost string, con net.Conn) (err error)
}

func (pt *ProxyTunnel) SetProtocol(procol Protocol) {
	pt.protocl = procol

}

func (pt *ProxyTunnel) SetControllFunc(l func(rawHost string, con net.Conn) (err error)) {
	pt.ControllFunc = l
}

func (pt *ProxyTunnel) HandleConnAsync(con net.Conn) {
	con.SetReadDeadline(time.Now().Add(time.Minute))
	host, _, _, err := GetServerRequest(con)
	if err != nil {
		ErrToFile("Server HandleConnection", err)
		con.Close()
		return
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
		// log.Println("X connect to ->", host)
		return err
	}
	// con.SetWriteDeadline(time.Now().Add(2 * time.Minute))
	_, err = con.Write(Socks5Confirm)
	if err != nil {
		ErrToFile("back con is break", err)
		remoteConn.Close()
	}

	pt.Pipe(remoteConn, con)
	return
}

func (pt *ProxyTunnel) Pipe(p1, p2 net.Conn) {
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
