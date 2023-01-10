package prokcp

import (
	"crypto/sha1"
	"net"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

func ConnectKcp(addr string, config baseconnection.ProtocolConfig) (session net.Conn, err error) {
	_key := config.Password
	_salt := config.SALT
	key := pbkdf2.Key([]byte(_key), []byte(_salt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	// dial to the echo server
	session, err = kcp.DialWithOptions(addr, block, 10, 3)
	return
}

func ConnectKcpFirstBuf(dst string, config baseconnection.ProtocolConfig, firstbuf ...[]byte) (con net.Conn, reply []byte, err error) {
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
