package install

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func readOptional(path string, limit int64) ([]byte, bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > limit {
		return nil, false, fmt.Errorf("%s exceeds %d bytes", path, limit)
	}
	payload, err := os.ReadFile(path)
	return payload, err == nil, err
}

func atomicWrite(path string, payload []byte, mode os.FileMode) error {
	if err := recoverSwap(path); err != nil {
		return err
	}
	if existing, found, err := readOptional(path, int64(len(payload))+1); err == nil && found && string(existing) == string(payload) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".aav-tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	swap := path + ".aav-swap"
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, swap); err != nil {
			return err
		}
	}
	if err := os.Rename(tempPath, path); err != nil {
		if _, swapErr := os.Stat(swap); swapErr == nil {
			_ = os.Rename(swap, path)
		}
		return err
	}
	_ = os.Remove(swap)
	return nil
}

func recoverSwap(path string) error {
	swap := path + ".aav-swap"
	_, targetErr := os.Stat(path)
	_, swapErr := os.Stat(swap)
	if errors.Is(targetErr, os.ErrNotExist) && swapErr == nil {
		return os.Rename(swap, path)
	}
	if targetErr == nil && swapErr == nil {
		return os.Remove(swap)
	}
	if targetErr != nil && !errors.Is(targetErr, os.ErrNotExist) {
		return targetErr
	}
	if swapErr != nil && !errors.Is(swapErr, os.ErrNotExist) {
		return swapErr
	}
	return nil
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func digest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
