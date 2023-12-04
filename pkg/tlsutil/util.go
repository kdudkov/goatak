package tlsutil

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"go.uber.org/zap"
	"strings"

	"software.sslmate.com/src/go-pkcs12"
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

func MakeP12TrustStore(certs map[string]*x509.Certificate, passwd string) ([]byte, error) {
	var entries []pkcs12.TrustStoreEntry

	for k, v := range certs {
		entries = append(entries, pkcs12.TrustStoreEntry{Cert: v, FriendlyName: k})
	}

	return pkcs12.LegacyRC2.EncodeTrustStoreEntries(entries, passwd)
}

func CertToPem(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}

func KeyToPem(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
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
		sb.WriteByte(10)
	}
	return sb.String()
}

func MakeCertPool(certs ...*x509.Certificate) *x509.CertPool {
	cp := x509.NewCertPool()
	for _, c := range certs {
		if c != nil {
			cp.AddCert(c)
		}
	}

	return cp
}

func LogCert(logger *zap.SugaredLogger, name string, cert *x509.Certificate) {
	if cert == nil {
		logger.Errorf("no %s!!!", name)
		return
	}
	logger.Infof("%s sn: %x", name, cert.SerialNumber)
	logger.Infof("%s subject: %s", name, cert.Subject.String())
	logger.Infof("%s issuer: %s", name, cert.Issuer.String())
	logger.Infof("%s valid till %s", name, cert.NotAfter)
	if len(cert.DNSNames) > 0 {
		logger.Infof("%s dns_names: %s", name, strings.Join(cert.DNSNames, ","))
	}
	if len(cert.IPAddresses) > 0 {
		ip1 := make([]string, len(cert.IPAddresses))
		for i, ip := range cert.IPAddresses {
			ip1[i] = ip.String()
		}
		logger.Infof("%s ip_addresses: %s", name, strings.Join(ip1, ","))
	}
}

func LogCerts(logger *zap.SugaredLogger, certs ...*x509.Certificate) {
	for i, c := range certs {
		LogCert(logger, fmt.Sprintf("cert #%d", i), c)
	}
}
