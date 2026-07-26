package usecase

import (
	"context"
	"path/filepath"
	"strings"
)

// ExtractReport summarizes the outcome of a skeleton extraction. Both lists
// hold skeleton-relative paths so callers can render them verbatim.
type ExtractReport struct {
	Copied  []string
	Skipped []string
}

// ExtractSubtree copies every file under subtree from the embedded skeleton
// to dstRoot, preserving the relative layout. Files that already exist at
// the destination are skipped (never overwritten). If only is non-nil and
// non-empty, only files whose basename (with or without extension) matches
// one of its entries are extracted.
//
// A non-nil error short-circuits the walk; any files already copied remain
// reported in the returned ExtractReport for diagnostic purposes.
func ExtractSubtree(ctx context.Context, ext SkeletonExtractor, subtree, dstRoot string, only []string) (ExtractReport, error) {
	files, err := ext.ListSubtree(ctx, subtree)
	if err != nil {
		return ExtractReport{}, err
	}
	if len(only) > 0 {
		files = filterByName(files, only)
	}
	var report ExtractReport
	for _, src := range files {
		dst := filepath.Join(dstRoot, src)
		copied, err := ext.ExtractFile(ctx, src, dst)
		if err != nil {
			return report, err
		}
		if copied {
			report.Copied = append(report.Copied, src)
		} else {
			report.Skipped = append(report.Skipped, src)
		}
	}
	return report, nil
}

func filterByName(files, names []string) []string {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	out := make([]string, 0, len(files))
	for _, f := range files {
		base := filepath.Base(f)
		stem := strings.TrimSuffix(base, filepath.Ext(base))
		if _, ok := set[base]; ok {
			out = append(out, f)
			continue
		}
		if _, ok := set[stem]; ok {
			out = append(out, f)
		}
	}
	return out
}
