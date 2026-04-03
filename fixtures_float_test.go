package taffy

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Float Fixtures", func() {
	fixtureDir := "test_fixtures/float"
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		Fail("failed to read fixture dir: " + err.Error())
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".xml" {
			continue
		}
		entry := entry
		It(entry.Name(), func() {
			runFixtureTest(filepath.Join(fixtureDir, entry.Name()))
		})
	}
})
