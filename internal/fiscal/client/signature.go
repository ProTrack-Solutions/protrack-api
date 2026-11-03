package client

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// DefaultSignatureTolerance: webhook com carimbo mais velho (ou do futuro) que isso é recusado.
const DefaultSignatureTolerance = 5 * time.Minute

// VerifyWebhookSignature confere o header X-ProTrack-Signature
// ("t=<unix>,v1=<hex HMAC-SHA256 de "<t>.<corpo>">") com o segredo whsec_ do endpoint.
func VerifyWebhookSignature(secret, header string, body []byte, now time.Time, tolerance time.Duration) bool {
	if secret == "" || header == "" {
		return false
	}
	var timestamp, received string
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			received = value
		}
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || received == "" {
		return false
	}
	age := now.Sub(time.Unix(ts, 0))
	if age < 0 {
		age = -age
	}
	if age > tolerance {
		return false
	}
	want, err := hex.DecodeString(received)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "."))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), want)
}
