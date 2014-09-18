package bqschema_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBqschema(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bqschema Suite")
}
