package parser

import (
	"github.com/creachadair/mds/mapset"
)

// ValidateOffline runs offline validations on a parsed PSL.
func ValidateOffline(l *List) []error {
	var ret []error

	for _, block := range BlocksOfType[*Section](l) {
		if block.Name == "PRIVATE DOMAINS" {
			ret = append(ret, validateEntityMetadata(block)...)
			break
		}
	}
	validateExpectedSections(l)
	validateSuffixUniqueness(l)

	return ret
}

// validateEntityMetadata verifies that all suffix blocks have some
// kind of entity name.
func validateEntityMetadata(block Block) []error {
	var ret []error
	for _, block := range BlocksOfType[*Suffixes](block) {
		if !block.Changed() {
			continue
		}

		if block.Info.Name == "" {
			ret = append(ret, ErrMissingEntityName{
				Suffixes: block,
			})
		}
		if len(block.Info.Maintainers) == 0 && !exemptFromContactInfo(block.Info.Name) {
			ret = append(ret, ErrMissingEntityEmail{
				Suffixes: block,
			})
		}
	}
	return ret
}

// validateExpectedSections verifies that the two top-level sections
// (ICANN and private domains) exist, are not duplicated, and that no
// other sections are present.
func validateExpectedSections(block Block) (errs []error) {
	// Use an ordered set for the wanted sections, so that we can
	// check section names in O(1) but also report missing sections in
	// a deterministic order.
	wanted := mapset.New("ICANN DOMAINS", "PRIVATE DOMAINS")
	found := map[string]*Section{}
	for _, section := range BlocksOfType[*Section](block) {
		if !wanted.Has(section.Name) && section.Changed() {
			errs = append(errs, ErrUnknownSection{section})
		} else if other, ok := found[section.Name]; ok && (section.Changed() || other.Changed()) {
			errs = append(errs, ErrDuplicateSection{section, other})
		} else {
			found[section.Name] = section
		}
	}

	for _, name := range wanted.Slice() {
		if _, ok := found[name]; !ok {
			errs = append(errs, ErrMissingSection{name})
		}
	}

	return errs
}

// validateSuffixUniqueness verifies that suffixes only appear once
// each.
func validateSuffixUniqueness(block Block) (errs []error) {
	suffixes := map[string]*Suffix{}    // domain.Name.String() -> Suffix
	wildcards := map[string]*Wildcard{} // base domain.Name.String() -> Wildcard

	for _, suffix := range BlocksOfType[*Suffix](block) {
		name := suffix.Domain.String()
		if other, ok := suffixes[name]; ok && (suffix.Changed() || other.Changed()) {
			errs = append(errs, ErrDuplicateSuffix{name, suffix, other})
		} else {
			suffixes[name] = suffix
		}
	}

	for _, wildcard := range BlocksOfType[*Wildcard](block) {
		name := wildcard.Domain.String()
		if other, ok := wildcards[name]; ok && (wildcard.Changed() || other.Changed()) {
			errs = append(errs, ErrDuplicateSuffix{"*." + name, wildcard, other})
		} else {
			wildcards[name] = wildcard
		}

		for _, exc := range wildcard.Exceptions {
			fqdn, err := wildcard.Domain.AddPrefix(exc)
			if err != nil && wildcard.Changed() {
				errs = append(errs, err)
				continue
			}
			name := fqdn.String()
			if suffix, ok := suffixes[name]; ok && (wildcard.Changed() || suffix.Changed()) {
				errs = append(errs, ErrConflictingSuffixAndException{suffix, wildcard})
			}
		}
	}

	return errs
}
