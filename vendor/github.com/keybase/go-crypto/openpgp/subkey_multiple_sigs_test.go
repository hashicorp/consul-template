package openpgp

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/keybase/go-crypto/openpgp/armor"
	"github.com/keybase/go-crypto/openpgp/packet"
)

// Keep key in parts that we concatenate in different ways to get different
// result.

const keyAndIdsPackets = `c652045b6f142613082a8648ce3d030107020304a54bdb6b6f7c145f584c3cf33ba9894b16708b3707310df27625546d2b10310a4ac4b23c516c8ceb4f38aa96a21e02c8717480c831095a2a287317d8e03e3c22cd264d69636861c582205067702054657374203c6e6f742d7a617075406b6579626173652e696f3ec2640413130a001605025b6f14260910433b1545547b718c021b03021901000034cf00ff68b1c1c4abded838df365ce8bec751ff2de8b58cf474b39cef708ac9790db30d0100a8702072316854172daff68dc0167c9e66b2159bca936087a30a8f7054ef20a8`
const subkeyPacket = `ce6d045b6f1426010300b97dfbd29467121a7f621d2eff9c78ad8f90017f2074c94f1999ae9956b7e2169af0bcf9d3021421b88e9166d11a2e3153d8ccd9c85d59525af654e4e4c63166273dc365a04ac40d8abce2397aa3058433ce76afba7c7362d130cd395e4e1f0d0011010001`

// Signature with Flags=Sign, valid cross-signature, and expiring 4 years from
// now.
const signSigPacket = `c2c01b0418130a001905025b6f14260910433b1545547b718c021b0205090784ce00007473200419010a000605025b6f14260000863803006e73fb2763cb717761b4c8cda9306037c58715454f92d4c39004cf7adffdfc25ea79b85d65840a13bb8eb1d8db455a2f72207195aeed8f6a37e6dfcd35ef5985de539f3bf17358841ad7581fc2cb5844dbb0b2d206e6ae6e99447fcb7f9306517a6100fe396ab483e28ccc6f55e9129c9c209e92eca03560c4baf3156e454347a8c4d27f00fe216749fa6aadb5018c00699b71040f3404572c257772b71751de234f361edaf7`

// Signature with Flags=Encrypt, no cross-signature (not needed), and never
// expiring. When both are present in bundle, this one will win and key will
// become encryption key.
const encryptSigPacket = `c2610418130a001305025b6f14260910433b1545547b718c021b0c000013aa0100b42663de14cbf358a84d96c997450fc7911426b4eff49aa36bcc532b352618e800ff5953423ce1f82b35ed8b421c3d9a3b3f4f02d0aa05bfa8f99c5b8711b1f290b4`

// Signature with Flags=Encrypt, cross-certified. This binding makes key expire
// after 4 years, so if it's combined with binding with no-expiration, it should
// lose.
const encryptXSignPacket = `c2c01b0418130a001905025b6f14260910433b1545547b718c021b0c05090784ce00007473200419010a000605025b6f14260000863803006e73fb2763cb717761b4c8cda9306037c58715454f92d4c39004cf7adffdfc25ea79b85d65840a13bb8eb1d8db455a2f72207195aeed8f6a37e6dfcd35ef5985de539f3bf17358841ad7581fc2cb5844dbb0b2d206e6ae6e99447fcb7f930651f00200ff785d955a6c2a10bded5f7033ba6fa9b38b58dcbf17039bf593fd060a4735e3ef00fd12376c99eb14665d8620c90debc5993be492dbb163e9bc364d52b2b8acc11c68`

// Signature with Flags=Sign, not cross-certified.
const signSigNoXSignPacket = `c2610418130a001305025b6f14260910433b1545547b718c021b020000ca2f00ff7c5d366c584ca03ea27cd0dad841f8adda24fc7efa212550ec773effc418136300fe32160c17b36a3a13be3ca6058d35dc7da89bfbb857753e6db45994183e58ed6d`

const armoredMessage = `
-----BEGIN PGP MESSAGE-----

xA0DAAoBRFStQ5saN6IBy+F0AOIAAAAA5GhlbGxvIGNyb3NzIHNpZ27jZWQgd29y
bGQAwnwEAAEKABAFAluX8qYJEERUrUObGjeiAADPOgMAaAUjgKY0r+vsO4bxXr5d
F99ostfQWReex/tkPGqvQRrwEVMKgymQ8zerQdu+30nl+UibIXu9LSvxPbQkPcWN
xC/ywM5zfa/WOMD1zrOjoCpUktnyMZN8H4P4bF8Az4aj
=7UNF
-----END PGP MESSAGE-----

`

func ConfigWithDate(dateStr string) *packet.Config {
	return &packet.Config{
		Time: func() time.Time {
			time1, _ := time.Parse("2006-01-02", dateStr)
			return time1
		},
	}
}

func readKeyFromHex(hexStr string) (el EntityList, err error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return el, err
	}
	return ReadKeyRing(bytes.NewBuffer(data))
}

func TestKeyEncryptSigWins(t *testing.T) {
	config := ConfigWithDate("2018-09-10")

	// `flagEncryptSig`` is "fresher" than `flagSignSig`` so the key will be
	// considered encryption key and not signing key.
	keyStr := keyAndIdsPackets + subkeyPacket + signSigPacket + encryptSigPacket
	kring, err := readKeyFromHex(keyStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(kring[0].Subkeys) == 0 {
		t.Fatalf("Subkey was not loaded")
	}
	if kring[0].Subkeys[0].Sig.FlagSign {
		t.Fatalf("Got unexpected FlagSign subkey=1")
	}
	sig, err := armor.Decode(strings.NewReader(armoredMessage))
	if err != nil {
		t.Fatal(err)
	}
	md, err := ReadMessage(sig.Body, kring, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	if !md.IsSigned {
		t.Fatalf("Expecting md.IsSigned")
	}
	// When we don't find the signing public key, SignedByKeyId will still have
	// the correct id, but SignedBy will be nil.
	if md.SignedByKeyId != kring[0].Subkeys[0].PublicKey.KeyId {
		t.Fatalf("Expected message should appear to be signed by subkey=0 (key id does not match)")
	}
	if md.SignedBy != nil {
		t.Fatalf("Expecting md.SignedBy to be nil, got: %v", md.SignedBy)
	}
}

func TestKeySignSigWins(t *testing.T) {
	config := ConfigWithDate("2018-09-10")

	// Do not add last signature this time, should be able to verify the
	// message.
	keyStr := keyAndIdsPackets + subkeyPacket + signSigPacket
	kring, err := readKeyFromHex(keyStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(kring[0].Subkeys) == 0 {
		t.Fatalf("Subkey was not loaded")
	}
	if !kring[0].Subkeys[0].Sig.FlagSign {
		t.Fatalf("Got unexpected FlagSign subkey=0")
	}
	sig, err := armor.Decode(strings.NewReader(armoredMessage))
	if err != nil {
		t.Fatal(err)
	}
	md, err := ReadMessage(sig.Body, kring, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	if !md.IsSigned {
		t.Fatalf("Expecting md.IsSigned")
	}
	// When we don't find the signing public key, SignedByKeyId will still have
	// the correct id, but SignedBy will be nil.
	if md.SignedByKeyId != kring[0].Subkeys[0].PublicKey.KeyId {
		t.Fatalf("Expected message should appear to be signed by subkey=0 (key id does not match)")
	}
	if md.SignedBy == nil || md.SignedBy.PublicKey.KeyId != md.SignedByKeyId {
		t.Fatalf("Got unexpected md.SignedBy: %v", md.SignedBy)
	}
}

func TestKeyWithUnrelatedCrossSign(t *testing.T) {
	config := ConfigWithDate("2018-09-10")

	// Have a bundle that has flags=sign binding with no cross-certification,
	// but a flags=encrypt binding *with* cross-certification. Key flags and
	// certification status should not be "merged" from both signatures.
	keyStr := keyAndIdsPackets + subkeyPacket + signSigNoXSignPacket + encryptXSignPacket
	kring, err := readKeyFromHex(keyStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(kring[0].Subkeys) == 0 {
		t.Fatalf("Subkey was not loaded")
	}
	// NOTE: GPG treats that subkey as signing subkey, but marks it as "unusable"
	// (because it lacks crosscertification). But for now assume we only care about
	// not being able to verify using this subkey, and whether it's available for
	// encryption is another matter.
	if kring[0].Subkeys[0].Sig.FlagSign {
		t.Fatalf("Got unexpected FlagSign=1")
	}
	if !kring[0].Subkeys[0].Sig.FlagEncryptCommunications {
		t.Fatalf("Got unexpected FlagEncrypt=0")
	}
	sig, err := armor.Decode(strings.NewReader(armoredMessage))
	if err != nil {
		t.Fatal(err)
	}
	md, err := ReadMessage(sig.Body, kring, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	if !md.IsSigned || md.SignedBy != nil {
		t.Fatalf("invalid MessageDetails signed state")
	}
}
