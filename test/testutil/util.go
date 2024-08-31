package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

const (
	KB = 1024
	MB = KB * KB
)

func RandString(length int) string {
	b := make([]byte, length/2+1)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	s := hex.EncodeToString(b)
	return s[:length]
}

func RandBytes(length int) []byte {
	return []byte(RandString(length))
}

func TempFilename(prefix string) string {
	return fmt.Sprintf("%s/%s%s", os.TempDir(), prefix, uuid.NewString())
}

func WaitForFile(filename string, maxDur time.Duration) error {
	deadline := time.Now().Add(maxDur)
	for {
		_, err := os.Stat(filename)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func Wait1SForFile(filename string) error {
	return WaitForFile(filename, 1*time.Second)
}

func Wait1SForFileThenOpen(filename string) (*os.File, error) {
	err := Wait1SForFile(filename)
	if err != nil {
		return nil, err
	}

	return os.Open(filename)
}
