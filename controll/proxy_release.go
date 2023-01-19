package controll

import (
	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"gitee.com/dark.H/ProxyZ/connections/prokcp"
	"gitee.com/dark.H/ProxyZ/connections/protls"
	"gitee.com/dark.H/gs"
)

func GetProxy() *baseconnection.ProxyTunnel {
	if Tunnels.Count() == 0 {

		tunnel := NewProxy("tls")
		Tunnels = append(Tunnels, tunnel)
		return tunnel
	} else {
		tunnel := Tunnels.Nth(0)
		return tunnel
	}
}

func DelProxy(name string) (found bool) {

	e := gs.List[*baseconnection.ProxyTunnel]{}
	for _, tun := range Tunnels {
		if tun.GetConfig().ID == name {
			found = true
			continue
		} else {
			e = e.Add(tun)
		}
	}
	GLOCK.Lock()
	Tunnels = e
	GLOCK.Unlock()
	return
}

func NewProxy(tp string) *baseconnection.ProxyTunnel {
	switch tp {
	case "tls":
		config := baseconnection.RandomConfig()
		protocl := protls.NewTlsServer(config)
		tunel := baseconnection.NewProxyTunnel(protocl)
		return tunel
	case "kcp":
		config := baseconnection.RandomConfig()
		protocl := prokcp.NewKcpServer(config)
		tunel := baseconnection.NewProxyTunnel(protocl)
		return tunel
	default:
		config := baseconnection.RandomConfig()
		protocl := protls.NewTlsServer(config)
		tunel := baseconnection.NewProxyTunnel(protocl)
		return tunel
	}
}
