package parser

// Validate runs validations on a parsed File.
//
// Validation only runs on a file that does not yet have any
// errors. The presence of errors may indicate structural issues that
// can break some validations.
func (p *parser) Validate() {
	if len(p.Errors) > 0 {
		return
	}

	p.requireEntityNames()
	p.requirePrivateDomainEmailContact()
}

// requireEntityNames verifies that all Suffix blocks have some kind
// of entity name.
func (p *parser) requireEntityNames() {
	for _, block := range p.AllSuffixBlocks() {
		if block.Entity == "" {
			p.addError(MissingEntityName{
				Suffixes: block,
			})
		}
	}
}

// requirePrivateDomainEmailContact verifies that all Suffix blocks in
// the private section have email contact information.
func (p *parser) requirePrivateDomainEmailContact() {
	for _, block := range p.File.SuffixBlocksInSection("PRIVATE DOMAINS") {
		if block.Submitter == nil {
			p.addError(MissingEntityEmail{
				Suffixes: block,
			})
		}
	}
}
