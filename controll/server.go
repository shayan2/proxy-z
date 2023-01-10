package controll

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"gitee.com/dark.H/ProxyZ/asset"
	"gitee.com/dark.H/ProxyZ/connections/baseconnection"

	"gitee.com/dark.H/gs"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
)

var (
	CERT    = "Resources/pem/cert.pem"
	KEYPEM  = "Resources/pem/key.pem"
	Tunnels = gs.List[*baseconnection.ProxyTunnel]{}
)

func setupHandler(www string) http.Handler {
	mux := http.NewServeMux()
	if len(www) > 0 {
		mux.HandleFunc("/z-files", func(w http.ResponseWriter, r *http.Request) {
			fs := gs.List[any]{}
			gs.Str(www).Ls().Every(func(no int, i gs.Str) {
				isDir := i.IsDir()
				name := i.Basename()
				size := i.FileSize()
				fs = fs.Add(gs.Dict[any]{
					"name":  name,
					"isDir": isDir,
					"size":  size,
				})
			})
			fmt.Fprintf(w, string(gs.Dict[any]{
				"status": "ok",
				"msg":    fs,
			}.Json()))

		})
		mux.Handle("/z-files-d/", http.StripPrefix("/z-files-d/", http.FileServer(http.Dir(www))))
		mux.HandleFunc("/z-files-u", uploadFileFunc(www))
	}
	mux.HandleFunc("/z-info", func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadAll(r.Body)
		if err != io.EOF && err != nil {
			w.WriteHeader(400)
			fmt.Fprintln(w, err.Error())
			return
		}
		if len(buf) == 0 {
			fmt.Fprintf(w, string(gs.Dict[any]{
				"status": "ok",
				"msg":    "alive",
			}.Json()))
			return
		}
		if d := gs.S(buf).Json(); len(d) > 0 {

		}
	})

	mux.HandleFunc("/proxy-get", func(w http.ResponseWriter, r *http.Request) {

	})
	return mux
}

func HTTP3Server(serverAddr, wwwDir string) {

	quicConf := &quic.Config{}
	handler := setupHandler(wwwDir)

	cerPEM, err := asset.Asset(CERT)
	if err != nil {
		log.Fatal(err)
	}
	keyPEM, err := asset.Asset(KEYPEM)
	if err != nil {
		log.Fatal(err)
	}

	// Load the certificate and private key
	cert, err := tls.X509KeyPair(cerPEM, keyPEM)
	if err != nil {
		panic(err)
	}

	// Create a TLS configuration
	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(cerPEM)
	tlsconfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            certpool,
		ClientCAs:          certpool,
		InsecureSkipVerify: false,
	}

	server := http3.Server{
		Handler:    handler,
		Addr:       serverAddr,
		QuicConfig: quicConf,
		TLSConfig:  tlsconfig,
	}
	// Bind to a port and listen for incoming connections

	err = server.ListenAndServe()
	if err != nil {
		log.Println("listen server tls err:", err)
	}
}
