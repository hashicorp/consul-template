package openpgp

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/keybase/go-crypto/openpgp/armor"
	"github.com/keybase/go-crypto/openpgp/packet"
)

func ensureCrossSignatureInBundle(bundleStr string, t *testing.T) {
	el, err := ReadArmoredKeyRing(strings.NewReader(bundleStr))
	if err != nil {
		t.Fatal(err)
	}
	if len(el) != 1 {
		t.Fatal("Expected to load 1 entity")
	}
	subkey := el[0].Subkeys[0]
	if !subkey.Sig.FlagsValid || !subkey.Sig.FlagSign {
		t.Fatal("Subkey is not signing subkey")
	}
	if subkey.Sig.EmbeddedSignature == nil {
		t.Fatal("EmbeddedSignature is nil")
	}
}

func TestCreateAndCrossSignSubkey(t *testing.T) {
	uid := packet.NewUserId("PGP Test Uid", "", "hello@keybase.io")
	signingPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	currentTime := time.Now()
	signingPrivKey := packet.NewECDSAPrivateKey(currentTime, signingPriv)
	entity := &Entity{
		PrimaryKey: &signingPrivKey.PublicKey,
		PrivateKey: signingPrivKey,
		Identities: make(map[string]*Identity),
	}
	isPrimaryID := true
	entity.Identities[uid.Id] = &Identity{
		Name:   uid.Name,
		UserId: uid,
		SelfSignature: &packet.Signature{
			CreationTime: currentTime,
			SigType:      packet.SigTypePositiveCert,
			PubKeyAlgo:   packet.PubKeyAlgoECDSA,
			Hash:         crypto.SHA512,
			IsPrimaryId:  &isPrimaryID,
			FlagsValid:   true,
			FlagSign:     true,
			FlagCertify:  true,
			IssuerKeyId:  &entity.PrimaryKey.KeyId,
		},
	}

	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		panic(err)
	}
	privKey := packet.NewECDSAPrivateKey(currentTime, key)
	subkey := Subkey{
		PublicKey:  &privKey.PublicKey,
		PrivateKey: privKey,
		Sig: &packet.Signature{
			CreationTime: currentTime,
			SigType:      packet.SigTypeSubkeyBinding,
			PubKeyAlgo:   packet.PubKeyAlgoECDSA,
			Hash:         crypto.SHA512,
			FlagsValid:   true,
			FlagSign:     true,
			IssuerKeyId:  &entity.PrimaryKey.KeyId,
		},
	}
	subkey.PrivateKey.IsSubkey = true
	subkey.PublicKey.IsSubkey = true
	entity.Subkeys = append(entity.Subkeys, subkey)

	var buf bytes.Buffer
	err = entity.SerializePrivate(&buf, nil)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	writer, err := armor.Encode(&out, "PGP PRIVATE KEY BLOCK", nil)
	_, _ = writer.Write(buf.Bytes())
	writer.Close()

	ensureCrossSignatureInBundle(out.String(), t)

	buf.Reset()
	err = entity.Serialize(&buf)

	out.Reset()
	writer, err = armor.Encode(&out, "PGP PUBLIC KEY BLOCK", nil)
	_, _ = writer.Write(buf.Bytes())
	writer.Close()

	ensureCrossSignatureInBundle(out.String(), t)

}
