package main

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

func TestCert(t *testing.T) {
	t.SkipNow()

	go func() {
		_ = Server(":55555", "../../ca.pem", "../../ca.key")
	}()

	time.Sleep(time.Millisecond * 500)
	Client("127.0.0.1:55555", "../../ca.pem", "../../test_server.p12", "111111")
}

func Server(addr, certFile, keyFile string) (err error) {
	caCertPEM, err := os.ReadFile(certFile)
	if err != nil {
		panic(err)
	}

	roots := x509.NewCertPool()

	ok := roots.AppendCertsFromPEM(caCertPEM)
	if !ok {
		panic("failed to parse root certificate")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		panic(err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    roots,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	listener, err := tls.Listen("tcp4", addr, tlsCfg)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()

		log.Printf("connect")

		if err != nil {
			log.Printf("Unable to accept connections: %#v", err)

			continue
		}

		log.Printf("SSL connection")

		c1 := conn.(*tls.Conn)
		if err := c1.Handshake(); err != nil {
			log.Printf("Handshake error: %#v", err)

			_ = c1.Close()

			continue
		}

		log.Printf("%d", len(c1.ConnectionState().PeerCertificates))

		for _, c := range c1.ConnectionState().PeerCertificates {
			log.Println(c.Subject.CommonName)
		}

		_, _ = c1.Write([]byte("Ok"))
		_ = c1.Close()
	}
}

func Client(addr, caFile, p12file, passw string) {
	conn, err := tls.Dial("tcp", addr, getTLSConfig(caFile, p12file, passw))
	if err != nil {
		panic(err)
	}

	b := make([]byte, 10)

	n, err := conn.Read(b)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(b[:n]))
}

func getTLSConfig(caFile, p12File string, passw string) *tls.Config {
	p12Data, err := os.ReadFile(p12File)
	if err != nil {
		panic(err)
	}

	key, cert, _, err := pkcs12.DecodeChain(p12Data, passw)
	if err != nil {
		panic(err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key.(crypto.PrivateKey),
		Leaf:        cert,
	}

	ca, err := os.ReadFile(caFile)
	if err != nil {
		panic(err)
	}

	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(ca)

	return &tls.Config{Certificates: []tls.Certificate{tlsCert}, RootCAs: roots}
}
