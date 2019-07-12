// +build p9debug

package p9

import (
	"fmt"
	"os"
)

func debugLog(str string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, str, args...)
}
