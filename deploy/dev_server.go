package deploy

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"time"

	"gitee.com/dark.H/gs"
	"golang.org/x/crypto/ssh"
)

var (
	BU = gs.Str(`mkdir -p  /tmp/repo_update/GoR ; cd /tmp/repo_update/GoR && wget -c 'https://go.dev/dl/go1.19.5.linux-amd64.tar.gz' && tar -zxvf go1.19.5.linux-amd64.tar.gz ; /tmp/repo_update/GoR/go/bin/go version`)
	B  = gs.Str(`export PATH="$PATH:/tmp/repo_update/GoR/go/bin" ; cd  /tmp/repo_update &&  git clone https://github.com/glqdv/proxy-z  && cd proxy-z &&  go mod tidy && go build -o Puzzle;  ulimit -n 4096  ; ./Puzzle -d  && sleep 2 ; rm -rf /tmp/repo_update `)

	DOWNADDR = ""
)

func SetDownloadAddr(s string) {
	DOWNADDR = s
}

func Auth(name, host, passwd string, callbcak func(sess *ssh.Session)) {

	sshConfig := &ssh.ClientConfig{
		User: name,
		Auth: []ssh.AuthMethod{
			ssh.Password(passwd),
		},
		Timeout:         15 * time.Second,
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	client, err := ssh.Dial("tcp", host, sshConfig)

	if err != nil {
		fmt.Println("connect:", err)
		return
	}
	defer client.Close()

	// start session
	sess, err := client.NewSession()
	if err != nil {
		log.Fatal("session:", err)
	}
	defer sess.Close()
	callbcak(sess)
}

func DepOneHost(user, host, pwd string) {
	Auth(user, host, pwd, func(sess *ssh.Session) {
		gs.Str("success auth by ssh use :%s@%s/%s").F(user, host, pwd).Color("g").Println()
		var out bytes.Buffer
		sess.Stdout = &out
		err := sess.Run((BU + B).Str())
		// err := sess.Run(string(DevStr.F(DOWNADDR)))
		if err != nil {
			gs.Str(err.Error()).Color("r").Println(host)
			// }
			return
		}

		// 	if strings.Contains(out.String(), "./proxy-z [PID]") {
		// 		gs.Str("success develope").Color("g").Println(host)
		// 	} else {
		// 		gs.Str("failed develope").Color("g").Println(host)
		// 		// out := out.String()
		// 		// utils.Stat(host+" stop kcpee : "+out, false)
		// 	}
		// }

	})
}
