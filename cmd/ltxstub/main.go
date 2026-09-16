// Command ltxstub writes a minimal valid LTX file (header + empty page block + page
// index + trailer) for one TXID with WALOffset=32/WALSize=0 and zero WAL
// salts, so litestream's verify step sees a salt mismatch and writes a fresh
// full snapshot at the next TXID instead of continuing incrementally.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/superfly/ltx"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "usage: ltxstub OUT TXID_HEX PAGESIZE COMMIT_PAGES")
		os.Exit(2)
	}
	txid, err := strconv.ParseUint(os.Args[2], 16, 64)
	must(err)
	pageSize, err := strconv.ParseUint(os.Args[3], 10, 32)
	must(err)
	commit, err := strconv.ParseUint(os.Args[4], 10, 32)
	must(err)

	f, err := os.Create(os.Args[1])
	must(err)
	enc, err := ltx.NewEncoder(f)
	must(err)
	must(enc.EncodeHeader(ltx.Header{
		Version:   ltx.Version,
		Flags:     ltx.HeaderFlagNoChecksum,
		PageSize:  uint32(pageSize),
		Commit:    uint32(commit),
		MinTXID:   ltx.TXID(txid),
		MaxTXID:   ltx.TXID(txid),
		Timestamp: time.Now().UnixMilli(),
		WALOffset: 32,
		WALSize:   0,
	}))
	must(enc.Close())
	must(f.Sync())
	must(f.Close())
	fmt.Println("wrote", os.Args[1])
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
