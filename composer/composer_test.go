package composer_test

import (
	"github.com/Lavoaster/cloudsmith-sync/composer"
	"testing"
)

var branchNameTests = [][]string{
	{"master", "dev-master", "9999999-dev"},
	{"develop", "dev-develop", "dev-develop"},
	{"feature/something-good", "dev-feature/something-good", "dev-feature/something-good"},
}
var tagNameTests = [][]string{
	{"5.0", "5.0.x-dev", "5.0.9999999.9999999-dev"},
	{"4", "4.x-dev", "4.9999999.9999999.9999999-dev"},
	{"4.x", "4.x-dev", "4.9999999.9999999.9999999-dev"},
}

func TestDeriveVersion(t *testing.T) {
	for _, test := range branchNameTests {
		input := test[0]
		expected1 := test[1]
		expected2 := test[2]

		actual1, actual2, _ := composer.DeriveVersion(input, true)

		if actual1 != expected1 || actual2 != expected2 {
			t.Errorf("[!] DeriveVersion(%s, true) = %v, %v; want %v, %v", input, actual1, actual2, expected1, expected2)
		}
	}
	for _, test := range tagNameTests {
		input := test[0]
		expected1 := test[1]
		expected2 := test[2]

		actual1, actual2, _ := composer.DeriveVersion(input, true)

		if actual1 != expected1 || actual2 != expected2 {
			t.Errorf("[!] DeriveVersion(%s, true) = %v, %v; want %v, %v", input, actual1, actual2, expected1, expected2)
		}
	}
}
