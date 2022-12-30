package prokcp

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"gitee.com/dark.H/gs"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

type KcpBase struct {
	kconfig          *KcpConfig
	config           *baseconnection.ProtocolConfig
	smuxConfig       *smux.Config
	Numconn          int
	aliveConn        int
	activateConn     int
	aliveReverseConn int
	kcpconnection    *kcp.UDPSession
	aliveRefreshRate time.Duration
	chScavenger      chan *smux.Session
	IsHeartBeat      bool
	Role             string
	Messages         chan string
	IfCompress       bool
	Plugin           string
	muxes            []struct {
		session *smux.Session
		ttl     time.Time
	}
	testmuxes chan struct {
		session *smux.Session
		ttl     time.Time
	}

	// RedirectBooks map[string]*Route
}

func (kcpBase *KcpBase) AddReverseCon() {
	kcpBase.aliveReverseConn++
}

func (kcpBase *KcpBase) DelReverseCon() {
	kcpBase.aliveReverseConn--
}

func (kcpBase *KcpBase) GetAliveReverseCon() int {
	return kcpBase.aliveReverseConn
}

func (kcpBase *KcpBase) SetRefreshRate(interval int) {
	kcpBase.aliveRefreshRate = time.Duration(interval) * time.Second
}

func (kcpBase *KcpBase) GetRefreshRate() time.Duration {
	if kcpBase.aliveRefreshRate == 0 {
		kcpBase.aliveRefreshRate = 2 * time.Second
	}
	return kcpBase.aliveRefreshRate
}

func (kcpBase *KcpBase) UpdateKcpConfig(kcpconn *kcp.UDPSession) {
	kcpconn.SetStreamMode(true)
	kcpconn.SetWriteDelay(false)
	kcpconn.SetNoDelay(kcpBase.kconfig.NoDelay, kcpBase.kconfig.Interval, kcpBase.kconfig.Resend, kcpBase.kconfig.NoCongestion)
	kcpconn.SetWindowSize(kcpBase.kconfig.SndWnd, kcpBase.kconfig.RcvWnd)
	kcpconn.SetMtu(kcpBase.kconfig.MTU)
	kcpconn.SetACKNoDelay(kcpBase.kconfig.AckNodelay)
}

func (kcpBase *KcpBase) GetSmuxConfig() *smux.Config {
	if kcpBase.smuxConfig == nil {
		kcpBase.smuxConfig = kcpBase.kconfig.GenerateConfig()
	}
	return kcpBase.smuxConfig
}

func (kcpBase *KcpBase) SetTunnelNum(n int) int {
	kcpBase.Numconn = n
	return kcpBase.Numconn
}

func (kcpBase *KcpBase) ReConnection() (session *smux.Session, err error) {
	config := kcpBase.config
	block := config.GeneratePassword()
	serverString := fmt.Sprintf("%s:%d", config.GetServerArray()[0], config.ServerPort)
	if connection, err := kcp.DialWithOptions(serverString, block, kcpBase.kconfig.DataShard, kcpBase.kconfig.ParityShard); err == nil {
		kcpBase.UpdateKcpConfig(connection)

		if kcpBase.IfCompress {
			if session, err = smux.Client(NewCompStream(connection), kcpBase.smuxConfig); err == nil {
				return session, nil
			}
		} else {
			if session, err = smux.Client(connection, kcpBase.smuxConfig); err == nil {
				return session, nil
			}
		}

	}
	return
}

func (serv *KcpBase) ListenUDP(ip string, port int) {
	// serverString := fmt.Sprintf("%s:%d", config.GetServerArray()[0], config.ServerPort)
	// ip, _, _ := net.SplitHostPort(l.Addr().String())
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP(ip),
	}
	c, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return
	}
	go serv.HandleUDP(c)
}

func (kcpBase *KcpBase) createConn(config *baseconnection.ProtocolConfig) (session *smux.Session, err error) {

	if kcpBase.config == nil {
		kcpBase.config = config
	}
	if kcpBase.smuxConfig == nil {
		kcpBase.smuxConfig = kcpBase.kconfig.GenerateConfig()

	}

	block := config.GeneratePassword()
	serverString := fmt.Sprintf("%s:%d", config.GetServerArray()[0], config.ServerPort)
	if connection, err := kcp.DialWithOptions(serverString, block, kcpBase.kconfig.DataShard, kcpBase.kconfig.ParityShard); err == nil {
		kcpBase.UpdateKcpConfig(connection)
		if kcpBase.IfCompress {
			if session, err = smux.Client(NewCompStream(connection), kcpBase.smuxConfig); err == nil {
				return session, nil
			}
		} else {
			if session, err = smux.Client(connection, kcpBase.smuxConfig); err == nil {
				return session, nil
			}
		}

	}
	return

	if err != nil {
		return nil, err
	}
	return session, nil

}

func (kcpBase *KcpBase) Activate(do_some func() error) error {
	kcpBase.activateConn++
	defer func() {
		kcpBase.activateConn--
	}()
	return do_some()
}

func (kcpBase *KcpBase) GetActivateConn() int {
	return kcpBase.activateConn
}

func (kcpBase *KcpBase) WaitConn(config *baseconnection.ProtocolConfig) *smux.Session {
	if config == nil {
		config = kcpBase.config
	}
	for {
		if session, err := kcpBase.createConn(config); err == nil {
			return session
		} else {
			if err.Error() == "listen udp4 :0: socket: too many open files" {

			} else {
				log.Println("re-connecting:", err)
			}

			time.Sleep(time.Second)
		}
	}
}

func (kcpBase *KcpBase) Init(config *baseconnection.ProtocolConfig) {
	// gs.S(kcpBase.kconfig).Println()
	// S.gs.Str(fmt.Sprint("Ds/Ps", kcp), fmt.Sprint("Conn/AutoExpire:", conn.Numconn, conn.AutoExpire)).Println()
	// S.Str("Start tunnel:", kcpBase.Numconn, "Config with:", kcpBase.config.Method, kcpBase.config.Password, kcpBase.config.Server.(string), kcpBase.config.ServerPort).Println()
	// S.Str("num:", kcpBase.Numconn).Println()
	kcpBase.Messages = make(chan string, 2)
	numconn := uint16(kcpBase.Numconn)
	kcpBase.muxes = make([]struct {
		session *smux.Session
		ttl     time.Time
	}, numconn)
	// kcpBase.testmuxes = make(chan struct {
	// 	session *smux.Session
	// 	ttl     time.Time
	// }, 5)

	if config == nil {
		config = kcpBase.config
	}

	if kcpBase.chScavenger == nil {
		kcpBase.chScavenger = make(chan *smux.Session, 256)
	}
	go scavenger(kcpBase.chScavenger, kcpBase.kconfig.ScavengeTTL)
}

func (kcpBase *KcpBase) SendMsg(pre string, content string) {
	if strings.HasPrefix(pre, ":") {
		gs.S("xxx" + "pre can not include ':' !!!").Println()
		return
	}
	kcpBase.Messages <- fmt.Sprintf("%s:%s", pre, content)
}

func (kcpBase *KcpBase) RecvMsg(filter ...string) (pre string, content string) {
	select {
	case msg := <-kcpBase.Messages:
		parts := strings.SplitN(msg, ":", 2)
		pre = parts[0]
		if len(filter) > 0 {
			if filter[0] == pre {
				content = parts[1]
			} else {
				kcpBase.Messages <- msg
			}
		}
		content = parts[1]
	default:
		time.Sleep(kcpBase.GetRefreshRate())
		content = ""
	}

	return
}

func (kcpBase *KcpBase) HeartBeatC(stream net.Conn) {
	kcpBase.IsHeartBeat = true
	defer func() {
		stream.Close()
		gs.S("~ Dead because hearbeat is stop").Println()
		os.Exit(0)
	}()
	for {
		now := time.Now()
		buf := make([]byte, 123)
		if !kcpBase.IsHeartBeat {
			gs.S("stop heart " + "Client").Println()
			break
		}
		if n, err := stream.Read(buf); err == nil {
			// time.Sleep(1 * time.Second)
			if string(buf[:n]) == "[BOARING]" {
				if kcpBase.activateConn <= 1 {
					time.Sleep(3 * time.Second)
				}
				if _, err := stream.Write([]byte("[ME TOO]")); err != nil {
					break
				}

			} else {
				if kcpBase.activateConn <= 1 {
					time.Sleep(kcpBase.GetRefreshRate())
				} else {
					time.Sleep(1 * time.Second)
				}

				kcpBase.SendMsg(stream.RemoteAddr().String(), string(buf[:n]))
				if _, err := stream.Write([]byte("OK")); err != nil {
					break
				}
				gs.S(now.Format(time.UnixDate) + string(buf[:n])).Println()
			}
		} else {
			gs.S("Heartbeat Err" + err.Error()).Println()
			kcpBase.IsHeartBeat = false
			break
		}
	}
	// kcpBase.IsHeartBeat = false
}

func (kcpBase *KcpBase) HeartBeatS(stream net.Conn, fromHost string, clearAfter func(h string)) {
	nowT := 0
	kcpBase.IsHeartBeat = true
	used := false
	defer stream.Close()
	for {
		// now := time.Now()
		buf := make([]byte, 1024)
		if !kcpBase.IsHeartBeat {
			gs.S("stop heart" + "server").Println()
			break
		}
		if pre, msg := kcpBase.RecvMsg(); msg == "" {
			if _, err := stream.Write([]byte("[BOARING]")); err == nil {
				if n, err := stream.Read(buf); err == nil {
					if kcpBase.activateConn > 1 {
						used = true
					}
					if string(buf[:n]) == "[ME TOO]" {

					} else {
						gs.S("close no me too error when write boaring!!").Println()
						break
					}
				} else {
					gs.S("close by remote so i closed too").Println()
					break
				}
			} else {
				gs.S("close some error" + err.Error()).Println()
				break
			}
		} else if msg == "[[EOF]]" && pre == "HEART" {
			gs.S("* * stop heart beet").Println()
			break
		} else {
			if msg == "more" {
				if _, err := stream.Write([]byte("9")); err == nil {

					n, err := stream.Read(buf)
					if err == nil {
						gs.S("reply msg:" + string(buf[:n])).Println()
						if string(buf[:n]) == "OK" {
							nowT += 9
						}
					} else {
						gs.S("close by remote so i closed too" + err.Error()).Println()
						break
					}
					// kcpBase.SendMsg(fromHost, "OK")
				} else {
					gs.S("close some error" + err.Error()).Println()
					break
				}
			}
		}

	}
	if used {
		clearAfter(fromHost)
	}
	// kcpBase.IsHeartBeat = false

}

func (kcpBase *KcpBase) UpdateConfig(uri string) {
	kcpBase.config = baseconnection.ParseURI(uri)
}

func (kcpBase *KcpBase) WithTestSession(config *baseconnection.ProtocolConfig, howTo func(sess *smux.Session)) {
	ss := new(struct {
		session *smux.Session
		ttl     time.Time
	})

	ss.session = kcpBase.WaitConn(config)
	ss.ttl = time.Now().Add(time.Duration(kcpBase.kconfig.AutoExpire) * time.Second)
	kcpBase.testmuxes <- *ss
	howTo(ss.session)
	<-kcpBase.testmuxes

}

func (kcpBase *KcpBase) WithSession(config *baseconnection.ProtocolConfig, id ...uint16) (session *smux.Session) {
	var idx uint16
	kcpBase.aliveConn++
	if len(id) > 0 {
		idx = id[0] % uint16(kcpBase.Numconn)
	}
	if config == nil {
		config = kcpBase.config
	}

	if kcpBase.muxes == nil {
		kcpBase.Init(config)
	}
	// Closed / Timeout / Session host incorrect will reconnect
	if kcpBase.muxes[idx].session == nil {
		kcpBase.chScavenger <- kcpBase.muxes[idx].session
		kcpBase.muxes[idx].session = kcpBase.WaitConn(config)
		kcpBase.muxes[idx].ttl = time.Now().Add(time.Duration(kcpBase.kconfig.AutoExpire) * time.Second)
	} else {
		if !strings.HasPrefix(kcpBase.muxes[idx].session.RemoteAddr().String(), config.Server.(string)) {
			kcpBase.chScavenger <- kcpBase.muxes[idx].session
			kcpBase.muxes[idx].session = kcpBase.WaitConn(config)
			kcpBase.muxes[idx].ttl = time.Now().Add(time.Duration(kcpBase.kconfig.AutoExpire) * time.Second)
		} else if kcpBase.muxes[idx].session.IsClosed() || (kcpBase.kconfig.AutoExpire > 0 && time.Now().After(kcpBase.muxes[idx].ttl)) {

			kcpBase.chScavenger <- kcpBase.muxes[idx].session
			kcpBase.muxes[idx].session = kcpBase.WaitConn(config)
			kcpBase.muxes[idx].ttl = time.Now().Add(time.Duration(kcpBase.kconfig.AutoExpire) * time.Second)
		}
		// else if kcpBase.muxes[idx].session.RemoteAddr().String() != config.Server.(string) {
		// 	sss := kcpBase.WaitConn(config)
		// 	kcpBase.chScavenger <- sss
		// 	return sss
		// }
	}

	session = kcpBase.muxes[idx].session
	return
}

func (kcpBase *KcpBase) GetSession(idx uint16) (session *smux.Session) {
	if !kcpBase.muxes[idx].session.IsClosed() {
		session = kcpBase.muxes[idx].session
	}
	return
}

func (kcpBase *KcpBase) SetConfig(config *baseconnection.ProtocolConfig) {
	kcpBase.config = config
}

func (kcpBase *KcpBase) SetKcpConfig(config *KcpConfig) {
	kcpBase.kconfig = config
}

func (kcpBase *KcpBase) GetConfig() *baseconnection.ProtocolConfig {
	// if kcpBase.config.SSPassword == "" {
	if kcpBase.Plugin == "ss" {
		kcpBase.config.SSPassword = kcpBase.config.Password
		kcpBase.config.SSMethod = "aes-256-gcm"

	}
	// }

	return kcpBase.config
}

func (kcpBase *KcpBase) GetKcpConfig() *KcpConfig {
	return kcpBase.kconfig
}

func (kcpBase *KcpBase) GetAliveNum() int {
	return kcpBase.aliveConn
}

func (kcpBase *KcpBase) ShowConfig() {
	gs.S(kcpBase.kconfig).Println()
	gs.S(kcpBase.GetSmuxConfig()).Println()
	gs.S("Connection num:%d").F(kcpBase.Numconn).Println()
}

// func (kcpBase *KcpBase) PipeTest(p1, p2 net.Conn) (p1data, p2data []byte) {
// 	p1.

// }

func (kcpBase *KcpBase) Pipe(p1, p2 net.Conn) {
	// start tunnel & wait for tunnel termination
	// p1.SetWriteDeadline(5 * time.Second)
	// p2.SetWriteDeadline(5 * time.Second)
	streamCopy := func(dst net.Conn, src io.ReadCloser, fr, to net.Addr) {
		// startAt := time.Now()
		baseconnection.Copy(dst, src)
		dst.SetWriteDeadline(time.Now().Add(15 * time.Second))
		dst.Close()
		// }()
	}
	go streamCopy(p1, p2, p2.RemoteAddr(), p1.RemoteAddr())
	streamCopy(p2, p1, p1.RemoteAddr(), p2.RemoteAddr())
	kcpBase.aliveConn--
}

func (kcpBase *KcpBase) AddAlive() {
	kcpBase.aliveConn++
}

type scavengeSession struct {
	session *smux.Session
	ts      time.Time
}

func scavenger(ch chan *smux.Session, ttl int) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var sessionList []scavengeSession
	for {
		select {
		case sess := <-ch:
			sessionList = append(sessionList, scavengeSession{sess, time.Now()})
			// log.Println("session marked as expired")
			// log.Println("session marked as expired", sess.RemoteAddr())
		case <-ticker.C:
			var newList []scavengeSession
			for k := range sessionList {
				s := sessionList[k]
				if s.session == nil {
					continue
				}
				if s.session.NumStreams() == 0 || s.session.IsClosed() {
					// log.Println("session normally closed", s.session.RemoteAddr())
					// log.Println("session normally closed")
					s.session.Close()
				} else if ttl >= 0 && time.Since(s.ts) >= time.Duration(ttl)*time.Second {
					// log.Println("session reached scavenge ttl", s.session.RemoteAddr())
					// log.Println("session reached scavenge ttl")
					s.session.Close()
				} else {
					newList = append(newList, sessionList[k])
				}
			}
			sessionList = newList
		}
	}
}
