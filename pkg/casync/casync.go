package casync

import (
	"flag"
	"fmt"
	"os/exec"

	"github.com/sequix/casync-snapshotter/pkg/fs"
	"github.com/sequix/casync-snapshotter/pkg/log"
)

var (
	flagCasyncPath = flag.String("-casync-path", "", "path to casync binary, by default, search it from PATH")
)

var (
	casyncPath string
)

func Init() {
	casyncPath = *flagCasyncPath
	if len(casyncPath) == 0 {
		cp, err := exec.LookPath("casync")
		if err != nil {
			log.G.Fatalf("not found casync binary in PATH: %s", err)
		}
		casyncPath = cp
	}
}

// make seed and chunk to a temp dir
func Make(srcDir, castr, caidx string) error {
	out, err := exec.Command(casyncPath, "make", "--store", castr, caidx, srcDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("caync make: output %s, error %w", string(out), err)
	}
	return nil
}

// mount from http endpoint
func Mount(dstDir, castr, caidx string) (fs.UmountFunc, error) {
	panic("todo")
}
