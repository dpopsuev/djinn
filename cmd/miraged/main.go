// djinnd — the Djinn Substrate substrate.
//
// Stub: prints not-yet-implemented message.
// The real daemon will manage agent lifecycle, space isolation,
// enrichment, and observation. See DJN-GOL-108.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "djinnd: not yet implemented — see DJN-GOL-108 (Substrate)")
	os.Exit(1)
}
