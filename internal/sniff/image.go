package sniff

// ref: https://mimesniff.spec.whatwg.org
var imageMIMEs = map[string]Sniffer{
	"image/jpeg": prefixSniffer{0xFF, 0xD8, 0xFF},
	"image/png":  prefixSniffer{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'},
	"image/bmp":  prefixSniffer("BM"),
	"image/webp": prefixSniffer("RIFF"),
	"image/gif":  prefixSniffer("GIF8"),
}

// ImageMIME sniffs the well-known image types, and returns its MIME.
func ImageMIME(header []byte) (string, bool) {
	for ext, sniffer := range imageMIMEs {
		if sniffer.Sniff(header) {
			return ext, true
		}
	}
	return "", false
}

// ImageExtension is equivalent to ImageMIME, but returns file extension
func ImageExtension(header []byte) (string, bool) {
	ext, ok := ImageMIME(header)
	if !ok {
		return "", false
	}
	// todo: use mime.ExtensionsByType
	return "." + ext[6:], true // "image/" is 6 bytes
}
