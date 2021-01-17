package s3store

import (
	"testing"

	"github.com/sequix/casync-snapshotter/pkg/log"
)

func Test(t *testing.T) {
	c, err := New()
	if err != nil {
		log.G.Fatal(err)
	}
	log.G.Info(c.HasChunk("caidx/e2c64cbb4638b06e84b5503cc99c45a21e9825431f6739d9485dd9cb8173b85b.caidx"))
}