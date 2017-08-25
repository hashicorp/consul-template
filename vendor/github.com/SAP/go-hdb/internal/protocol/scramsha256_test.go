/*
Copyright 2014 SAP SE

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package protocol

import (
	"testing"
)

type testRecord struct {
	salt            []byte
	serverChallenge []byte
	clientChallenge []byte
	password        []byte
	clientProof     []byte
}

var testData = []*testRecord{
	&testRecord{
		salt:            []byte{214, 199, 255, 118, 92, 174, 94, 190, 197, 225, 57, 154, 157, 109, 119, 245},
		serverChallenge: []byte{224, 22, 242, 18, 237, 99, 6, 28, 162, 248, 96, 7, 115, 152, 134, 65, 141, 65, 168, 126, 168, 86, 87, 72, 16, 119, 12, 91, 227, 123, 51, 194, 203, 168, 56, 133, 70, 236, 230, 214, 89, 167, 130, 123, 132, 178, 211, 186},
		clientChallenge: []byte{219, 141, 27, 200, 255, 90, 182, 125, 133, 151, 127, 36, 26, 106, 213, 31, 57, 89, 50, 201, 237, 11, 158, 110, 8, 13, 2, 71, 9, 235, 213, 27,
			64, 43, 181, 181, 147, 140, 10, 63, 156, 133, 133, 165, 171, 67, 187, 250, 41, 145, 176, 164, 137, 54, 72, 42, 47, 112, 252, 77, 102, 152, 220, 223},
		password:    []byte{65, 100, 109, 105, 110, 49, 50, 51, 52},
		clientProof: []byte{0, 1, 32, 23, 243, 209, 70, 117, 54, 25, 92, 21, 173, 194, 108, 63, 25, 188, 185, 230, 61, 124, 190, 73, 80, 225, 126, 191, 119, 32, 112, 231, 72, 184, 199},
	},
}

func TestScramsha256(t *testing.T) {

	for _, r := range testData {
		clientProof := clientProof(r.salt, r.serverChallenge, r.clientChallenge, r.password)

		for i, v := range clientProof {
			if v != r.clientProof[i] {
				t.Fatalf("diff index % d - got %v - expected %v", i, clientProof, r.clientProof)
			}
		}

	}
}
