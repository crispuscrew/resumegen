package guard

func TrimIsNeeded(pageCount, maxPages float64) bool {
	return pageCount > maxPages
}