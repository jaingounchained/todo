package util

import "strings"

const (
	TextPlain      = "text/plain"
	ApplicationPDF = "application/pdf"
	ImageJPEG      = "image/jpeg"
	ImageJPG       = "image/jpg"
	ImagePNG       = "image/png"
)

func IsSupportedMimeType(mimeType string) bool {
	switch strings.ToLower(mimeType) {
	case TextPlain, ApplicationPDF, ImageJPEG, ImageJPG, ImagePNG:
		return true
	}

	return false
}
