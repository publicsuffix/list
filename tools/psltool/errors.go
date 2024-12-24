package main

import (
	"errors"
	"reflect"

	"github.com/publicsuffix/list/tools/internal/parser"
)

const (
	invalidTag  = "❌invalid"
	dnsFailTag  = "❌FAIL - DNS VALIDATION"
	sortFailTag = "❌FAIL - FIX SORTING ⏬"
)

var errToLabel = map[error]string{
	// all parser errors, sort in alphabetical order
	//parser.ErrCommentPreventsSectionSort{}:    "",
	parser.ErrConflictingSuffixAndException{}: invalidTag,
	//parser.ErrCommentPreventsSuffixSort{}:     "",
	parser.ErrDuplicateSection{}:     invalidTag,
	parser.ErrDuplicateSuffix{}:      invalidTag,
	parser.ErrInvalidEncoding{}:      invalidTag,
	parser.ErrInvalidSuffix{}:        invalidTag,
	parser.ErrInvalidUnicode{}:       invalidTag,
	parser.ErrMissingEntityEmail{}:   invalidTag,
	parser.ErrMissingEntityName{}:    invalidTag,
	parser.ErrMissingSection{}:       invalidTag,
	parser.ErrMissingTXTRecord{}:     dnsFailTag,
	parser.ErrMismatchedSection{}:    invalidTag,
	parser.ErrNestedSection{}:        invalidTag,
	parser.ErrSectionInSuffixBlock{}: invalidTag,
	parser.ErrTXTCheckFailure{}:      dnsFailTag,
	parser.ErrTXTRecordMismatch{}:    dnsFailTag,
	parser.ErrUnclosedSection{}:      invalidTag,
	parser.ErrUnknownSection{}:       invalidTag,
	parser.ErrUnknownSectionMarker{}: invalidTag,
	parser.ErrUnknownSectionMarker{}: invalidTag,
	parser.ErrUnclosedSection{}:      invalidTag,

	// all other errors
	ErrReformat: sortFailTag,
}

var (
	ErrReformat = errors.New("file needs reformatting, run 'psltool fmt' to fix")
)

func errorsToLabels(errs []error) []string {
	labels := make([]string, 0, len(errs))

	var (
		sortSuccess = true
		dnsSuccess  = true
	)
	setLabel := func(label string) {
		switch label {
		case sortFailTag:
			sortSuccess = false
		case dnsFailTag:
			dnsSuccess = false
		}
		labels = append(labels, label)
	}

	for _, err := range errs {
		if label, ok := errToLabel[err]; ok {
			setLabel(label)
			continue
		}
		for tpl, label := range errToLabel {
			if isType(err, tpl) {
				setLabel(label)
				break
			}
		}
	}

	if sortSuccess {
		labels = append(labels, "✔️Sorting Validated")
	}
	if dnsSuccess {
		labels = append(labels, "✔️DNS _psl Validated")
	}

	return labels
}

func isType(err error, tpl error) bool {
	if errors.Is(err, tpl) {
		return true
	}
	if reflect.TypeOf(err) == reflect.TypeOf(tpl) {
		return true
	}
	if wraped, ok := err.(interface{ Unwrap() error }); ok {
		return isType(wraped.Unwrap(), tpl)
	}
	return false
}
