package prokcp

import (
	"log"
	"time"

	"github.com/xtaci/smux"
)

type KcpConfig struct {
	Mode         string `json:"mode"`
	NoDelay      int    `json:"nodelay"`
	Interval     int    `json:"interval"`
	Resend       int    `json:"resend"`
	NoCongestion int    `json:"nocongeestion"`
	AutoExpire   int    `json:"autoexpire"`
	ScavengeTTL  int    `json:"scavengettl"`
	MTU          int    `json:"mtu"`
	SndWnd       int    `json:"sndwnd"`
	RcvWnd       int    `json:"rcvwnd"`
	DataShard    int    `json:"datashard"`
	ParityShard  int    `json:"parityshard"`
	KeepAlive    int    `json:"keepalive"`
	SmuxBuf      int    `json:"smuxbuf"`
	StreamBuf    int    `json:"streambuf"`
	AckNodelay   bool   `json:"acknodelay"`
	SocketBuf    int    `json:"socketbuf"`
}

func (kconfig *KcpConfig) SetAsDefault() {
	// kconfig.Mode = "fast4"
	kconfig.KeepAlive = 10
	kconfig.MTU = 1350
	kconfig.DataShard = 10
	kconfig.ParityShard = 3
	kconfig.SndWnd = 2048 * 2
	kconfig.RcvWnd = 2048 * 2
	kconfig.ScavengeTTL = 600
	kconfig.AutoExpire = 7
	kconfig.SmuxBuf = 4194304 * 2
	kconfig.StreamBuf = 2097152 * 2
	kconfig.AckNodelay = false
	kconfig.SocketBuf = 4194304 * 2
}

func (kconfig *KcpConfig) UpdateMode() {
	// kconfig.Mode = mode
	switch kconfig.Mode {
	case "normal":
		kconfig.NoDelay, kconfig.Interval, kconfig.Resend, kconfig.NoCongestion = 0, 40, 2, 1
	case "fast":
		kconfig.NoDelay, kconfig.Interval, kconfig.Resend, kconfig.NoCongestion = 0, 30, 2, 1
	case "fast2":
		kconfig.NoDelay, kconfig.Interval, kconfig.Resend, kconfig.NoCongestion = 1, 20, 2, 1
	case "fast3":
		kconfig.NoDelay, kconfig.Interval, kconfig.Resend, kconfig.NoCongestion = 1, 10, 2, 1

	case "fast4":
		kconfig.NoDelay, kconfig.Interval, kconfig.Resend, kconfig.NoCongestion = 1, 5, 2, 1
	}
	// ColorL("kcp mode", kconfig.Mode)
}

func (kconfig *KcpConfig) GenerateConfig() *smux.Config {
	smuxConfig := smux.DefaultConfig()
	kconfig.UpdateMode()
	// smuxConfig.Version = 2
	smuxConfig.MaxReceiveBuffer = kconfig.SmuxBuf
	smuxConfig.MaxStreamBuffer = kconfig.StreamBuf
	smuxConfig.KeepAliveInterval = time.Duration(kconfig.KeepAlive) * time.Second
	if err := smux.VerifyConfig(smuxConfig); err != nil {
		log.Fatalf("%+v", err)
	}
	return smuxConfig
}
