package vault

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/vault/physical"
)

const (
	// initialKeyTerm is the hard coded initial key term. This is
	// used only for values that are not encrypted with the keyring.
	initialKeyTerm = 1

	// termSize the number of bytes used for the key term.
	termSize = 4
)

// Versions of the AESGCM storage methodology
const (
	AESGCMVersion1 = 0x1
	AESGCMVersion2 = 0x2
)

// barrierInit is the JSON encoded value stored
type barrierInit struct {
	Version int    // Version is the current format version
	Key     []byte // Key is the primary encryption key
}

// AESGCMBarrier is a SecurityBarrier implementation that uses the AES
// cipher core and the Galois Counter Mode block mode. It defaults to
// the golang NONCE default value of 12 and a key size of 256
// bit. AES-GCM is high performance, and provides both confidentiality
// and integrity.
type AESGCMBarrier struct {
	backend physical.Backend

	l      sync.RWMutex
	sealed bool

	// keyring is used to maintain all of the encryption keys, including
	// the active key used for encryption, but also prior keys to allow
	// decryption of keys encrypted under previous terms.
	keyring *Keyring

	// cache is used to reduce the number of AEAD constructions we do
	cache     map[uint32]cipher.AEAD
	cacheLock sync.RWMutex

	// currentAESGCMVersionByte is prefixed to a message to allow for
	// future versioning of barrier implementations. It's var instead
	// of const to allow for testing
	currentAESGCMVersionByte byte
}

// NewAESGCMBarrier is used to construct a new barrier that uses
// the provided physical backend for storage.
func NewAESGCMBarrier(physical physical.Backend) (*AESGCMBarrier, error) {
	b := &AESGCMBarrier{
		backend: physical,
		sealed:  true,
		cache:   make(map[uint32]cipher.AEAD),
		currentAESGCMVersionByte: byte(AESGCMVersion2),
	}
	return b, nil
}

// Initialized checks if the barrier has been initialized
// and has a master key set.
func (b *AESGCMBarrier) Initialized() (bool, error) {
	// Read the keyring file
	out, err := b.backend.Get(keyringPath)
	if err != nil {
		return false, fmt.Errorf("failed to check for initialization: %v", err)
	}
	if out != nil {
		return true, nil
	}

	// Fallback, check for the old sentinel file
	out, err = b.backend.Get(barrierInitPath)
	if err != nil {
		return false, fmt.Errorf("failed to check for initialization: %v", err)
	}
	return out != nil, nil
}

// Initialize works only if the barrier has not been initialized
// and makes use of the given master key.
func (b *AESGCMBarrier) Initialize(key []byte) error {
	// Verify the key size
	min, max := b.KeyLength()
	if len(key) < min || len(key) > max {
		return fmt.Errorf("Key size must be %d or %d", min, max)
	}

	// Check if already initialized
	if alreadyInit, err := b.Initialized(); err != nil {
		return err
	} else if alreadyInit {
		return ErrBarrierAlreadyInit
	}

	// Generate encryption key
	encrypt, err := b.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %v", err)
	}

	// Create a new keyring, install the keys
	keyring := NewKeyring()
	keyring = keyring.SetMasterKey(key)
	keyring, err = keyring.AddKey(&Key{
		Term:    1,
		Version: 1,
		Value:   encrypt,
	})
	if err != nil {
		return fmt.Errorf("failed to create keyring: %v", err)
	}
	return b.persistKeyring(keyring)
}

// persistKeyring is used to write out the keyring using the
// master key to encrypt it.
func (b *AESGCMBarrier) persistKeyring(keyring *Keyring) error {
	// Create the keyring entry
	buf, err := keyring.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize keyring: %v", err)
	}
	defer memzero(buf)

	// Create the AES-GCM
	gcm, err := b.aeadFromKey(keyring.MasterKey())
	if err != nil {
		return err
	}

	// Encrypt the barrier init value
	value := b.encrypt(keyringPath, initialKeyTerm, gcm, buf)

	// Create the keyring physical entry
	pe := &physical.Entry{
		Key:   keyringPath,
		Value: value,
	}
	if err := b.backend.Put(pe); err != nil {
		return fmt.Errorf("failed to persist keyring: %v", err)
	}

	// Serialize the master key value
	key := &Key{
		Term:    1,
		Version: 1,
		Value:   keyring.MasterKey(),
	}
	buf, err = key.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize master key: %v", err)
	}
	defer memzero(buf)

	// Encrypt the master key
	activeKey := keyring.ActiveKey()
	aead, err := b.aeadFromKey(activeKey.Value)
	if err != nil {
		return err
	}
	value = b.encrypt(masterKeyPath, activeKey.Term, aead, buf)

	// Update the masterKeyPath for standby instances
	pe = &physical.Entry{
		Key:   masterKeyPath,
		Value: value,
	}
	if err := b.backend.Put(pe); err != nil {
		return fmt.Errorf("failed to persist master key: %v", err)
	}
	return nil
}

// GenerateKey is used to generate a new key
func (b *AESGCMBarrier) GenerateKey() ([]byte, error) {
	// Generate a 256bit key
	buf := make([]byte, 2*aes.BlockSize)
	_, err := rand.Read(buf)
	return buf, err
}

// KeyLength is used to sanity check a key
func (b *AESGCMBarrier) KeyLength() (int, int) {
	return aes.BlockSize, 2 * aes.BlockSize
}

// Sealed checks if the barrier has been unlocked yet. The Barrier
// is not expected to be able to perform any CRUD until it is unsealed.
func (b *AESGCMBarrier) Sealed() (bool, error) {
	b.l.RLock()
	defer b.l.RUnlock()
	return b.sealed, nil
}

// VerifyMaster is used to check if the given key matches the master key
func (b *AESGCMBarrier) VerifyMaster(key []byte) error {
	b.l.RLock()
	defer b.l.RUnlock()
	if b.sealed {
		return ErrBarrierSealed
	}
	if subtle.ConstantTimeCompare(key, b.keyring.MasterKey()) != 1 {
		return ErrBarrierInvalidKey
	}
	return nil
}

// ReloadKeyring is used to re-read the underlying keyring.
// This is used for HA deployments to ensure the latest keyring
// is present in the leader.
func (b *AESGCMBarrier) ReloadKeyring() error {
	b.l.Lock()
	defer b.l.Unlock()

	// Create the AES-GCM
	gcm, err := b.aeadFromKey(b.keyring.MasterKey())
	if err != nil {
		return err
	}

	// Read in the keyring
	out, err := b.backend.Get(keyringPath)
	if err != nil {
		return fmt.Errorf("failed to check for keyring: %v", err)
	}

	// Ensure that the keyring exists. This should never happen,
	// and indicates something really bad has happened.
	if out == nil {
		return fmt.Errorf("keyring unexpectedly missing")
	}

	// Decrypt the barrier init key
	plain, err := b.decrypt(keyringPath, gcm, out.Value)
	if err != nil {
		if strings.Contains(err.Error(), "message authentication failed") {
			return ErrBarrierInvalidKey
		}
		return err
	}
	defer memzero(plain)

	// Recover the keyring
	keyring, err := DeserializeKeyring(plain)
	if err != nil {
		return fmt.Errorf("keyring deserialization failed: %v", err)
	}

	// Setup the keyring and finish
	b.keyring = keyring
	return nil
}

// ReloadMasterKey is used to re-read the underlying masterkey.
// This is used for HA deployments to ensure the latest master key
// is available for keyring reloading.
func (b *AESGCMBarrier) ReloadMasterKey() error {
	// Read the masterKeyPath upgrade
	out, err := b.Get(masterKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read master key path: %v", err)
	}

	// The masterKeyPath could be missing (backwards incompatable),
	// we can ignore this and attempt to make progress with the current
	// master key.
	if out == nil {
		return nil
	}

	// Deserialize the master key
	key, err := DeserializeKey(out.Value)
	if err != nil {
		return fmt.Errorf("failed to deserialize key: %v", err)
	}

	b.l.Lock()
	defer b.l.Unlock()

	// Check if the master key is the same
	if bytes.Equal(b.keyring.MasterKey(), key.Value) {
		return nil
	}

	// Update the master key
	b.keyring = b.keyring.SetMasterKey(key.Value)
	return nil
}

// Unseal is used to provide the master key which permits the barrier
// to be unsealed. If the key is not correct, the barrier remains sealed.
func (b *AESGCMBarrier) Unseal(key []byte) error {
	b.l.Lock()
	defer b.l.Unlock()

	// Do nothing if already unsealed
	if !b.sealed {
		return nil
	}

	// Create the AES-GCM
	gcm, err := b.aeadFromKey(key)
	if err != nil {
		return err
	}

	// Read in the keyring
	out, err := b.backend.Get(keyringPath)
	if err != nil {
		return fmt.Errorf("failed to check for keyring: %v", err)
	}
	if out != nil {
		// Decrypt the barrier init key
		plain, err := b.decrypt(keyringPath, gcm, out.Value)
		if err != nil {
			if strings.Contains(err.Error(), "message authentication failed") {
				return ErrBarrierInvalidKey
			}
			return err
		}
		defer memzero(plain)

		// Recover the keyring
		keyring, err := DeserializeKeyring(plain)
		if err != nil {
			return fmt.Errorf("keyring deserialization failed: %v", err)
		}

		// Setup the keyring and finish
		b.keyring = keyring
		b.sealed = false
		return nil
	}

	// Read the barrier initialization key
	out, err = b.backend.Get(barrierInitPath)
	if err != nil {
		return fmt.Errorf("failed to check for initialization: %v", err)
	}
	if out == nil {
		return ErrBarrierNotInit
	}

	// Decrypt the barrier init key
	plain, err := b.decrypt(barrierInitPath, gcm, out.Value)
	if err != nil {
		if strings.Contains(err.Error(), "message authentication failed") {
			return ErrBarrierInvalidKey
		}
		return err
	}
	defer memzero(plain)

	// Unmarshal the barrier init
	var init barrierInit
	if err := json.Unmarshal(plain, &init); err != nil {
		return fmt.Errorf("failed to unmarshal barrier init file")
	}

	// Setup a new keyring, this is for backwards compatability
	keyring := NewKeyring()
	keyring = keyring.SetMasterKey(key)
	keyring, err = keyring.AddKey(&Key{
		Term:    1,
		Version: 1,
		Value:   init.Key,
	})
	if err != nil {
		return fmt.Errorf("failed to create keyring: %v", err)
	}
	if err := b.persistKeyring(keyring); err != nil {
		return err
	}

	// Delete the old barrier entry
	if err := b.backend.Delete(barrierInitPath); err != nil {
		return fmt.Errorf("failed to delete barrier init file: %v", err)
	}

	// Set the vault as unsealed
	b.keyring = keyring
	b.sealed = false
	return nil
}

// Seal is used to re-seal the barrier. This requires the barrier to
// be unsealed again to perform any further operations.
func (b *AESGCMBarrier) Seal() error {
	b.l.Lock()
	defer b.l.Unlock()

	// Remove the primary key, and seal the vault
	b.cache = make(map[uint32]cipher.AEAD)
	b.keyring = nil
	b.sealed = true
	return nil
}

// Rotate is used to create a new encryption key. All future writes
// should use the new key, while old values should still be decryptable.
func (b *AESGCMBarrier) Rotate() (uint32, error) {
	b.l.Lock()
	defer b.l.Unlock()
	if b.sealed {
		return 0, ErrBarrierSealed
	}

	// Generate a new key
	encrypt, err := b.GenerateKey()
	if err != nil {
		return 0, fmt.Errorf("failed to generate encryption key: %v", err)
	}

	// Get the next term
	term := b.keyring.ActiveTerm()
	newTerm := term + 1

	// Add a new encryption key
	newKeyring, err := b.keyring.AddKey(&Key{
		Term:    newTerm,
		Version: 1,
		Value:   encrypt,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to add new encryption key: %v", err)
	}

	// Persist the new keyring
	if err := b.persistKeyring(newKeyring); err != nil {
		return 0, err
	}

	// Swap the keyrings
	b.keyring = newKeyring
	return newTerm, nil
}

// CreateUpgrade creates an upgrade path key to the given term from the previous term
func (b *AESGCMBarrier) CreateUpgrade(term uint32) error {
	b.l.RLock()
	defer b.l.RUnlock()
	if b.sealed {
		return ErrBarrierSealed
	}

	// Get the key for this term
	termKey := b.keyring.TermKey(term)
	buf, err := termKey.Serialize()
	if err != nil {
		return err
	}

	// Get the AEAD for the previous term
	prevTerm := term - 1
	primary, err := b.aeadForTerm(prevTerm)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s%d", keyringUpgradePrefix, prevTerm)
	value := b.encrypt(key, prevTerm, primary, buf)
	// Create upgrade key
	pe := &physical.Entry{
		Key:   key,
		Value: value,
	}
	return b.backend.Put(pe)
}

// DestroyUpgrade destroys the upgrade path key to the given term
func (b *AESGCMBarrier) DestroyUpgrade(term uint32) error {
	path := fmt.Sprintf("%s%d", keyringUpgradePrefix, term-1)
	return b.Delete(path)
}

// CheckUpgrade looks for an upgrade to the current term and installs it
func (b *AESGCMBarrier) CheckUpgrade() (bool, uint32, error) {
	b.l.RLock()
	defer b.l.RUnlock()
	if b.sealed {
		return false, 0, ErrBarrierSealed
	}

	// Get the current term
	activeTerm := b.keyring.ActiveTerm()

	// Check for an upgrade key
	upgrade := fmt.Sprintf("%s%d", keyringUpgradePrefix, activeTerm)
	entry, err := b.Get(upgrade)
	if err != nil {
		return false, 0, err
	}

	// Nothing to do if no upgrade
	if entry == nil {
		return false, 0, nil
	}

	// Deserialize the key
	key, err := DeserializeKey(entry.Value)
	if err != nil {
		return false, 0, err
	}

	// Upgrade from read lock to write lock
	b.l.RUnlock()
	defer b.l.RLock()
	b.l.Lock()
	defer b.l.Unlock()

	// Update the keyring
	newKeyring, err := b.keyring.AddKey(key)
	if err != nil {
		return false, 0, fmt.Errorf("failed to add new encryption key: %v", err)
	}
	b.keyring = newKeyring

	// Done!
	return true, key.Term, nil
}

// ActiveKeyInfo is used to inform details about the active key
func (b *AESGCMBarrier) ActiveKeyInfo() (*KeyInfo, error) {
	b.l.RLock()
	defer b.l.RUnlock()
	if b.sealed {
		return nil, ErrBarrierSealed
	}

	// Determine the key install time
	term := b.keyring.ActiveTerm()
	key := b.keyring.TermKey(term)

	// Return the key info
	info := &KeyInfo{
		Term:        int(term),
		InstallTime: key.InstallTime,
	}
	return info, nil
}

// Rekey is used to change the master key used to protect the keyring
func (b *AESGCMBarrier) Rekey(key []byte) error {
	b.l.Lock()
	defer b.l.Unlock()
	if b.sealed {
		return ErrBarrierSealed
	}

	// Verify the key size
	min, max := b.KeyLength()
	if len(key) < min || len(key) > max {
		return fmt.Errorf("Key size must be %d or %d", min, max)
	}

	// Add a new encryption key
	newKeyring := b.keyring.SetMasterKey(key)

	// Persist the new keyring
	if err := b.persistKeyring(newKeyring); err != nil {
		return err
	}

	// Swap the keyrings
	b.keyring = newKeyring
	return nil
}

// Put is used to insert or update an entry
func (b *AESGCMBarrier) Put(entry *Entry) error {
	defer metrics.MeasureSince([]string{"barrier", "put"}, time.Now())
	b.l.RLock()
	defer b.l.RUnlock()
	if b.sealed {
		return ErrBarrierSealed
	}

	term := b.keyring.ActiveTerm()
	primary, err := b.aeadForTerm(term)
	if err != nil {
		return err
	}

	pe := &physical.Entry{
		Key:   entry.Key,
		Value: b.encrypt(entry.Key, term, primary, entry.Value),
	}
	return b.backend.Put(pe)
}

// Get is used to fetch an entry
func (b *AESGCMBarrier) Get(key string) (*Entry, error) {
	defer metrics.MeasureSince([]string{"barrier", "get"}, time.Now())
	b.l.RLock()
	defer b.l.RUnlock()
	if b.sealed {
		return nil, ErrBarrierSealed
	}

	// Read the key from the backend
	pe, err := b.backend.Get(key)
	if err != nil {
		return nil, err
	} else if pe == nil {
		return nil, nil
	}

	// Decrypt the ciphertext
	plain, err := b.decryptKeyring(key, pe.Value)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %v", err)
	}

	// Wrap in a logical entry
	entry := &Entry{
		Key:   key,
		Value: plain,
	}
	return entry, nil
}

// Delete is used to permanently delete an entry
func (b *AESGCMBarrier) Delete(key string) error {
	defer metrics.MeasureSince([]string{"barrier", "delete"}, time.Now())
	b.l.RLock()
	defer b.l.RUnlock()
	if b.sealed {
		return ErrBarrierSealed
	}

	return b.backend.Delete(key)
}

// List is used ot list all the keys under a given
// prefix, up to the next prefix.
func (b *AESGCMBarrier) List(prefix string) ([]string, error) {
	defer metrics.MeasureSince([]string{"barrier", "list"}, time.Now())
	b.l.RLock()
	defer b.l.RUnlock()
	if b.sealed {
		return nil, ErrBarrierSealed
	}

	return b.backend.List(prefix)
}

// aeadForTerm returns the AES-GCM AEAD for the given term
func (b *AESGCMBarrier) aeadForTerm(term uint32) (cipher.AEAD, error) {
	// Check for the keyring
	keyring := b.keyring
	if keyring == nil {
		return nil, nil
	}

	// Check the cache for the aead
	b.cacheLock.RLock()
	aead, ok := b.cache[term]
	b.cacheLock.RUnlock()
	if ok {
		return aead, nil
	}

	// Read the underlying key
	key := keyring.TermKey(term)
	if key == nil {
		return nil, nil
	}

	// Create a new aead
	aead, err := b.aeadFromKey(key.Value)
	if err != nil {
		return nil, err
	}

	// Update the cache
	b.cacheLock.Lock()
	b.cache[term] = aead
	b.cacheLock.Unlock()
	return aead, nil
}

// aeadFromKey returns an AES-GCM AEAD using the given key.
func (b *AESGCMBarrier) aeadFromKey(key []byte) (cipher.AEAD, error) {
	// Create the AES cipher
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	// Create the GCM mode AEAD
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCM mode")
	}
	return gcm, nil
}

// encrypt is used to encrypt a value
func (b *AESGCMBarrier) encrypt(path string, term uint32, gcm cipher.AEAD, plain []byte) []byte {
	// Allocate the output buffer with room for tern, version byte,
	// nonce, GCM tag and the plaintext
	capacity := termSize + 1 + gcm.NonceSize() + gcm.Overhead() + len(plain)
	size := termSize + 1 + gcm.NonceSize()
	out := make([]byte, size, capacity)

	// Set the key term
	binary.BigEndian.PutUint32(out[:4], term)

	// Set the version byte
	out[4] = b.currentAESGCMVersionByte

	// Generate a random nonce
	nonce := out[5 : 5+gcm.NonceSize()]
	rand.Read(nonce)

	// Seal the output
	switch b.currentAESGCMVersionByte {
	case AESGCMVersion1:
		out = gcm.Seal(out, nonce, plain, nil)
	case AESGCMVersion2:
		out = gcm.Seal(out, nonce, plain, []byte(path))
	default:
		panic("Unknown AESGCM version")
	}

	return out
}

// decrypt is used to decrypt a value
func (b *AESGCMBarrier) decrypt(path string, gcm cipher.AEAD, cipher []byte) ([]byte, error) {
	// Verify the term is always just one
	term := binary.BigEndian.Uint32(cipher[:4])
	if term != initialKeyTerm {
		return nil, fmt.Errorf("term mis-match")
	}

	// Capture the parts
	nonce := cipher[5 : 5+gcm.NonceSize()]
	raw := cipher[5+gcm.NonceSize():]
	out := make([]byte, 0, len(raw)-gcm.NonceSize())

	// Verify the cipher byte and attempt to open
	switch cipher[4] {
	case AESGCMVersion1:
		return gcm.Open(out, nonce, raw, nil)
	case AESGCMVersion2:
		return gcm.Open(out, nonce, raw, []byte(path))
	default:
		return nil, fmt.Errorf("version bytes mis-match")
	}
}

// decryptKeyring is used to decrypt a value using the keyring
func (b *AESGCMBarrier) decryptKeyring(path string, cipher []byte) ([]byte, error) {
	// Verify the term
	term := binary.BigEndian.Uint32(cipher[:4])

	// Get the GCM by term
	// It is expensive to do this first but it is not a
	// normal case that this won't match
	gcm, err := b.aeadForTerm(term)
	if err != nil {
		return nil, err
	}
	if gcm == nil {
		return nil, fmt.Errorf("no decryption key available for term %d", term)
	}

	nonce := cipher[5 : 5+gcm.NonceSize()]
	raw := cipher[5+gcm.NonceSize():]
	out := make([]byte, 0, len(raw)-gcm.NonceSize())

	// Attempt to open
	switch cipher[4] {
	case AESGCMVersion1:
		return gcm.Open(out, nonce, raw, nil)
	case AESGCMVersion2:
		return gcm.Open(out, nonce, raw, []byte(path))
	default:
		return nil, fmt.Errorf("version bytes mis-match")
	}
}
