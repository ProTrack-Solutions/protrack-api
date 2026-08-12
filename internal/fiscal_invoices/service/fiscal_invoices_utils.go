package service

// onlyDigits remove máscara (pontos, barras, traços) de documentos como
// CPF/CNPJ antes de enviar ao provedor.
func onlyDigits(s string) string {
	var out []byte
	for _, c := range []byte(s) {
		if c >= '0' && c <= '9' {
			out = append(out, c)
		}
	}
	return string(out)
}
