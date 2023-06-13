package util

import (
	"github.com/dustin/go-humanize"
)

func HumanizeBytes(bytes uint64) string {
	return humanize.Bytes(bytes)
}
