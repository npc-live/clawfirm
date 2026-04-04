package remote

import (
	"encoding/base64"

	"github.com/skip2/go-qrcode"
)

// qrEncode renders content as a QR code PNG and returns a data URL.
func qrEncode(content string, size int) (string, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, size)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
