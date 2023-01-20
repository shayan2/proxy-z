package prokcp

import (
	"crypto/sha1"
	"net"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

func ConnectKcp(addr string, config *baseconnection.ProtocolConfig) (conn net.Conn, err error) {
	_key := config.Password
	_salt := config.SALT
	key := pbkdf2.Key([]byte(_key), []byte(_salt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	// dial to the echo server

	MTU := 1350
	DataShard := 10
	ParityShard := 3
	SndWnd := 2048 * 2
	RcvWnd := 2048 * 2
	AckNodelay := false
	kcpconn, err := kcp.DialWithOptions(addr, block, DataShard, ParityShard)
	switch config.Type {
	case "normal":
		// kconfig.NoDelay, kconfig.Interval, kconfig.Resend, kconfig.NoCongestion = 0, 40, 2, 1
		kcpconn.SetNoDelay(0, 40, 2, 1)
	case "fast":
		kcpconn.SetNoDelay(0, 30, 2, 1)

	case "fast2":
		kcpconn.SetNoDelay(0, 20, 2, 1)

	case "fast3":
		kcpconn.SetNoDelay(1, 10, 2, 1)

	case "fast4":

		kcpconn.SetNoDelay(1, 5, 2, 1)
	}
	kcpconn.SetStreamMode(true)
	kcpconn.SetWriteDelay(false)

	kcpconn.SetWindowSize(SndWnd, RcvWnd)
	kcpconn.SetMtu(MTU)
	kcpconn.SetACKNoDelay(AckNodelay)

	return kcpconn, nil
}

func ConnectKcpFirstBuf(dst string, config *baseconnection.ProtocolConfig, firstbuf ...[]byte) (con net.Conn, reply []byte, err error) {
	con, err = ConnectKcp(dst, config)

	if firstbuf != nil {

		con.Write(firstbuf[0])
		buf := make([]byte, 8096)
		n, err := con.Read(buf)

		if err != nil {
			return nil, nil, err
		}
		reply = make([]byte, n)
		copy(reply, buf[:n])
		return con, reply, nil
	}

	return
}
