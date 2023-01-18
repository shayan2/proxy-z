package protls

import (
	"net"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
)

func ConnectTls(dst string, config *baseconnection.ProtocolConfig) (con net.Conn, err error)
