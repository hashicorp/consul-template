// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package armor

import (
	"bytes"
	"hash/adler32"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"fmt"
)

func TestDecodeEncode(t *testing.T) {
	buf := bytes.NewBuffer([]byte(armorExample1))
	result, err := Decode(buf)
	if err != nil {
		t.Error(err)
	}
	expectedType := "PGP SIGNATURE"
	if result.Type != expectedType {
		t.Errorf("result.Type: got:%s want:%s", result.Type, expectedType)
	}
	if len(result.Header) != 1 {
		t.Errorf("len(result.Header): got:%d want:1", len(result.Header))
	}
	v, ok := result.Header["Version"]
	if !ok || v != "GnuPG v1.4.10 (GNU/Linux)" {
		t.Errorf("result.Header: got:%#v", result.Header)
	}

	contents, err := ioutil.ReadAll(result.Body)
	if err != nil {
		t.Error(err)
	}

	if adler32.Checksum(contents) != 0x27b144be {
		t.Errorf("contents: got: %x", contents)
	}

	buf = bytes.NewBuffer(nil)
	w, err := Encode(buf, result.Type, result.Header)
	if err != nil {
		t.Error(err)
	}
	_, err = w.Write(contents)
	if err != nil {
		t.Error(err)
	}
	w.Close()

	if !bytes.Equal(buf.Bytes(), []byte(armorExample1)) {
		t.Errorf("got: %s\nwant: %s", string(buf.Bytes()), armorExample1)
	}
}

func TestLongHeader(t *testing.T) {
	buf := bytes.NewBuffer([]byte(armorLongLine))
	result, err := Decode(buf)
	if err != nil {
		t.Error(err)
		return
	}
	value, ok := result.Header["Version"]
	if !ok {
		t.Errorf("missing Version header")
	}
	if value != longValueExpected {
		t.Errorf("got: %s want: %s", value, longValueExpected)
	}
}

func decodeAndReadAll(t *testing.T, armor string) (*Block, string) {
	result, err := Decode(bytes.NewBuffer([]byte(armor)))
	if err != nil {
		t.Fatal(err)
	}

	dataStr, err := ioutil.ReadAll(result.Body)
	if err != nil {
		fmt.Printf("Failing payload is:\n\n%s\n", armor)
		t.Errorf("Error after ReadAll: %+v", err)
	}

	return result, string(dataStr)
}

func decodeAndReadShortReads(t *testing.T, armor string) (ret string) {
	result, err := Decode(bytes.NewBuffer([]byte(armor)))
	if err != nil {
		t.Fatal(err)
	}

	var readLengths = [...]int{3, 3, 5, 6, 4, 5, 6, 1, 1, 4, 2, 2}
	var p [10]byte
	i := 0
	for {
		n := readLengths[i%len(readLengths)]
		i++
		z, err := result.Body.Read(p[:n])
		ret += string(p[:z])
		if err == io.EOF {
			break
		} else if err != nil {
			t.Error(err)
			break
		}
	}

	return ret
}

func decodeAndRead(t *testing.T, armor string) {
	_, ret1 := decodeAndReadAll(t, armor)
	ret2 := decodeAndReadShortReads(t, armor)
	if ret1 != ret2 {
		t.Errorf("ReadAll and short reads didn't agree: %q != %q", ret1, ret2)
	}
}

func decodeAndReadFail(t *testing.T, expectedErr string, armor string) *Block {
	result, err := Decode(bytes.NewBuffer([]byte(armor)))
	if err != nil {
		t.Fatal(err)
	}

	_, err = ioutil.ReadAll(result.Body)
	if err == nil {
		fmt.Printf("Expecting payload to fail:\n\n%s\n", armor)
		t.Errorf("Expected an error after ReadAll")
	} else {
		if !strings.Contains(err.Error(), expectedErr) {
			t.Errorf("Expected error to contain %q, but it's: %v", expectedErr, err)
		}
	}

	return result
}

func TestZeroWidthSpace(t *testing.T) {
	decodeAndRead(t, armorZeroWidthSpace)

	result, _ := decodeAndReadAll(t, armorMoreZeroWidths)
	if result.lReader.crc == nil {
		// Make sure that ZERO-WIDTH SPACE did not mess with crc reading.
		t.Error("Expected CRC to be read")
	}
}

func TestNoNewlines(t *testing.T) {
	decodeAndRead(t, armorNoNewlines)
	decodeAndRead(t, armorNoNewlines2)
}

func TestFoldedCRC(t *testing.T) {
	// KBPGP does not fail to read here, but it doesn't discover CRC
	// (discards the folded in CRC as garbage), and it's probably the
	// right behavior to aim for here.

	result, _ := decodeAndReadAll(t, armorNoNewlinesBrokenCRC)
	if result.lReader.crc == nil {
		// Make sure that ZERO-WIDTH SPACE did not mess with crc reading.
		t.Error("Expected CRC to be read")
	}

	decodeAndRead(t, foldedCRC2)
}

func TestWhitespaceInCRC(t *testing.T) {
	result, _ := decodeAndReadAll(t, armorWhitespaceInCRC)
	if result.lReader.crc == nil {
		t.Error("Expected CRC to be read")
	}
}

func TestEmptyLinesAfterCRC(t *testing.T) {
	decodeAndRead(t, emptyLinesAfterCRC1)
	decodeAndRead(t, emptyLinesAfterCRC2)
}

func TestEntirePayloadIsOneLineWithCRC(t *testing.T) {
	_, ret1 := decodeAndReadAll(t, everythingIsOneLineIncludingCRC)
	if ret1 != "hello world 1\n" {
		t.Error("Invalid data returned when dearmoring")
	}
	ret2 := decodeAndReadShortReads(t, everythingIsOneLineIncludingCRC)
	if ret1 != ret2 {
		t.Error("short reads did not match what ReadAll returned")
	}
}

var armorErrorText = "invalid data: armor invalid"

func TestNoArmorEnd(t *testing.T) {
	removeArmorEnd := func(str string) string {
		return strings.Replace(str, "-----END PGP SIGNATURE-----", "", 1)
	}

	decodeAndReadFail(t, armorErrorText, removeArmorEnd(armorExample1))
	decodeAndReadFail(t, armorErrorText, removeArmorEnd(armorNoNewlines))
	decodeAndReadFail(t, armorErrorText, removeArmorEnd(armorNoNewlines2))
	decodeAndReadFail(t, armorErrorText, removeArmorEnd(emptyLinesAfterCRC1))
	decodeAndReadFail(t, armorErrorText, removeArmorEnd(emptyLinesAfterCRC2))
}

func TestMalformedCRCs(t *testing.T) {
	// Test CRC being in random places in payload trying to confuse our parser.
	decodeAndReadFail(t, armorErrorText, confuseArmorAndCRC)
	decodeAndReadFail(t, armorErrorText, testBadCRC)

	decodeAndReadFail(t, armorErrorText, testMultipleCrcs)
	decodeAndReadFail(t, armorErrorText, testMultipleCrcs2)

	decodeAndReadFail(t, "error decoding CRC: wrong size CRC", testNonDecodableCRC)
	decodeAndReadFail(t, "error decoding CRC: illegal base64 data at input byte 1", testNonDecodableCRC2)

	decodeAndReadFail(t, armorErrorText, stuffAfterChecksum1)
	decodeAndReadFail(t, armorErrorText, stuffAfterChecksum2)
}

const armorExample1 = `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1.4.10 (GNU/Linux)

iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm
4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt
p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW
TxRjs+fJCIFuo71xb1g=
=/teI
-----END PGP SIGNATURE-----
`

const armorLongLine = `-----BEGIN PGP SIGNATURE-----
Version: 0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz

iQEcBAABAgAGBQJMtFESAAoJEKsQXJGvOPsVj40H/1WW6jaMXv4BW+1ueDSMDwM8
kx1fLOXbVM5/Kn5LStZNt1jWWnpxdz7eq3uiqeCQjmqUoRde3YbB2EMnnwRbAhpp
cacnAvy9ZQ78OTxUdNW1mhX5bS6q1MTEJnl+DcyigD70HG/yNNQD7sOPMdYQw0TA
byQBwmLwmTsuZsrYqB68QyLHI+DUugn+kX6Hd2WDB62DKa2suoIUIHQQCd/ofwB3
WfCYInXQKKOSxu2YOg2Eb4kLNhSMc1i9uKUWAH+sdgJh7NBgdoE4MaNtBFkHXRvv
okWuf3+xA9ksp1npSY/mDvgHijmjvtpRDe6iUeqfCn8N9u9CBg8geANgaG8+QA4=
=wfQG
-----END PGP SIGNATURE-----`

const longValueExpected = "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"

const armorZeroWidthSpace = `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1.4.10 (GNU/Linux)
` + "\u200b" + `
iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm
4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt
p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW
TxRjs+fJCIFuo71xb1g=
=/teI
-----END PGP SIGNATURE-----
`

const armorMoreZeroWidths = `-----BEGIN PGP SIGNATURE-----` + "\u200b" + `
Version: GnuPG v1.4.10 (GNU/Linux)

iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm` + "\u200b" + `
4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt
` + "\u200b" + `p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW
TxRjs+fJCIFuo71xb1g=` + "\u200b" + `
` + "\u200b" + `=/teI
-----END PGP SIGNATURE-----`

const armorNoNewlines = `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1.4.10 (GNU/Linux)

iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm ` +
	`4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt ` +
	`p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW ` +
	`TxRjs+fJCIFuo71xb1g=
=/teI
-----END PGP SIGNATURE-----
`

const armorNoNewlines2 = `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1.4.10 (GNU/Linux)

iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm ` + "\t" +
	`4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt     ` +
	`p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW       ` + `
TxRjs+fJCIFuo71xb1g=
=/teI
-----END PGP SIGNATURE-----
`

// The last line ("=/teI") is folded into rest of the armor, meaning
// the crc sum will not be found and checked.
const armorNoNewlinesBrokenCRC = `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1.4.10 (GNU/Linux)

iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm ` + "\t" +
	`4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt     ` +
	`p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW       ` +
	`TxRjs+fJCIFuo71xb1g==/teI
-----END PGP SIGNATURE-----
`

const armorWhitespaceInCRC = `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1.4.10 (GNU/Linux)

iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm ` + "\t" +
	`4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt     ` +
	`p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW       ` + `
TxRjs+fJCIFuo71xb1g=
=/teI` + "\u200b " + `
-----END PGP SIGNATURE-----
`

// Same as the key above but without shenanigans, just no newline
// before CRC.
const foldedCRC2 = `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1.4.10 (GNU/Linux)

iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm
4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt
p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW
TxRjs+fJCIFuo71xb1g==/teI
-----END PGP SIGNATURE-----
`

const emptyLinesAfterCRC1 = `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1.4.10 (GNU/Linux)

iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm
4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt
p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW
TxRjs+fJCIFuo71xb1g=
=/teI



-----END PGP SIGNATURE-----
`

const emptyLinesAfterCRC2 = `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1.4.10 (GNU/Linux)

iJwEAAECAAYFAk1Fv/0ACgkQo01+GMIMMbsYTwQAiAw+QAaNfY6WBdplZ/uMAccm
4g+81QPmTSGHnetSb6WBiY13kVzK4HQiZH8JSkmmroMLuGeJwsRTEL4wbjRyUKEt
p1xwUZDECs234F1xiG5enc5SGlRtP7foLBz9lOsjx+LEcA4sTl5/2eZR9zyFZqWW
TxRjs+fJCIFuo71xb1g==/teI



-----END PGP SIGNATURE-----
`

// Really short base64 payload here so it's guaranteed not to be
// buffered and returned as two lines by linereader.
const everythingIsOneLineIncludingCRC = `-----BEGIN PGP ARMORED FILE-----
Comment: Use "gpg --dearmor" for unpacking

aGVsbG8gd29ybGQgMQo==hvYi
-----END PGP ARMORED FILE-----
`

const confuseArmorAndCRC = `-----BEGIN PGP ARMORED FILE-----
Comment: Use "gpg --dearmor" for unpacking

SHVoIGhlbGxvIHdvcmxkCg==
-----END PGP ARMORED FILE-----=n5G6
`

const testBadCRC = `-----BEGIN PGP ARMORED FILE-----
Comment: Use "gpg --dearmor" for unpacking

SHVoIGhlbGxvIHdvcmxkCg==
=n5G5
-----END PGP ARMORED FILE-----
`

const testMultipleCrcs = `-----BEGIN PGP ARMORED FILE-----

SHVoIGhlbGxvIHdvcmxkCg===n5G6
=n5G6
-----END PGP ARMORED FILE-----
`

const testMultipleCrcs2 = `-----BEGIN PGP ARMORED FILE-----

SHVoIGhlbGxvIHdvcmxkCg==
=n5G6
=n5G6
-----END PGP ARMORED FILE-----
`

const testNonDecodableCRC = `-----BEGIN PGP ARMORED FILE-----

SHVoIGhlbGxvIHdvcmxkCg==
=n9==
-----END PGP ARMORED FILE-----
`

const testNonDecodableCRC2 = `-----BEGIN PGP ARMORED FILE-----

SHVoIGhlbGxvIHdvcmxkCg==
=n%!@
-----END PGP ARMORED FILE-----
`

const stuffAfterChecksum1 = `-----BEGIN PGP ARMORED FILE-----

aGVsbG8gdGVzdCBoZWxsbyB0ZXN0IGhlbGxvIHRlc3QgaGVsbG8gdGVzdCBoZWxs
=9XyA
byB0ZXN0IGhlbGxvIHRlc3QgaGVsbG8gdGVzdCBoZWxsbyB0ZXN0IA==
-----END PGP ARMORED FILE-----
`

const stuffAfterChecksum2 = `-----BEGIN PGP ARMORED FILE-----

aGVsbG8gdGVzdCBoZWxsbyB0ZXN0IGhlbGxvIHRlc3QgaGVsbG8g
dGVzdCBoZWxs=9XyA
byB0ZXN0IGhlbGxvIHRlc3QgaGVsbG8gdGVzdCBoZWxsbyB0ZXN0IA==
=9XyA
-----END PGP ARMORED FILE-----

`
