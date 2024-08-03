package util

import "strings"

const (
	textPlain      = "text/plain"
	applicationPdf = "application/pdf"
	imageJpeg      = "image/jpeg"
	imageJpg       = "image/jpg"
	imagePng       = "image/png"
)

func IsSupportedMimeType(mimeType string) bool {
	switch strings.ToLower(mimeType) {
	case textPlain, applicationPdf, imageJpeg, imageJpg, imagePng:
		return true
	}

	return false
}
