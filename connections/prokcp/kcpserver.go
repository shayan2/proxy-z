package prokcp

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gitee.com/dark.H/go-remote-repl/datas"
	"github.com/fatih/color"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	// "github.com/cs8425/smux"
)

const (
	idType  = 0 // address type index
	idIP0   = 1 // ip address start index
	idDmLen = 1 // domain address length index
	idDm0   = 2 // domain address start index

	typeIPv4     = 1 // type is ipv4 address
	typeDm       = 3 // type is domain address
	typeIPv6     = 4 // type is ipv6 address
	typeRedirect = 9

	lenIPv4        = net.IPv4len + 2 // ipv4 + 2port
	lenIPv6        = net.IPv6len + 2 // ipv6 + 2port
	lenDmBase      = 2               // 1addrLen + 2port, plus addrLen
	AddrMask  byte = 0xf
	// lenHmacSha1 = 10
)

var (
	debug                 bool
	sanitizeIps           bool
	udp                   bool
	managerAddr           string
	smuxConfig            = smux.DefaultConfig()
	Socks5ConnectedRemote = []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43}
)

type Channel struct {
	stream net.Conn
	host   string
}

func newChannel(stream net.Conn, host string) Channel {
	return Channel{
		stream: stream,
		host:   host,
	}
}

// KcpServer used for server
type KcpServer struct {
	utils.KcpBase
	// RedirectMode  bool
	TunnelChan     chan Channel
	TcpListenPorts map[string]int
	AcceptConn     int
	// RedirectBook  *utils.Config
}

// NewKcpServer init KcpServer
func NewKcpServer(config *utils.Config, kconfig *utils.KcpConfig) (kcpServer *KcpServer) {
	kcpServer = new(KcpServer)
	kcpServer.RedirectBooks = make(map[string]*utils.Route)
	kcpServer.TunnelChan = make(chan Channel, 50)
	kcpServer.TcpListenPorts = make(map[string]int)
	log.Println("create serve in :", config.Server, "port:", config.ServerPort, "with pwd:", config.Password)
	kcpServer.SetConfig(config)
	kcpServer.SetKcpConfig(kconfig)

	return
}

// Listen for kcp
func (serve *KcpServer) Listen() {
	configcopy := serve.GetConfig()
	config := &utils.Config{}
	utils.DeepCopy(config, configcopy)
	var block kcp.BlockCrypt
	if _, ok := datas.MemDB.Kd["ip who use"]; !ok {
		datas.MemDB.Kd["ip who use"] = make(datas.Gi)
	}
	// tls server listener
	if config.Method == "tls" {
		utils.ColorL("<======= Tls Server Mode =======>")
		serve.ListenInTls(config)
		return
	} else {
		utils.ColorL("<======", config.Method, "=======>")
	}
	if serve.Plugin == "ss" {
		utils.ColorL("Shadowsocks set", config.Password, config.ServerPort)
		block = config.GeneratePassword("ss")
	} else {
		block = config.GeneratePassword()
	}
	kconfig := serve.GetKcpConfig()
	// kconfig := smux.DefaultConfig()

	severString := fmt.Sprintf("%s:%d", config.GetServerArray()[0], config.ServerPort)
	serve.ShowConfig()
	if listener, err := kcp.ListenWithOptions(severString, block, kconfig.DataShard, kconfig.ParityShard); err == nil {
		listener.SetReadBuffer(kconfig.SocketBuf)
		listener.SetWriteBuffer(kconfig.SocketBuf)

		listener.SetDSCP(0)
		g := color.New(color.FgGreen)
		g.Printf("accept ready \r")
		// ccCount := 0
		for {
			conn, err := listener.AcceptKCP()
			serve.AcceptConn++
			g.Printf("\rAlive: %d/%d  Time:%s \r", serve.GetAliveNum(), serve.AcceptConn, time.Now().String()[:20])
			// g.Println("new con:", conn.RemoteAddr())
			serve.UpdateKcpConfig(conn)
			if err != nil {
				if !strings.Contains(err.Error(), "too many open files") {
					log.Fatal(err)
				}
				continue
			}
			// ccCount++
			if serve.IfCompress {
				utils.ColorL("compress")
				go serve.ListenMux(general.NewCompStream(conn))
			} else {
				utils.ColorL("no compress")
				go serve.ListenMux(conn)
			}
		}
	} else {
		log.Fatal(err)
	}
}

func (serve *KcpServer) ListenMux(conn io.ReadWriteCloser) {
	sconfig := serve.GetSmuxConfig()

	mux, err := smux.Server(conn, sconfig)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		mux.Close()
		serve.AcceptConn--
	}()
	var rr uint16
	for {
		if stream, err := mux.AcceptStream(); err == nil {
			serve.AddAlive()
			// utils.ColorL("ListenMux")
			go serve.handleStream(rr, stream)
		} else {
			break
		}
		rr++
		rr %= uint16(serve.Numconn)

	}
	// utils.ColorL("exit mux")
}

func (serve *KcpServer) needMoreTunnel(stream *smux.Stream) {
	for {

	}
}

func (serve *KcpServer) ListenInTls(config *utils.Config) {
	if tlsConfig, err := config.ToTlsConfig(); err != nil {
		log.Fatal("config -> tls config error:", err)
	} else {
		if listener, err := tlsConfig.WithTlsListener(); err != nil {
			log.Fatal(err)
		} else {
			var rr uint16
			for {
				conn, err := listener.Accept()
				if err != nil {
					utils.ColorL("server: accept: %s", err)
					break
				}
				// defer conn.Close()
				go func() {
					log.Printf("server: accepted from %s", conn.RemoteAddr())
					tlsConn, _ := conn.(*tls.Conn)
					if serve.IfCompress {
						serve.handleStream(rr, general.NewCompStream(tlsConn))
					} else {
						serve.handleStream(rr, tlsConn)
					}

				}()
				rr++
				rr %= uint16(serve.Numconn)
			}
		}
	}
}

func (serve *KcpServer) handleStream(rr uint16, stream net.Conn) error {
	// g := color.New(color.FgGreen)
	// utils.ColorL("incomming :", stream.RemoteAddr())
	var host string
	var raw []byte
	var isUdp bool
	var err error

	// utils.ColorL("server request")

	config := serve.GetConfig()

	switch serve.Plugin {
	case "ss":
		var password string
		if config.SSPassword == "" {
			password = config.Password
		} else {
			password = config.SSPassword
		}
		key := []byte{}
		cipher := config.SSMethod
		if cipher == "" {
			cipher = "aes-256-gcm"
		}
		ciph, _ := PickCipher(cipher, key, password)
		stream = ciph.StreamConn(stream)
		utils.ColorL("ss", password)
		host, raw, isUdp, err = utils.GetSSServerRequest(stream)
	default:
		host, raw, isUdp, err = utils.GetServerRequest(stream)
	}
	// utils.ColorL("raw:", raw)

	// utils.ColorL("start handle stream")
	// host, raw, isUdp, err := utils.GetServerRequest(stream)
	// utils.ColorL("err:", err)
	// if err != nil {
	// 	config := serve.GetConfig()

	// 	password := config.SSPassword
	// 	key := []byte{}
	// 	cipher := config.SSMethod
	// 	ciph, _ := PickCipher(cipher, key, password)

	// 	stream2 := ciph.NewStreamConn(stream)
	// 	// stream = ciph.StreamConn(stream)
	// 	// utils.ColorL("raw:", raw)
	// 	raw, host, isUdp, err = stream2.ParseSSHeader(raw)
	// 	fmt.Println("ss de:", host, "raw", raw)
	// 	stream2.LastAhead = nil
	// 	stream = ciph.StreamConn(stream)
	// 	// host, raw, isUdp, err := utils.GetSSServerRequest(stream)
	// }

	fromHost := strings.Split(stream.RemoteAddr().String(), ":")[0]
	datas.MemDB.Kd["ip who use"][fromHost] = time.Now().String()
	if err != nil {
		log.Println("getRequest:", err)
		stream.Close()
		return err
	}
	// if host != "TUNNEL_CONNECT" {
	// 	g.Println("Accept stream req: | ", host, "| isUdp: ", isUdp)
	// }
	if strings.HasPrefix(host, "redirect://") {
		serve.handleBook(fromHost, host, stream)
	} else {
		// jump to other server
		if b, ok := serve.RedirectBooks[fromHost]; ok {
			// g.Printf("proxy %s \n", fromHost)
			if b.IfNoExpired() {
				switch b.Mode() {
				case "tunnel":
					if host == "TUNNEL_CONNECT" {
						go func() {
							serve.AddReverseCon()
							serve.TunnelChan <- newChannel(stream, fromHost)
						}()
						// serve.TunnelMap[fromHost] = stream
					} else {
						if isUdp {
							serve.handleRemoteUDP(stream, host)
						} else {
							serve.handleRemote(stream, host)
						}

					}
				case "connect":
					if host == "TUNNEL_CONNECT" {
						go func() {
							serve.AddReverseCon()
							serve.TunnelChan <- newChannel(stream, fromHost)
						}()
						// serve.TunnelMap[fromHost] = stream
					} else {
						if isUdp {
							serve.handleRemoteUDP(stream, host)
						} else {
							serve.handleRemote(stream, host)
						}
					}

				default:
					config := b.GetConfig()
					if config.Method != "tls" {
						session := serve.WithSession(config, rr)
						utils.ColorL("forward ->", config.Server.(string))
						serve.handleSession(session, stream, true, raw)
					} else {
						tconfig, err := config.ToTlsConfig()
						if err != nil {
							utils.ColorE(err)
						}
						redirectConn, err := tconfig.WithConn()
						if err != nil {
							utils.ColorE(err)
						}
						fmt.Println("route ->", utils.FGCOLORS[2](config.Server.(string)))
						serve.handleSessionCon(redirectConn, stream, true, raw)
					}

				}

			} else {
				delete(serve.RedirectBooks, fromHost)
				utils.ColorL("route expired: ", fromHost)
				if isUdp {
					serve.handleRemoteUDP(stream, host)
				} else {
					serve.handleRemote(stream, host)
				}
			}

		} else {
			if isUdp {
				serve.handleRemoteUDP(stream, host)
			} else {
				serve.handleRemote(stream, host)
			}
		}

	}
	return nil
}

func (serve *KcpServer) StopTunnel(host string) {
	delete(serve.TcpListenPorts, host)
	delete(serve.RedirectBooks, host)

	serve.IsHeartBeat = false
	for {
		if len(serve.TunnelChan) > 0 {
			for i := 0; i < len(serve.TunnelChan); i++ {
				c := <-serve.TunnelChan
				c.stream.Close()
			}
			time.Sleep(100 * time.Microsecond)
		} else {
			break
		}
	}

	utils.ColorL("release", "tunnel:", len(serve.TunnelChan))
}

func (serve *KcpServer) WithChannel(host, connectTo string) Channel {
	if len(serve.TunnelChan) == 0 {
		utils.ColorL("try to send more")
		serve.SendMsg(host, "more")
	}
	for { //i := 0; i < len(serve.TunnelChan); i++ {
		ch := <-serve.TunnelChan
		// if ch.stream.
		if ch.host == connectTo {
			return ch
		} else {
			serve.TunnelChan <- ch
		}
	}
	// channel := <-serve.TunnelChan
	// return channel
	// return nil
}

func (serve *KcpServer) handleConnect(connectTo, host, cmd string, stream net.Conn) (err error) {
	// err = errors.New("NO Valid CHANNEL SWITCH !!")
	testFirstdata := make([]byte, 1122)
	if readN, rerr := stream.Read(testFirstdata); rerr != nil {
		utils.ColorL("Channel Pipe test read error , try agin to get a new channel\nError: ", rerr)
		// continue
		err = rerr
		return
	} else {
		utils.ColorL("Test Read <<OK>>:", testFirstdata[:readN])
		channel := serve.WithChannel(host, connectTo)
		testWriteN := 0
		channel.stream.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if testWriteN, err = channel.stream.Write(testFirstdata[:readN]); err != nil || testWriteN < 1 {
			utils.ColorL("Channel Pipe test write error , try agin to get a new channel\nError: ", err)
			for i := 0; i < len(serve.TunnelChan); i++ {
				channel := serve.WithChannel(host, connectTo)
				channel.stream.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if testWriteN, err := channel.stream.Write(testFirstdata[:readN]); err != nil || testWriteN < 1 {
					utils.ColorL("Channel Pipe test write error , try agin to get a new channel\nError: ", err)
					continue
				}
				utils.ColorL("Test Write <<OK>>", testWriteN)
				go serve.Pipe(stream, channel.stream)
				err = nil
				break

			}
		} else {
			utils.ColorL("Test Write <<OK>>:", testWriteN)
			go serve.Pipe(stream, channel.stream)
			err = nil
			return
		}

	}
	if err != nil {
		utils.ColorL("Test Write <<FAILD>>")
	}

	return

	// if channel.host == connectTo {
	// 	utils.ColorL("connect -> tunnel")
	// 	go utils.Pipe(stream, channel.stream)

	// } else {
	// 	utils.ColorL("channel match failed", channel.host, connectTo)
	// 	serve.TunnelChan <- channel
	// }
}

func (serve *KcpServer) handleBook(sockHost, cmd string, stream net.Conn) {
	g := color.New(color.FgGreen)
	data := serve.bookHandle(sockHost, cmd)
	if cmd == "redirect://TUNNEL_INIT" {
		stream.Write([]byte(sockHost))
	} else if cmd == "redirect://ls" || cmd == "redirect://keys" {

		if _, err := stream.Write(data); err != nil {
			g.Println("book angry:", err)
		} else {
			g.Println("book say:", string(data))
		}
		stream.Close()
		return
	}

	if m, ok := serve.RedirectBooks[sockHost]; ok {

		if m.Mode() == "tunnel" {
			if _, ok := serve.TcpListenPorts[sockHost]; !ok {
				serve.IsHeartBeat = true
				utils.ColorL("HeartBeat", "Startup")
				go serve.HeartBeatS(stream, sockHost, serve.StopTunnel)

				// serve.tcpToStream(sockHost)
				// serve.StopTunnel(sockHost)
				utils.ColorL("to normal mode")
				return
			}
		} else if m.Mode() == "connect" {
			connectTo := strings.TrimSpace(m.GetConfig().Server.(string))
			fmt.Println(utils.FGCOLORS[3](sockHost, "=>", connectTo))
			serve.handleConnect(connectTo, sockHost, cmd, stream)
			return
		}
	}

	if _, err := stream.Write(data); err != nil {
		g.Println("book angry:", err)
	} else {
		g.Println("book say:", string(data))
	}
	if cmd == "redirect://kill-my-life" {
		os.Exit(0)
	}
	// go func() {
	// 	time.Sleep(1 * time.Second)
	// 	stream.Close()
	// }()
	stream.Close()
}

func (serve *KcpServer) bookHandle(host, cmd string) (repl []byte) {
	// log.Println("book cmd:", cmd)

	if cmd == "redirect://stop" {
		if v, ok := serve.RedirectBooks[host]; ok {
			serve.StopTunnel(host)
			repl = []byte(fmt.Sprintf("stop redirection:%s", v.GetConfig().Server.(string)))
		} else {
			repl = []byte("no redirection to stop")
		}
		return
		// serve.RedirectMode = false
		// repl = []byte("stop redirection")
	} else if cmd == "redirect://kill-my-life" {
		repl = []byte("kill server mode")
		// os.Exit(0)
		return
	} else if cmd == "redirect://TUNNEL_INIT" {
		if route, err := utils.NewRoute("TUNNEL"); err == nil {
			route.SetConfig(&utils.Config{
				Server: host,
			})
			serve.RedirectBooks[host] = route
			utils.ColorL("redirect +", host)
		}
	}

	main := strings.Split(cmd, "redirect://")[1]
	if strings.HasPrefix(main, "ss://") {
		utils.BOOK.Add(main)
		repl = []byte(fmt.Sprintf("add %s to book", main))
	} else if strings.HasPrefix(main, "cc://") {
		if route, err := utils.NewRoute(main); err == nil {
			utils.ColorL("connect -> tunnel:", route.GetConfig().Server.(string))
			serve.RedirectBooks[host] = route
		}
	} else if main == "keys" {
		cc := []string{}
		for k := range serve.RedirectBooks {
			cc = append(cc, k)
		}
		if repl, err := json.Marshal(cc); err != nil {
			repl = []byte(err.Error())
		} else {
			return repl
		}
	} else if strings.HasPrefix(main, "ls") {
		data := new(struct {
			Data []string
			Left int
			Used string
			Mode string
		})

		if ls, err := utils.BOOK.Ls(); err == nil {
			data.Data = ls
			if v, ok := serve.RedirectBooks[host]; ok {
				data.Left = v.Left()
				data.Used = v.GetConfig().Server.(string)
				data.Mode = v.Mode()
			}
			if repl, err = json.Marshal(data); err != nil {
				repl = []byte(err.Error())
			} else {
				return repl
			}

		} else {
			repl = []byte(fmt.Sprintf("%v", err))
		}

	} else if strings.HasPrefix(main, "start") {

		if route, err := utils.NewRoute(main); err == nil {
			serve.RedirectBooks[host] = route
			repl = []byte(host + "redirect -> " + route.Host())
		} else {
			repl = []byte(err.Error())
		}
	}
	return
}

func (serve *KcpServer) handleSessionCon(p2, p1 net.Conn, quiet bool, data []byte) {
	defer p1.Close()

	defer p2.Close()
	if len(data) > 0 {
		p2.Write(data)
	}
	serve.Pipe(p1, p2)
}

func (serve *KcpServer) handleSession(session *smux.Session, p1 net.Conn, quiet bool, data []byte) {
	defer p1.Close()
	p2, err := session.OpenStream()
	if err != nil {
		return
	}
	if len(data) > 0 {
		if _, err = p2.Write(data); err != nil {
		}
	}
	defer p2.Close()
	serve.Pipe(p1, p2)
}

// handleEcho send back everything it received
func (serve *KcpServer) handleRemote(conn net.Conn, host string) {
	// utils.ColorL("func:", "handleRemote")

	closed := false
	if strings.ContainsRune(host, 0x00) {
		log.Println("invalid domain name.")
		closed = true
		return
	}
	num := serve.GetAliveNum()
	remote, err := net.Dial("tcp", host)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
			// log too many open file error
			// EMFILE is process reaches open file limits, ENFILE is system limit
			log.Println(fmt.Sprintf("%d/%d", num, serve.AcceptConn), "dial error too many file!!:", err)
		} else {
			log.Println(fmt.Sprintf("%d/%d", num, serve.AcceptConn), "handleRemote", host, "Err", err)
		}
		// log.Println("X connect to ->", host)
		return
	}

	switch serve.Plugin {
	case "ss":
		utils.ColorL(fmt.Sprintf("%d/%d", num, serve.AcceptConn), "handleRemote", host, "Shadowsocks", "ok")
	default:
		_, err = conn.Write(Socks5ConnectedRemote)
		if err != nil {
			utils.ColorL(fmt.Sprintf("%d/%d", num, serve.AcceptConn), "Err", err)
		}
		utils.ColorL(fmt.Sprintf("%d/%d", num, serve.AcceptConn), "handleRemote", host, "ok")

	}

	defer func() {
		if !closed {
			remote.Close()
		}
	}()

	serve.Pipe(conn, remote)

	// go utils.LogCopy(conn, remote)
	// utils.LogCopy(remote, conn)
}

// handleEcho send back everything it received
func (serve *KcpServer) handleRemoteUDP(conn net.Conn, host string) {
	closed := false
	if strings.ContainsRune(host, 0x00) {
		log.Println("invalid domain name.")
		closed = true
		return
	}
	ts := strings.SplitN(host, ":", 2)
	port, _ := strconv.Atoi(ts[1])
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	dstAddr := &net.UDPAddr{IP: net.ParseIP(ts[0]), Port: port}

	remote, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
			// log too many open file error
			// EMFILE is process reaches open file limits, ENFILE is system limit
			log.Println("dial error:", err)
		} else {
			log.Println("error connecting to:", host, err)
		}
		log.Println("X connect to ->", host)
		return
	}
	log.Println("connect to ->", host)

	defer func() {
		if !closed {
			remote.Close()
		}
	}()

	serve.Pipe(conn, remote)

	// go utils.LogCopy(conn, remote)
	// utils.LogCopy(remote, conn)
}
