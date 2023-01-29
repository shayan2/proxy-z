package servercontroll

import (
	"sync"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"gitee.com/dark.H/ProxyZ/connections/prokcp"
	"gitee.com/dark.H/ProxyZ/connections/protls"
	"gitee.com/dark.H/gs"
)

var (
	lock         = sync.RWMutex{}
	ErrTypeCount = gs.Dict[int]{
		"tls": 0,
		"kcp": 0,
	}
)

func GetProxy() *baseconnection.ProxyTunnel {
	if Tunnels.Count() == 0 {

		tunnel := NewProxy("kcp")
		AddProxy(tunnel)
		return tunnel
	} else {
		tunnel := Tunnels.Nth(0)
		return tunnel
	}
}

func AddProxy(c *baseconnection.ProxyTunnel) {
	lock.Lock()
	Tunnels = append(Tunnels, c)
	lock.Unlock()
}

func DelProxy(name string) (found bool) {

	e := gs.List[*baseconnection.ProxyTunnel]{}
	for _, tun := range Tunnels {
		if tun == nil {
			continue
		}
		if tun.GetConfig().ID == name {
			if num, ok := ErrTypeCount[tun.GetConfig().ProxyType]; ok {
				num += 1
				lock.Lock()
				ErrTypeCount[tun.GetConfig().ProxyType] = num
				lock.Unlock()
			}
			tun.SetWaitToClose()
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

func NewProxyByErrCount() (c *baseconnection.ProxyTunnel) {
	tlsnum := ErrTypeCount["tls"]
	kcpnum := ErrTypeCount["kcp"]
	if kcpnum < tlsnum {
		c = NewProxy("kcp")
	} else {
		c = NewProxy("tls")
	}
	AddProxy(c)
	return
}

func GetProxyByID(name string) (c *baseconnection.ProxyTunnel) {
	for _, tun := range Tunnels {
		if tun.GetConfig().ID == name {
			return tun
		} else {

		}
	}
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
