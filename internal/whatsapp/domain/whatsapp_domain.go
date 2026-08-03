package domain

type EvolutionCreateResponse struct {
	Instance struct {
		InstanceName string `json:"instanceName"`
		Status       string `json:"status"`
	} `json:"instance"`
	QRCode struct {
		Base64 string `json:"base64"` // Este é o cara que você quer
		Code   string `json:"code"`   // Versão em texto do QR
	} `json:"qrcode"`
}

type EvolutionConnectionStateResponse struct {
	Instance struct {
		InstanceName string `json:"instanceName"`
		State        string `json:"state"`
	} `json:"instance"`
}

type CreateInstanceRequest struct {
	Integration string `json:"integration"`
	QrCode      bool   `json:"qr_code"`
}

type EvolutionConnectResponse struct {
	Code string `json:"code"`
}

type ConnectonStateResponse struct {
	InstanceName string `json:"instance_name"`
	State        string `json:"state"`
}
