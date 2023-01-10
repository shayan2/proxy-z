package prokcp

import (
	"crypto/sha1"
	"net"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"golang.org/x/crypto/pbkdf2"
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
	Server string
	config baseconnection.ProtocolConfig
	// RedirectMode  bool
	TunnelChan     chan Channel
	TcpListenPorts map[string]int
	AcceptConn     int
	// RedirectBook  *utils.Config
}

func (ksever *KcpServer) Accept() (con net.Conn, err error) {
	_key := ksever.config.Password
	_salt := ksever.config.SALT
	key := pbkdf2.Key([]byte(_key), []byte(_salt), 4096, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	var listener net.Listener
	listener, err = kcp.ListenWithOptions(ksever.Server, block, 10, 3)
	if err != nil {
		return nil, err
	}
	return listener.Accept()
}
