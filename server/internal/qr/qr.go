// Package qr renders QR codes for TOTP provisioning URIs.
package qr

import (
	"encoding/base64"
	"fmt"

	"rsc.io/qr"
)

// DataURI encodes content as a PNG QR code and returns it as an image data URI.
// scale is the number of image pixels per QR module; values below one are
// clamped to one. The callers treat rendering as best-effort: manual secret
// entry remains available when encoding fails.
func DataURI(content string, scale int) (string, error) {
	if scale < 1 {
		scale = 1
	}

	code, err := qr.Encode(content, qr.M)
	if err != nil {
		return "", fmt.Errorf("encode qr: %w", err)
	}
	code.Scale = scale

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(code.PNG()), nil
}
