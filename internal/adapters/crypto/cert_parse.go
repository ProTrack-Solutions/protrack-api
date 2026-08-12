package crypto

import (
	"fmt"
	"time"

	"golang.org/x/crypto/pkcs12"
)

// CertificateInfo carrega os metadados extraídos de um certificado digital
// A1 (.pfx/.p12) — usados para exibir/validar a validade sem precisar
// descriptografar e reabrir o arquivo depois.
type CertificateInfo struct {
	SubjectCN string
	NotAfter  time.Time
}

// ParseCertificate decodifica um arquivo PKCS#12 (.pfx/.p12) protegido por
// senha e extrai o CN do titular e a data de expiração (NotAfter) do
// certificado. Usado para popular cert_subject_cn/expires_at em
// company_certificates, permitindo bloquear emissão com certificado vencido.
//
// golang.org/x/crypto/pkcs12 está "congelado" (não recebe novas features) e
// só decodifica os algoritmos legados usados pela maioria dos certificados
// A1 emitidos por ACs do ICP-Brasil (RC2/3DES). Certificados com PBES2/AES
// (mais raros hoje em dia) podem falhar aqui — nesse caso o upload não é
// bloqueado, mas cert_subject_cn/expires_at ficam vazios (ver caller).
func ParseCertificate(pfxData []byte, password string) (CertificateInfo, error) {
	_, cert, err := pkcs12.Decode(pfxData, password)
	if err != nil {
		return CertificateInfo{}, fmt.Errorf("crypto: falha ao decodificar certificado PKCS#12: %w", err)
	}

	return CertificateInfo{
		SubjectCN: cert.Subject.CommonName,
		NotAfter:  cert.NotAfter,
	}, nil
}
