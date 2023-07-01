package tlsutil

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"software.sslmate.com/src/go-pkcs12"
	"strings"
)

func ParseCert(s string) (*x509.Certificate, error) {
	bb := bytes.Buffer{}
	bb.WriteString("-----BEGIN CERTIFICATE-----\n")
	bb.WriteString(s)
	bb.WriteString("\n-----END CERTIFICATE-----")
	csrBlock, _ := pem.Decode(bb.Bytes())

	return x509.ParseCertificate(csrBlock.Bytes)
}

func ParseCsr(b []byte) (*x509.CertificateRequest, error) {
	bb := bytes.Buffer{}
	bb.WriteString("-----BEGIN CERTIFICATE REQUEST-----\n")
	bb.Write(b)
	bb.WriteString("\n-----END CERTIFICATE REQUEST-----")
	csrBlock, _ := pem.Decode(bb.Bytes())

	return x509.ParseCertificateRequest(csrBlock.Bytes)
}

func MakeP12(certs map[string]*x509.Certificate, passwd string) ([]byte, error) {
	var entries []pkcs12.TrustStoreEntry

	for k, v := range certs {
		entries = append(entries, pkcs12.TrustStoreEntry{Cert: v, FriendlyName: k})
	}

	return pkcs12.EncodeTrustStoreEntries(rand.Reader, entries, passwd)
}

func CertToPem(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}

func CertToStr(cert *x509.Certificate, header bool) string {
	s := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	if header {
		return s
	}

	ss := strings.Split(s, "\n")
	sb := strings.Builder{}
	for _, s1 := range ss {
		if s1 == "" || strings.HasPrefix(s1, "----") {
			continue
		}
		sb.WriteString(s1)
		sb.WriteByte(0x10)
	}
	return sb.String()
}
