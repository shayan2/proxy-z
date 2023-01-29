package deploy

import (
	"bytes"
	"log"
	"net"
	"os"
	"time"

	"gitee.com/dark.H/gn"
	"gitee.com/dark.H/gs"
	"github.com/go-git/go-git/v5"
	"golang.org/x/crypto/ssh"
)

var (
	BU = gs.Str(`mkdir -p  /tmp/repo_update/GoR ; cd /tmp/repo_update/GoR && wget -c 'https://go.dev/dl/go1.19.5.linux-amd64.tar.gz' && tar -zxf go1.19.5.linux-amd64.tar.gz ; /tmp/repo_update/GoR/go/bin/go version;`)
	B  = gs.Str(`ps aux | grep './Puzzle' | grep -v grep| awk '{print $2}' | xargs kill -9 ;export PATH="$PATH:/tmp/repo_update/GoR/go/bin" ; cd  /tmp/repo_update &&  git clone https://github.com/glqdv/proxy-z  && cd proxy-z &&  go mod tidy && go build -o Puzzle;  ulimit -n 4096  ;./Puzzle -h; ./Puzzle -d  && sleep 2 ; rm -rf /tmp/repo_update `)

	DOWNADDR = ""
)

func SetDownloadAddr(s string) {
	DOWNADDR = s
}

func Auth(name, host, passwd string, callbcak func(c *ssh.Client, s *ssh.Session)) {

	sshConfig := &ssh.ClientConfig{
		User: name,
		Auth: []ssh.AuthMethod{
			ssh.Password(passwd),
		},
		Timeout:         15 * time.Second,
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
	}
	keyFile := gs.Str("~").ExpandUser().PathJoin(".ssh", "id_rsa")
	if keyFile.IsExists() {
		if keybuf := keyFile.MustAsFile(); keybuf != "" {
			signal, err := ssh.ParsePrivateKey(keybuf.Bytes())
			if err == nil {
				sshConfig.Auth = append(sshConfig.Auth,
					ssh.PublicKeys(signal),
				)
			}
		}

	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	client, err := ssh.Dial("tcp", host, sshConfig)

	if err != nil {
		gs.Str(err.Error()).Println("Err")
		return
	}
	defer client.Close()

	// start session
	sess, err := client.NewSession()
	if err != nil {
		log.Fatal("session:", err)
	}
	defer sess.Close()
	callbcak(client, sess)
}

func DepOneHost(user, host, pwd string) {
	Auth(user, host, pwd, func(client *ssh.Client, sess *ssh.Session) {
		gs.Str("success auth by ssh use :%s@%s/%s").F(user, host, pwd).Color("g").Println()
		var out bytes.Buffer
		sess.Stdout = &out
		err := sess.Run(BU.Str())
		// err := sess.Run(string(DevStr.F(DOWNADDR)))
		if err != nil {
			gs.Str(err.Error()).Color("r").Println(host)
			// }
			return
		} else {
			gs.Str(out.String()).Trim().Color("g").Println(host)
		}
		sess.Close()
		var out2 = bytes.NewBuffer([]byte{})
		sess2, err := client.NewSession()
		if err != nil {
			gs.Str(err.Error()).Color("r").Println("Err")
			return
		}
		sess2.Stdout = out2

		err = sess2.Run(B.Str())
		if err != nil {
			gs.Str(err.Error()).Color("r").Println(host)
			return
		} else {
			// gs.Str(out2.String()).Color("g").Println(host)
		}

	})
}

func DepBySSH(sshStr string) {
	user := "root"
	host := ""
	pwd := ""
	if gs.Str(sshStr).In("@") {
		gs.Str(sshStr).Split("@").Every(func(no int, i gs.Str) {
			if no == 0 {
				user = i.Str()
			} else {
				if i.In("/") {
					i.Split("/").Every(func(no int, i gs.Str) {
						if no == 0 {
							host = i.Str()
						} else {
							pwd = i.Str()
						}
					})

				} else {
					host = i.Str()
				}
			}
		})
	} else {
		i := gs.Str(sshStr)
		if i.In("/") {
			i.Split("/").Every(func(no int, i gs.Str) {
				if no == 0 {
					host = i.Str()
				} else {
					pwd = i.Str()
				}
			})
		} else {
			host = i.Str()
		}
	}
	if !gs.Str(host).In(":") {
		host += ":22"
	}
	if user != "" && host != "" {
		DepOneHost(user, host, pwd)
	} else {
		gs.Str("user:%s host:%s pwd:%s").F(user, host, pwd).Println()
	}
}

func GetConfig(user string, pwd string) (err error) {
	REPO_TMP := gs.TMP.PathJoin("repo")
	defer REPO_TMP.Rm()
	REPO_PATH := REPO_TMP.PathJoin("pz")
	if REPO_PATH.IsExists() {
		REPO_PATH.Rm()
	}
	repoUrl := "https://gitee.com/dark.H/"
	_, err = git.PlainClone(REPO_PATH.Str(), false, &git.CloneOptions{
		URL:      repoUrl,
		Progress: os.Stdout,
	})
	return err
}

type onevps struct {
	Host     string
	Pwd      string
	Location string
	Tag      string
}

func (o onevps) Println() {
	w := gs.Str("tag:%s ").F(o.Tag).Color("b", "B") + gs.Str("host: %s ").F(o.Host).Color("g") + gs.Str("loc: "+o.Location).Color("m", "B")
	w.Println()
}

func (o onevps) Build() {
	DepOneHost("root", o.Host+":22", o.Pwd)
}

func SearchFromVultr(tag, api string) (vpss gs.List[onevps]) {
	nt := gs.Str("https://api.vultr.com/v1/server/list").AsRequest()
	nt = nt.SetMethod(gs.Str("POST")).SetHead("API-Key", gs.Str(api))
	r := gn.AsReq(nt).HTTPS()
	r.Build()

	if res := r.Go(); res.Str != "" {
		data := res.Json()
		data.Every(func(k string, v any) {
			vals := data[k].(map[string]interface{})
			tag := vals["tag"].(string)
			passwd := vals["default_password"].(string)
			host := vals["main_ip"].(string)
			location := vals["location"].(string)
			if gs.Str(tag + host + location).In(tag) {
				vpss = vpss.Add(onevps{
					Host:     host,
					Tag:      tag,
					Pwd:      passwd,
					Location: location,
				})
			}
		})

	}

	return
}
