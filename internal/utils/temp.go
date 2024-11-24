package utils

import (
	"fmt"
	"io"
	"os"
)

func WriteTempFile(rd io.Reader, ext string) (string, error) {
	audioFile, err := os.CreateTemp("", "*"+ext)
	if err != nil {
		return "", fmt.Errorf("ffmpeg create temp file: %w", err)
	}

	if _, err := io.Copy(audioFile, rd); err != nil {
		return "", fmt.Errorf("ffmpeg write temp file: %w", err)
	}

	if err := audioFile.Close(); err != nil {
		return "", fmt.Errorf("ffmpeg close temp file: %w", err)
	}

	return audioFile.Name(), nil
}
