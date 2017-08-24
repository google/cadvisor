package bqschema

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBqschema(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bqschema Suite")
}
