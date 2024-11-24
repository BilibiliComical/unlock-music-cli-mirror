package qmc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"golang.org/x/text/unicode/norm"
	"unlock-music.dev/mmkv"
)

var streamKeyVault mmkv.Vault

// TODO: move to factory
func readKeyFromMMKV(file string, logger *zap.Logger) ([]byte, error) {
	if file == "" {
		return nil, errors.New("file path is required while reading key from mmkv")
	}

	//goland:noinspection GoBoolExpressions
	if runtime.GOOS != "darwin" {
		return nil, errors.New("mmkv vault not supported on this platform")
	}

	if streamKeyVault == nil {
		mmkvDir, err := getRelativeMMKVDir(file)
		if err != nil {
			mmkvDir, err = getDefaultMMKVDir()
			if err != nil {
				return nil, fmt.Errorf("mmkv key valut not found: %w", err)
			}
		}

		mgr, err := mmkv.NewManager(mmkvDir)
		if err != nil {
			return nil, fmt.Errorf("init mmkv manager: %w", err)
		}

		streamKeyVault, err = mgr.OpenVault("MMKVStreamEncryptId")
		if err != nil {
			return nil, fmt.Errorf("open mmkv vault: %w", err)
		}

		logger.Debug("mmkv vault opened", zap.Strings("keys", streamKeyVault.Keys()))
	}

	_, partName := filepath.Split(file)
	partName = normalizeUnicode(partName)

	buf, err := streamKeyVault.GetBytes(file)
	if buf == nil {
		filePaths := streamKeyVault.Keys()
		fileNames := lo.Map(filePaths, func(filePath string, _ int) string {
			_, name := filepath.Split(filePath)
			return normalizeUnicode(name)
		})

		for _, key := range fileNames { // fallback: match filename only
			if key != partName {
				continue
			}
			idx := slices.Index(fileNames, key)
			buf, err = streamKeyVault.GetBytes(filePaths[idx])
			if err != nil {
				logger.Warn("read key from mmkv", zap.String("key", filePaths[idx]), zap.Error(err))
			}
		}

	}

	if len(buf) == 0 {
		return nil, errors.New("key not found in mmkv vault")
	}

	return deriveKey(buf)
}

func OpenMMKV(mmkvPath string, key string, logger *zap.Logger) error {
	filePath, fileName := filepath.Split(mmkvPath)
	mgr, err := mmkv.NewManager(filepath.Dir(filePath))
	if err != nil {
		return fmt.Errorf("init mmkv manager: %w", err)
	}

	// If `vaultKey` is empty, the key is ignored.
	streamKeyVault, err = mgr.OpenVaultCrypto(fileName, key)

	if err != nil {
		return fmt.Errorf("open mmkv vault: %w", err)
	}

	logger.Debug("mmkv vault opened", zap.Strings("keys", streamKeyVault.Keys()))
	return nil
}

// /
func readKeyFromMMKVCustom(mid string) ([]byte, error) {
	if streamKeyVault == nil {
		return nil, fmt.Errorf("mmkv vault not loaded")
	}

	// get ekey from mmkv vault
	eKey, err := streamKeyVault.GetBytes(mid)
	if err != nil {
		return nil, fmt.Errorf("get eKey error: %w", err)
	}
	return deriveKey(eKey)
}

// / getRelativeMMKVDir get mmkv dir relative to file (legacy QQMusic for macOS behaviour)
func getRelativeMMKVDir(file string) (string, error) {
	mmkvDir := filepath.Join(filepath.Dir(file), "../mmkv")
	if _, err := os.Stat(mmkvDir); err != nil {
		return "", fmt.Errorf("stat default mmkv dir: %w", err)
	}

	keyFile := filepath.Join(mmkvDir, "MMKVStreamEncryptId")
	if _, err := os.Stat(keyFile); err != nil {
		return "", fmt.Errorf("stat default mmkv file: %w", err)
	}

	return mmkvDir, nil
}

func getDefaultMMKVDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home dir: %w", err)
	}

	mmkvDir := filepath.Join(
		homeDir,
		"Library/Containers/com.tencent.QQMusicMac/Data",
		"Library/Application Support/QQMusicMac/mmkv",
	)
	if _, err := os.Stat(mmkvDir); err != nil {
		return "", fmt.Errorf("stat default mmkv dir: %w", err)
	}

	keyFile := filepath.Join(mmkvDir, "MMKVStreamEncryptId")
	if _, err := os.Stat(keyFile); err != nil {
		return "", fmt.Errorf("stat default mmkv file: %w", err)
	}

	return mmkvDir, nil
}

// normalizeUnicode normalizes unicode string to NFC.
// since macOS may change some characters in the file name.
// e.g. "ぜ"(e3 81 9c) -> "ぜ"(e3 81 9b e3 82 99)
func normalizeUnicode(str string) string {
	return norm.NFC.String(str)
}
