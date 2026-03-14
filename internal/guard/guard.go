package guard

import "log"

func TrimIsNeeded(pageCount, maxPages float64) bool {
	if maxPages < 0 { log.Fatalf("Invalid page limit: %f. Page limit must be a non-negative number.", maxPages) }
	if maxPages == 0 { return false }
	return pageCount > maxPages
}