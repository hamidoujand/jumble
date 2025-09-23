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
	"sync"

	"github.com/google/uuid"
)

var ErrKeyNotFound = errors.New("key not found")

const maxPEMSize = 1024 * 1024 //1MB

type Key struct {
	private *rsa.PrivateKey
	public  *rsa.PublicKey
	kid     string
}

type KeyStore struct {
	store map[string]Key
	mu    sync.RWMutex
}

func New() *KeyStore {
	return &KeyStore{
		store: make(map[string]Key),
	}
}

func (ks *KeyStore) LoadFromFileSystem(fsys fs.FS) (int, error) {
	// Example: ks.LoadRSAKeys(os.DirFS("/infra/keys/"))
	// Example: /infra/keys/54bb2165-71e1-41a6-af3e-7da4a0e1e2c1.pem

	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}

		//skip dirs
		if d.IsDir() {
			return nil
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

		//key id
		kid := uuid.NewString()

		key := Key{
			private: privateKey,
			public:  &privateKey.PublicKey,
			kid:     kid,
		}

		ks.mu.Lock()
		ks.store[kid] = key
		ks.mu.Unlock()
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
