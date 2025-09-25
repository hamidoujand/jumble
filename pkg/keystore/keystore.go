// Package keystore provides an in-memory storage for auth keys.
package keystore

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
)

var ErrKeyNotFound = errors.New("key not found")

const maxPEMSize = 1024 * 1024 //1MB

type Key struct {
	private *rsa.PrivateKey
	public  *rsa.PublicKey
}

type KeyStore struct {
	store     map[string]Key
	mu        sync.RWMutex
	activeKey string
}

func New() *KeyStore {
	return &KeyStore{
		store: make(map[string]Key),
	}
}

func (ks *KeyStore) LoadFromFileSystem(fsys fs.FS) (int, error) {
	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}

		//skip dirs
		if d.IsDir() {
			// Kubernetes secrets are mounted using a symlink structure which is
			// causing the walker function to see the same file twice.

			// /etc/rsa-keys/
			// ├── ..data -> ..2025_09_25_12_36_31.3055298050/
			// ├── ..2025_09_25_12_36_31.3055298050/
			// │   └── private.pem
			// └── private.pem -> ..data/private.pem   ACTUAL FILE we want

			// Skip Kubernetes internal directories
			if strings.HasPrefix(d.Name(), "..") {
				return fs.SkipDir // use when you want to skip the entire directory and not process its content as well.
			}
			return nil //use when you want to skip current file/dir but want to process files inside of that dir
			//just skipping the dir itself.
		}

		//skip files without .pem ext
		if filepath.Ext(path) != ".pem" {
			return nil
		}

		file, err := fsys.Open(path)
		if err != nil {
			return fmt.Errorf("openning file %s: %w", path, err)
		}

		defer file.Close()

		pemBytes, err := io.ReadAll(io.LimitReader(file, maxPEMSize))
		if err != nil {
			return fmt.Errorf("readAll: %w", err)
		}

		//create a private key from it

		pemBlock, _ := pem.Decode(pemBytes)
		if pemBlock == nil || pemBlock.Type != "PRIVATE KEY" {
			return fmt.Errorf("invalid pem bytes: key must be either in PKCS1 or PKCS8")
		}

		var parsedKey any
		parsedKey, err = x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
		if err != nil {
			//try pkcs8
			parsedKey, err = x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
			if err != nil {
				return fmt.Errorf("failed to parse private key as PKCS1 or PKCS8: %w", err)
			}
		}

		privateKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return errors.New("key is not a valid rsa private key")
		}

		//filename will be uuid.pem and will be used as the ID of that key.

		id := strings.TrimSuffix(filepath.Base(path), ".pem")

		key := Key{
			private: privateKey,
			public:  &privateKey.PublicKey,
		}

		ks.mu.Lock()
		defer ks.mu.Unlock()
		ks.store[id] = key
		return nil
	}

	if err := fs.WalkDir(fsys, ".", walker); err != nil {
		return 0, fmt.Errorf("walkDir: %w", err)
	}

	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return len(ks.store), nil
}

func (ks *KeyStore) PrivateKey(kid string) (*rsa.PrivateKey, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	k, ok := ks.store[kid]
	if !ok {
		return nil, ErrKeyNotFound
	}

	return k.private, nil
}

func (ks *KeyStore) PublicKey(kid string) (*rsa.PublicKey, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	k, ok := ks.store[kid]
	if !ok {
		return nil, ErrKeyNotFound
	}

	return k.public, nil
}

func (ks *KeyStore) SetActiveKey(key string) error {
	//check to see if the key is a valid key
	found := func() bool {
		ks.mu.RLock()
		defer ks.mu.RUnlock()
		_, ok := ks.store[key]
		return ok
	}()

	if !found {
		return fmt.Errorf("key[%s] not found in keystore", key)
	}

	//set it as the active key
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.activeKey = key
	return nil
}

func (ks *KeyStore) GetActiveKid() string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.activeKey
}
