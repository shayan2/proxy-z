package prosmux

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/xtaci/smux"
)

type SmuxConfig struct {
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
	Listener     net.Listener
	ClientConn   net.Conn
	handleStream func(conn net.Conn) (err error)
}

func (kconfig *SmuxConfig) SetAsDefault() {
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

func NewSmuxServer(listener net.Listener, handle func(con net.Conn) (err error)) (s *SmuxConfig) {
	s = new(SmuxConfig)
	s.Listener = listener
	s.handleStream = handle
	s.SetAsDefault()
	return
}

func NewSmuxClient(conn net.Conn) (s *SmuxConfig) {
	s = new(SmuxConfig)
	// Create a multiplexer using smux
	// conf := s.GenerateConfig()
	s.ClientConn = conn
	s.SetAsDefault()
	return
}

func (s *SmuxConfig) NewConnnect() (con net.Conn, err error) {
	conf := s.GenerateConfig()
	mux, err := smux.Client(s.ClientConn, conf)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Create a new stream on the multiplexer
	stream, err := mux.OpenStream()
	if err != nil {
		return
	}

	return stream, err
}

func (kconfig *SmuxConfig) UpdateMode() {
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

func (kconfig *SmuxConfig) GenerateConfig() *smux.Config {
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

func (m *SmuxConfig) Server() (err error) {
	for {
		// Accept a TCP connection
		conn, err := m.Listener.Accept()
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		go m.AccpetStream(conn)
	}

	// return err
}

func (m *SmuxConfig) AccpetStream(conn net.Conn) (err error) {
	smuxconfig := m.GenerateConfig()
	err = smux.VerifyConfig(smuxconfig)
	if err != nil {
		panic(err)
	}

	// Use smux to multiplex the connection
	mux, err := smux.Server(conn, smuxconfig)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Use WaitGroup to wait for all streams to finish
	var wg sync.WaitGroup
	for {
		// Accept a new stream
		stream, err := mux.AcceptStream()
		if err != nil {
			fmt.Println(err)
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.handleStream(stream)
		}()
	}

	// Wait for all streams to finish before closing the multiplexer
	wg.Wait()
	mux.Close()

	return
}
