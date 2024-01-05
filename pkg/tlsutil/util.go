package tlsutil

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"software.sslmate.com/src/go-pkcs12"
)

const cr = 10

func ParseBlock(b []byte, typ string) *pem.Block {
	bb := bytes.Buffer{}
	bb.WriteString(fmt.Sprintf("-----BEGIN %s-----\n", typ))
	bb.Write(b)
	bb.WriteString(fmt.Sprintf("\n-----END %s-----", typ))
	block, _ := pem.Decode(bb.Bytes())

	return block
}

func ParseCert(s string) (*x509.Certificate, error) {
	csrBlock := ParseBlock([]byte(s), "CERTIFICATE")

	return x509.ParseCertificate(csrBlock.Bytes)
}

func ParseCsr(b []byte) (*x509.CertificateRequest, error) {
	csrBlock := ParseBlock(b, "REQUEST")

	return x509.ParseCertificateRequest(csrBlock.Bytes)
}

func MakeP12TrustStore(certs map[string]*x509.Certificate, passwd string) ([]byte, error) {
	entries := make([]pkcs12.TrustStoreEntry, 0, len(certs))

	for k, v := range certs {
		entries = append(entries, pkcs12.TrustStoreEntry{Cert: v, FriendlyName: k})
	}

	return pkcs12.LegacyRC2.EncodeTrustStoreEntries(entries, passwd)
}

func CertToStr(cert *x509.Certificate, withHeader bool) string {
	s := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw, Headers: nil}))
	if withHeader {
		return s
	}

	ss := strings.Split(s, "\n")
	sb := strings.Builder{}

	for _, s1 := range ss {
		if s1 == "" || strings.HasPrefix(s1, "----") {
			continue
		}

		sb.WriteString(s1)
		sb.WriteByte(cr)
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

func DecodeAllCerts(bytes []byte) ([]*x509.Certificate, error) {
	return DecodeAllByType("CERTIFICATE", bytes)
}

func DecodeAllByType(typ string, bytes []byte) ([]*x509.Certificate, error) {
	var block *pem.Block

	certs := make([]*x509.Certificate, 0)

	for {
		block, bytes = pem.Decode(bytes)
		if block == nil {
			break
		}

		if block.Type == typ {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return certs, err
			}

			certs = append(certs, cert)
		}
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no %s in found", typ)
	}

	return certs, nil
}

func LogCerts(logger *zap.SugaredLogger, certs ...*x509.Certificate) {
	for i, c := range certs {
		LogCert(logger, fmt.Sprintf("cert #%d", i), c)
	}
}
