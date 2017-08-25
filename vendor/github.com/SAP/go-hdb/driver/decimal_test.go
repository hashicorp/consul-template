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

package driver

import (
	"math/big"
	"testing"
)

func TestDecimalInfo(t *testing.T) {
	t.Logf("maximum decimal value %v", maxDecimal)
	t.Logf("~log2(10): %f", lg10)
}

type testDigits10 struct {
	x      *big.Int
	digits int
}

var testDigits10Data = []*testDigits10{
	&testDigits10{new(big.Int).SetInt64(0), 1},
	&testDigits10{new(big.Int).SetInt64(1), 1},
	&testDigits10{new(big.Int).SetInt64(9), 1},
	&testDigits10{new(big.Int).SetInt64(10), 2},
	&testDigits10{new(big.Int).SetInt64(99), 2},
	&testDigits10{new(big.Int).SetInt64(100), 3},
	&testDigits10{new(big.Int).SetInt64(999), 3},
	&testDigits10{new(big.Int).SetInt64(1000), 4},
	&testDigits10{new(big.Int).SetInt64(9999), 4},
	&testDigits10{new(big.Int).SetInt64(10000), 5},
	&testDigits10{new(big.Int).SetInt64(99999), 5},
	&testDigits10{new(big.Int).SetInt64(100000), 6},
	&testDigits10{new(big.Int).SetInt64(999999), 6},
	&testDigits10{new(big.Int).SetInt64(1000000), 7},
	&testDigits10{new(big.Int).SetInt64(9999999), 7},
	&testDigits10{new(big.Int).SetInt64(10000000), 8},
	&testDigits10{new(big.Int).SetInt64(99999999), 8},
	&testDigits10{new(big.Int).SetInt64(100000000), 9},
	&testDigits10{new(big.Int).SetInt64(999999999), 9},
	&testDigits10{new(big.Int).SetInt64(1000000000), 10},
	&testDigits10{new(big.Int).SetInt64(9999999999), 10},
}

func TestDigits10(t *testing.T) {
	for i, d := range testDigits10Data {
		digits := digits10(d.x)
		if d.digits != digits {
			t.Fatalf("value %d int %s digits %d - expected digits %d", i, d.x, digits, d.digits)
		}
	}
}

type testRat struct {
	// in
	x      *big.Rat
	digits int
	minExp int
	maxExp int
	// out
	cmp *big.Int
	neg bool
	exp int
	df  decFlags
}

var testRatData = []*testRat{
	&testRat{new(big.Rat).SetFrac64(0, 1), 3, -2, 2, new(big.Int).SetInt64(0), false, 0, 0},                              //convert 0
	&testRat{new(big.Rat).SetFrac64(1, 1), 3, -2, 2, new(big.Int).SetInt64(1), false, 0, 0},                              //convert 1
	&testRat{new(big.Rat).SetFrac64(1, 10), 3, -2, 2, new(big.Int).SetInt64(1), false, -1, 0},                            //convert 1/10
	&testRat{new(big.Rat).SetFrac64(1, 99), 3, -2, 2, new(big.Int).SetInt64(1), false, -2, dfNotExact},                   //convert 1/99
	&testRat{new(big.Rat).SetFrac64(1, 100), 3, -2, 2, new(big.Int).SetInt64(1), false, -2, 0},                           //convert 1/100
	&testRat{new(big.Rat).SetFrac64(1, 1000), 3, -2, 2, new(big.Int).SetInt64(1), false, -3, dfUnderflow},                //convert 1/1000
	&testRat{new(big.Rat).SetFrac64(10, 1), 3, -2, 2, new(big.Int).SetInt64(1), false, 1, 0},                             //convert 10
	&testRat{new(big.Rat).SetFrac64(100, 1), 3, -2, 2, new(big.Int).SetInt64(1), false, 2, 0},                            //convert 100
	&testRat{new(big.Rat).SetFrac64(1000, 1), 3, -2, 2, new(big.Int).SetInt64(10), false, 2, 0},                          //convert 1000
	&testRat{new(big.Rat).SetFrac64(10000, 1), 3, -2, 2, new(big.Int).SetInt64(100), false, 2, 0},                        //convert 10000
	&testRat{new(big.Rat).SetFrac64(100000, 1), 3, -2, 2, new(big.Int).SetInt64(100), false, 3, dfOverflow},              //convert 100000
	&testRat{new(big.Rat).SetFrac64(999999, 1), 3, -2, 2, new(big.Int).SetInt64(100), false, 4, dfNotExact | dfOverflow}, //convert 999999
	&testRat{new(big.Rat).SetFrac64(99999, 1), 3, -2, 2, new(big.Int).SetInt64(100), false, 3, dfNotExact | dfOverflow},  //convert 99999
	&testRat{new(big.Rat).SetFrac64(9999, 1), 3, -2, 2, new(big.Int).SetInt64(100), false, 2, dfNotExact},                //convert 9999
	&testRat{new(big.Rat).SetFrac64(99950, 1), 3, -2, 2, new(big.Int).SetInt64(100), false, 3, dfNotExact | dfOverflow},  //convert 99950
	&testRat{new(big.Rat).SetFrac64(99949, 1), 3, -2, 2, new(big.Int).SetInt64(999), false, 2, dfNotExact},               //convert 99949

	&testRat{new(big.Rat).SetFrac64(1, 3), 5, -5, 5, new(big.Int).SetInt64(33333), false, -5, dfNotExact}, //convert 1/3
	&testRat{new(big.Rat).SetFrac64(2, 3), 5, -5, 5, new(big.Int).SetInt64(66667), false, -5, dfNotExact}, //convert 2/3
	&testRat{new(big.Rat).SetFrac64(11, 2), 5, -5, 5, new(big.Int).SetInt64(55), false, -1, 0},            //convert 11/2

}

func TestConvertRat(t *testing.T) {
	m := new(big.Int)

	for i := 0; i < 1; i++ { // use for performance tests
		for j, d := range testRatData {
			neg, exp, df := convertRatToDecimal(d.x, m, d.digits, d.minExp, d.maxExp)
			if m.Cmp(d.cmp) != 0 || neg != d.neg || exp != d.exp || df != d.df {
				t.Fatalf("converted %d value m %s neg %t exp %d df %b - expected m %s neg %t exp %d df %b", j, m, neg, exp, df, d.cmp, d.neg, d.exp, d.df)
			}
		}
	}
}
