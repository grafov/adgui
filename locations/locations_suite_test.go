package locations_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLocationsSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Locations Suite")
}
