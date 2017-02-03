package keysutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/hkdf"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/helper/errutil"
	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/helper/kdf"
	"github.com/hashicorp/vault/logical"
)

// Careful with iota; don't put anything before it in this const block because
// we need the default of zero to be the old-style KDF
const (
	Kdf_hmac_sha256_counter = iota // built-in helper
	Kdf_hkdf_sha256                // golang.org/x/crypto/hkdf
)

// Or this one...we need the default of zero to be the original AES256-GCM96
const (
	KeyType_AES256_GCM96 = iota
	KeyType_ECDSA_P256
)

const ErrTooOld = "ciphertext or signature version is disallowed by policy (too old)"

type ecdsaSignature struct {
	R, S *big.Int
}

type KeyType int

func (kt KeyType) EncryptionSupported() bool {
	switch kt {
	case KeyType_AES256_GCM96:
		return true
	}
	return false
}

func (kt KeyType) DecryptionSupported() bool {
	switch kt {
	case KeyType_AES256_GCM96:
		return true
	}
	return false
}

func (kt KeyType) SigningSupported() bool {
	switch kt {
	case KeyType_ECDSA_P256:
		return true
	}
	return false
}

func (kt KeyType) DerivationSupported() bool {
	switch kt {
	case KeyType_AES256_GCM96:
		return true
	}
	return false
}

func (kt KeyType) String() string {
	switch kt {
	case KeyType_AES256_GCM96:
		return "aes256-gcm96"
	case KeyType_ECDSA_P256:
		return "ecdsa-p256"
	}

	return "[unknown]"
}

// KeyEntry stores the key and metadata
type KeyEntry struct {
	AESKey             []byte   `json:"key"`
	HMACKey            []byte   `json:"hmac_key"`
	CreationTime       int64    `json:"creation_time"`
	EC_X               *big.Int `json:"ec_x"`
	EC_Y               *big.Int `json:"ec_y"`
	EC_D               *big.Int `json:"ec_d"`
	FormattedPublicKey string   `json:"public_key"`
}

// keyEntryMap is used to allow JSON marshal/unmarshal
type keyEntryMap map[int]KeyEntry

// MarshalJSON implements JSON marshaling
func (kem keyEntryMap) MarshalJSON() ([]byte, error) {
	intermediate := map[string]KeyEntry{}
	for k, v := range kem {
		intermediate[strconv.Itoa(k)] = v
	}
	return json.Marshal(&intermediate)
}

// MarshalJSON implements JSON unmarshaling
func (kem keyEntryMap) UnmarshalJSON(data []byte) error {
	intermediate := map[string]KeyEntry{}
	if err := jsonutil.DecodeJSON(data, &intermediate); err != nil {
		return err
	}
	for k, v := range intermediate {
		keyval, err := strconv.Atoi(k)
		if err != nil {
			return err
		}
		kem[keyval] = v
	}

	return nil
}

// Policy is the struct used to store metadata
type Policy struct {
	Name string      `json:"name"`
	Key  []byte      `json:"key,omitempty"` //DEPRECATED
	Keys keyEntryMap `json:"keys"`

	// Derived keys MUST provide a context and the master underlying key is
	// never used. If convergent encryption is true, the context will be used
	// as the nonce as well.
	Derived              bool `json:"derived"`
	KDF                  int  `json:"kdf"`
	ConvergentEncryption bool `json:"convergent_encryption"`

	// Whether the key is exportable
	Exportable bool `json:"exportable"`

	// The minimum version of the key allowed to be used
	// for decryption
	MinDecryptionVersion int `json:"min_decryption_version"`

	// The latest key version in this policy
	LatestVersion int `json:"latest_version"`

	// The latest key version in the archive. We never delete these, so this is
	// a max.
	ArchiveVersion int `json:"archive_version"`

	// Whether the key is allowed to be deleted
	DeletionAllowed bool `json:"deletion_allowed"`

	// The version of the convergent nonce to use
	ConvergentVersion int `json:"convergent_version"`

	// The type of key
	Type KeyType `json:"type"`
}

// ArchivedKeys stores old keys. This is used to keep the key loading time sane
// when there are huge numbers of rotations.
type archivedKeys struct {
	Keys []KeyEntry `json:"keys"`
}

func (p *Policy) LoadArchive(storage logical.Storage) (*archivedKeys, error) {
	archive := &archivedKeys{}

	raw, err := storage.Get("archive/" + p.Name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		archive.Keys = make([]KeyEntry, 0)
		return archive, nil
	}

	if err := jsonutil.DecodeJSON(raw.Value, archive); err != nil {
		return nil, err
	}

	return archive, nil
}

func (p *Policy) storeArchive(archive *archivedKeys, storage logical.Storage) error {
	// Encode the policy
	buf, err := json.Marshal(archive)
	if err != nil {
		return err
	}

	// Write the policy into storage
	err = storage.Put(&logical.StorageEntry{
		Key:   "archive/" + p.Name,
		Value: buf,
	})
	if err != nil {
		return err
	}

	return nil
}

// handleArchiving manages the movement of keys to and from the policy archive.
// This should *ONLY* be called from Persist() since it assumes that the policy
// will be persisted afterwards.
func (p *Policy) handleArchiving(storage logical.Storage) error {
	// We need to move keys that are no longer accessible to archivedKeys, and keys
	// that now need to be accessible back here.
	//
	// For safety, because there isn't really a good reason to, we never delete
	// keys from the archive even when we move them back.

	// Check if we have the latest minimum version in the current set of keys
	_, keysContainsMinimum := p.Keys[p.MinDecryptionVersion]

	// Sanity checks
	switch {
	case p.MinDecryptionVersion < 1:
		return fmt.Errorf("minimum decryption version of %d is less than 1", p.MinDecryptionVersion)
	case p.LatestVersion < 1:
		return fmt.Errorf("latest version of %d is less than 1", p.LatestVersion)
	case !keysContainsMinimum && p.ArchiveVersion != p.LatestVersion:
		return fmt.Errorf("need to move keys from archive but archive version not up-to-date")
	case p.ArchiveVersion > p.LatestVersion:
		return fmt.Errorf("archive version of %d is greater than the latest version %d",
			p.ArchiveVersion, p.LatestVersion)
	case p.MinDecryptionVersion > p.LatestVersion:
		return fmt.Errorf("minimum decryption version of %d is greater than the latest version %d",
			p.MinDecryptionVersion, p.LatestVersion)
	}

	archive, err := p.LoadArchive(storage)
	if err != nil {
		return err
	}

	if !keysContainsMinimum {
		// Need to move keys *from* archive

		for i := p.MinDecryptionVersion; i <= p.LatestVersion; i++ {
			p.Keys[i] = archive.Keys[i]
		}

		return nil
	}

	// Need to move keys *to* archive

	// We need a size that is equivalent to the latest version (number of keys)
	// but adding one since slice numbering starts at 0 and we're indexing by
	// key version
	if len(archive.Keys) < p.LatestVersion+1 {
		// Increase the size of the archive slice
		newKeys := make([]KeyEntry, p.LatestVersion+1)
		copy(newKeys, archive.Keys)
		archive.Keys = newKeys
	}

	// We are storing all keys in the archive, so we ensure that it is up to
	// date up to p.LatestVersion
	for i := p.ArchiveVersion + 1; i <= p.LatestVersion; i++ {
		archive.Keys[i] = p.Keys[i]
		p.ArchiveVersion = i
	}

	err = p.storeArchive(archive, storage)
	if err != nil {
		return err
	}

	// Perform deletion afterwards so that if there is an error saving we
	// haven't messed with the current policy
	for i := p.LatestVersion - len(p.Keys) + 1; i < p.MinDecryptionVersion; i++ {
		delete(p.Keys, i)
	}

	return nil
}

func (p *Policy) Persist(storage logical.Storage) error {
	err := p.handleArchiving(storage)
	if err != nil {
		return err
	}

	// Encode the policy
	buf, err := p.Serialize()
	if err != nil {
		return err
	}

	// Write the policy into storage
	err = storage.Put(&logical.StorageEntry{
		Key:   "policy/" + p.Name,
		Value: buf,
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Policy) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Policy) NeedsUpgrade() bool {
	// Ensure we've moved from Key -> Keys
	if p.Key != nil && len(p.Key) > 0 {
		return true
	}

	// With archiving, past assumptions about the length of the keys map are no
	// longer valid
	if p.LatestVersion == 0 && len(p.Keys) != 0 {
		return true
	}

	// We disallow setting the version to 0, since they start at 1 since moving
	// to rotate-able keys, so update if it's set to 0
	if p.MinDecryptionVersion == 0 {
		return true
	}

	// On first load after an upgrade, copy keys to the archive
	if p.ArchiveVersion == 0 {
		return true
	}

	// Need to write the version
	if p.ConvergentEncryption && p.ConvergentVersion == 0 {
		return true
	}

	if p.Keys[p.LatestVersion].HMACKey == nil || len(p.Keys[p.LatestVersion].HMACKey) == 0 {
		return true
	}

	return false
}

func (p *Policy) Upgrade(storage logical.Storage) error {
	persistNeeded := false
	// Ensure we've moved from Key -> Keys
	if p.Key != nil && len(p.Key) > 0 {
		p.MigrateKeyToKeysMap()
		persistNeeded = true
	}

	// With archiving, past assumptions about the length of the keys map are no
	// longer valid
	if p.LatestVersion == 0 && len(p.Keys) != 0 {
		p.LatestVersion = len(p.Keys)
		persistNeeded = true
	}

	// We disallow setting the version to 0, since they start at 1 since moving
	// to rotate-able keys, so update if it's set to 0
	if p.MinDecryptionVersion == 0 {
		p.MinDecryptionVersion = 1
		persistNeeded = true
	}

	// On first load after an upgrade, copy keys to the archive
	if p.ArchiveVersion == 0 {
		persistNeeded = true
	}

	if p.ConvergentEncryption && p.ConvergentVersion == 0 {
		p.ConvergentVersion = 1
		persistNeeded = true
	}

	if p.Keys[p.LatestVersion].HMACKey == nil || len(p.Keys[p.LatestVersion].HMACKey) == 0 {
		entry := p.Keys[p.LatestVersion]
		hmacKey, err := uuid.GenerateRandomBytes(32)
		if err != nil {
			return err
		}
		entry.HMACKey = hmacKey
		p.Keys[p.LatestVersion] = entry
		persistNeeded = true
	}

	if persistNeeded {
		err := p.Persist(storage)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeriveKey is used to derive the encryption key that should be used depending
// on the policy. If derivation is disabled the raw key is used and no context
// is required, otherwise the KDF mode is used with the context to derive the
// proper key.
func (p *Policy) DeriveKey(context []byte, ver int) ([]byte, error) {
	if !p.Type.DerivationSupported() {
		return nil, errutil.UserError{Err: fmt.Sprintf("derivation not supported for key type %v", p.Type)}
	}

	if p.Keys == nil || p.LatestVersion == 0 {
		return nil, errutil.InternalError{Err: "unable to access the key; no key versions found"}
	}

	if ver <= 0 || ver > p.LatestVersion {
		return nil, errutil.UserError{Err: "invalid key version"}
	}

	// Fast-path non-derived keys
	if !p.Derived {
		return p.Keys[ver].AESKey, nil
	}

	// Ensure a context is provided
	if len(context) == 0 {
		return nil, errutil.UserError{Err: "missing 'context' for key deriviation. The key was created using a derived key, which means additional, per-request information must be included in order to encrypt or decrypt information"}
	}

	switch p.KDF {
	case Kdf_hmac_sha256_counter:
		prf := kdf.HMACSHA256PRF
		prfLen := kdf.HMACSHA256PRFLen
		return kdf.CounterMode(prf, prfLen, p.Keys[ver].AESKey, context, 256)
	case Kdf_hkdf_sha256:
		reader := hkdf.New(sha256.New, p.Keys[ver].AESKey, nil, context)
		derBytes := bytes.NewBuffer(nil)
		derBytes.Grow(32)
		limReader := &io.LimitedReader{
			R: reader,
			N: 32,
		}
		n, err := derBytes.ReadFrom(limReader)
		if err != nil {
			return nil, errutil.InternalError{Err: fmt.Sprintf("error reading returned derived bytes: %v", err)}
		}
		if n != 32 {
			return nil, errutil.InternalError{Err: fmt.Sprintf("unable to read enough derived bytes, needed 32, got %d", n)}
		}
		return derBytes.Bytes(), nil
	default:
		return nil, errutil.InternalError{Err: "unsupported key derivation mode"}
	}
}

func (p *Policy) Encrypt(context, nonce []byte, value string) (string, error) {
	if !p.Type.EncryptionSupported() {
		return "", errutil.UserError{Err: fmt.Sprintf("message encryption not supported for key type %v", p.Type)}
	}

	// Guard against a potentially invalid key type
	switch p.Type {
	case KeyType_AES256_GCM96:
	default:
		return "", errutil.InternalError{Err: fmt.Sprintf("unsupported key type %v", p.Type)}
	}

	// Decode the plaintext value
	plaintext, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", errutil.UserError{Err: "failed to base64-decode plaintext"}
	}

	// Derive the key that should be used
	key, err := p.DeriveKey(context, p.LatestVersion)
	if err != nil {
		return "", err
	}

	// Guard against a potentially invalid key type
	switch p.Type {
	case KeyType_AES256_GCM96:
	default:
		return "", errutil.InternalError{Err: fmt.Sprintf("unsupported key type %v", p.Type)}
	}

	// Setup the cipher
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return "", errutil.InternalError{Err: err.Error()}
	}

	// Setup the GCM AEAD
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return "", errutil.InternalError{Err: err.Error()}
	}

	if p.ConvergentEncryption {
		switch p.ConvergentVersion {
		case 1:
			if len(nonce) != gcm.NonceSize() {
				return "", errutil.UserError{Err: fmt.Sprintf("base64-decoded nonce must be %d bytes long when using convergent encryption with this key", gcm.NonceSize())}
			}
		default:
			nonceHmac := hmac.New(sha256.New, context)
			nonceHmac.Write(plaintext)
			nonceSum := nonceHmac.Sum(nil)
			nonce = nonceSum[:gcm.NonceSize()]
		}
	} else {
		// Compute random nonce
		nonce, err = uuid.GenerateRandomBytes(gcm.NonceSize())
		if err != nil {
			return "", errutil.InternalError{Err: err.Error()}
		}
	}

	// Encrypt and tag with GCM
	out := gcm.Seal(nil, nonce, plaintext, nil)

	// Place the encrypted data after the nonce
	full := out
	if !p.ConvergentEncryption || p.ConvergentVersion > 1 {
		full = append(nonce, out...)
	}

	// Convert to base64
	encoded := base64.StdEncoding.EncodeToString(full)

	// Prepend some information
	encoded = "vault:v" + strconv.Itoa(p.LatestVersion) + ":" + encoded

	return encoded, nil
}

func (p *Policy) Decrypt(context, nonce []byte, value string) (string, error) {
	if !p.Type.DecryptionSupported() {
		return "", errutil.UserError{Err: fmt.Sprintf("message decryption not supported for key type %v", p.Type)}
	}

	// Verify the prefix
	if !strings.HasPrefix(value, "vault:v") {
		return "", errutil.UserError{Err: "invalid ciphertext: no prefix"}
	}

	if p.ConvergentEncryption && p.ConvergentVersion == 1 && (nonce == nil || len(nonce) == 0) {
		return "", errutil.UserError{Err: "invalid convergent nonce supplied"}
	}

	splitVerCiphertext := strings.SplitN(strings.TrimPrefix(value, "vault:v"), ":", 2)
	if len(splitVerCiphertext) != 2 {
		return "", errutil.UserError{Err: "invalid ciphertext: wrong number of fields"}
	}

	ver, err := strconv.Atoi(splitVerCiphertext[0])
	if err != nil {
		return "", errutil.UserError{Err: "invalid ciphertext: version number could not be decoded"}
	}

	if ver == 0 {
		// Compatibility mode with initial implementation, where keys start at
		// zero
		ver = 1
	}

	if ver > p.LatestVersion {
		return "", errutil.UserError{Err: "invalid ciphertext: version is too new"}
	}

	if p.MinDecryptionVersion > 0 && ver < p.MinDecryptionVersion {
		return "", errutil.UserError{Err: ErrTooOld}
	}

	// Derive the key that should be used
	key, err := p.DeriveKey(context, ver)
	if err != nil {
		return "", err
	}

	// Guard against a potentially invalid key type
	switch p.Type {
	case KeyType_AES256_GCM96:
	default:
		return "", errutil.InternalError{Err: fmt.Sprintf("unsupported key type %v", p.Type)}
	}

	// Decode the base64
	decoded, err := base64.StdEncoding.DecodeString(splitVerCiphertext[1])
	if err != nil {
		return "", errutil.UserError{Err: "invalid ciphertext: could not decode base64"}
	}

	// Setup the cipher
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return "", errutil.InternalError{Err: err.Error()}
	}

	// Setup the GCM AEAD
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return "", errutil.InternalError{Err: err.Error()}
	}

	// Extract the nonce and ciphertext
	var ciphertext []byte
	if p.ConvergentEncryption && p.ConvergentVersion < 2 {
		ciphertext = decoded
	} else {
		nonce = decoded[:gcm.NonceSize()]
		ciphertext = decoded[gcm.NonceSize():]
	}

	// Verify and Decrypt
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errutil.UserError{Err: "invalid ciphertext: unable to decrypt"}
	}

	return base64.StdEncoding.EncodeToString(plain), nil
}

func (p *Policy) HMACKey(version int) ([]byte, error) {
	if version < p.MinDecryptionVersion {
		return nil, fmt.Errorf("key version disallowed by policy (minimum is %d)", p.MinDecryptionVersion)
	}

	if version > p.LatestVersion {
		return nil, fmt.Errorf("key version does not exist; latest key version is %d", p.LatestVersion)
	}

	if p.Keys[version].HMACKey == nil {
		return nil, fmt.Errorf("no HMAC key exists for that key version")
	}

	return p.Keys[version].HMACKey, nil
}

func (p *Policy) Sign(hashedInput []byte) (string, error) {
	if !p.Type.SigningSupported() {
		return "", fmt.Errorf("message signing not supported for key type %v", p.Type)
	}

	var sig []byte
	switch p.Type {
	case KeyType_ECDSA_P256:
		keyParams := p.Keys[p.LatestVersion]
		key := &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     keyParams.EC_X,
				Y:     keyParams.EC_Y,
			},
			D: keyParams.EC_D,
		}
		r, s, err := ecdsa.Sign(rand.Reader, key, hashedInput)
		if err != nil {
			return "", err
		}
		marshaledSig, err := asn1.Marshal(ecdsaSignature{
			R: r,
			S: s,
		})
		if err != nil {
			return "", err
		}
		sig = marshaledSig

	default:
		return "", fmt.Errorf("unsupported key type %v", p.Type)
	}

	// Convert to base64
	encoded := base64.StdEncoding.EncodeToString(sig)

	// Prepend some information
	encoded = "vault:v" + strconv.Itoa(p.LatestVersion) + ":" + encoded

	return encoded, nil
}

func (p *Policy) VerifySignature(hashedInput []byte, sig string) (bool, error) {
	if !p.Type.SigningSupported() {
		return false, errutil.UserError{Err: fmt.Sprintf("message verification not supported for key type %v", p.Type)}
	}

	// Verify the prefix
	if !strings.HasPrefix(sig, "vault:v") {
		return false, errutil.UserError{Err: "invalid signature: no prefix"}
	}

	splitVerSig := strings.SplitN(strings.TrimPrefix(sig, "vault:v"), ":", 2)
	if len(splitVerSig) != 2 {
		return false, errutil.UserError{Err: "invalid signature: wrong number of fields"}
	}

	ver, err := strconv.Atoi(splitVerSig[0])
	if err != nil {
		return false, errutil.UserError{Err: "invalid signature: version number could not be decoded"}
	}

	if ver > p.LatestVersion {
		return false, errutil.UserError{Err: "invalid signature: version is too new"}
	}

	if p.MinDecryptionVersion > 0 && ver < p.MinDecryptionVersion {
		return false, errutil.UserError{Err: ErrTooOld}
	}

	switch p.Type {
	case KeyType_ECDSA_P256:
		asn1Sig, err := base64.StdEncoding.DecodeString(splitVerSig[1])
		if err != nil {
			return false, errutil.UserError{Err: "invalid base64 signature value"}
		}

		var ecdsaSig ecdsaSignature
		rest, err := asn1.Unmarshal(asn1Sig, &ecdsaSig)
		if err != nil {
			return false, errutil.UserError{Err: "supplied signature is invalid"}
		}
		if rest != nil && len(rest) != 0 {
			return false, errutil.UserError{Err: "supplied signature contains extra data"}
		}

		keyParams := p.Keys[ver]
		key := &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     keyParams.EC_X,
			Y:     keyParams.EC_Y,
		}

		return ecdsa.Verify(key, hashedInput, ecdsaSig.R, ecdsaSig.S), nil
	default:
		return false, errutil.InternalError{Err: fmt.Sprintf("unsupported key type %v", p.Type)}
	}

	return false, errutil.InternalError{Err: "no valid key type found"}
}

func (p *Policy) Rotate(storage logical.Storage) error {
	if p.Keys == nil {
		// This is an initial key rotation when generating a new policy. We
		// don't need to call migrate here because if we've called getPolicy to
		// get the policy in the first place it will have been run.
		p.Keys = keyEntryMap{}
	}

	p.LatestVersion += 1
	entry := KeyEntry{
		CreationTime: time.Now().Unix(),
	}

	hmacKey, err := uuid.GenerateRandomBytes(32)
	if err != nil {
		return err
	}
	entry.HMACKey = hmacKey

	switch p.Type {
	case KeyType_AES256_GCM96:
		// Generate a 256bit key
		newKey, err := uuid.GenerateRandomBytes(32)
		if err != nil {
			return err
		}
		entry.AESKey = newKey

	case KeyType_ECDSA_P256:
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}
		entry.EC_D = privKey.D
		entry.EC_X = privKey.X
		entry.EC_Y = privKey.Y
		derBytes, err := x509.MarshalPKIXPublicKey(privKey.Public())
		if err != nil {
			return fmt.Errorf("error marshaling public key: %s", err)
		}
		pemBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: derBytes,
		}
		pemBytes := pem.EncodeToMemory(pemBlock)
		if pemBytes == nil || len(pemBytes) == 0 {
			return fmt.Errorf("error PEM-encoding public key")
		}
		entry.FormattedPublicKey = string(pemBytes)
	}

	p.Keys[p.LatestVersion] = entry

	// This ensures that with new key creations min decryption version is set
	// to 1 rather than the int default of 0, since keys start at 1 (either
	// fresh or after migration to the key map)
	if p.MinDecryptionVersion == 0 {
		p.MinDecryptionVersion = 1
	}

	return p.Persist(storage)
}

func (p *Policy) MigrateKeyToKeysMap() {
	p.Keys = keyEntryMap{
		1: KeyEntry{
			AESKey:       p.Key,
			CreationTime: time.Now().Unix(),
		},
	}
	p.Key = nil
}
