package events

import "github.com/google/uuid"

// RoutingKeyPurchaseOrderEmail e-mail do pedido de compra ao fornecedor
const RoutingKeyPurchaseOrderEmail = "email.purchase_order"

// PurchaseOrderEmail mensagem pronta (HTML e PDF já gerados pela API): o
// consumer só entrega. Data dos anexos vai em base64 no JSON.
type PurchaseOrderEmail struct {
	CompanyID   uuid.UUID         `json:"company_id"`
	OrderID     uuid.UUID         `json:"order_id"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc"`
	ReplyTo     string            `json:"reply_to"`
	FromName    string            `json:"from_name"`
	Subject     string            `json:"subject"`
	HTML        string            `json:"html"`
	Attachments []EmailAttachment `json:"attachments"`
}

type EmailAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}
