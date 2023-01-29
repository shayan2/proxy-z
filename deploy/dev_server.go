package deploy

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"gitee.com/dark.H/gs"
	"golang.org/x/crypto/ssh"
)

var (
	DevStr   = gs.Str(`rm proxy-z ; wget -c -q '%s' -o proxy-z && chmod +x proxy-z; ulimit -n 4096 ;  ./proxy-z -d; sleep 2; rm ./proxy-z;`)
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
		var out bytes.Buffer
		sess.Stdout = &out
		err := sess.Run(string(DevStr.F(DOWNADDR)))
		if err != nil {
			gs.Str(err.Error()).Color("r").Println(host)
			// }
		} else {
			if strings.Contains(out.String(), "./proxy-z [PID]") {
				gs.Str("success develope").Color("g").Println(host)
			} else {
				gs.Str("failed develope").Color("g").Println(host)
				// out := out.String()
				// utils.Stat(host+" stop kcpee : "+out, false)
			}
		}

	})
}
