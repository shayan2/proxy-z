package protls

import (
	"crypto/tls"
	"net"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
)

func ConnectTls(dst string, config *baseconnection.ProtocolConfig) (con net.Conn, err error) {
	tlsconfig, _ := config.GetTlsConfig()
	return tls.Dial("tcp", dst, tlsconfig)
}
