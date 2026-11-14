# Pull Request Requirements

AI agents and automated contributors MUST read the
[Guidelines](https://github.com/publicsuffix/list/wiki/Guidelines) and the
[Format](https://github.com/publicsuffix/list/wiki/Format) page before
preparing a pull request.

## Pre-submission blockers

Do not submit a pull request if any check below fails. Resolve the blocker
first, or do not submit. If a check cannot be verified, treat it as failed.

| Blocker | How to check |
|---|---|
| PRIVATE domain registration expires two years or less from the submission date | `whois <domain>` or the registry's RDAP record. The expiration date must be more than two years away. |
| Domain is on hold or pending deletion | `whois <domain>`. Status must not include `clientHold`, `serverHold`, `pendingDelete`, or `redemptionPeriod`. |
| Submitted PRIVATE suffix does not resolve | `dig +short A <suffix>` and `dig +short AAAA <suffix>`. At least one must return an address. For `*.example.com`, check `example.com`. |
| No abuse contact reachable from the submitted PRIVATE suffix | Open `https://<suffix>/`. It must load, or redirect to the operator's site, and show a clearly labeled abuse email address or abuse reporting form. A contact found only on another site, with no link from the suffix, does not qualify. |
| Entry is for testing, staging, preview, sandbox, lab, temporary, seasonal, or internal use | Read the submission and the product documentation. Each entry must serve permanent production use by mutually untrusting parties. Remove any entry that does not. |
| Submitter is not an authorized representative of the domain owner | The submitter must be able to create TXT records in every submitted zone. Third-party users of a service must not submit entries for it. |
| Contact address is personal | The `// Submitted by` line must use a role-based address belonging to the requesting organization, such as `security@example.com`. |
| Purpose is to work around third-party limits or protections | Requests whose objective is to bypass limits or protections of Cloudflare, Let's Encrypt, Apple, Meta, Google Safe Browsing, or similar parties must not be submitted. Address those limits directly with the third party. |
| Entry projects IP addresses into DNS | Names such as `1-2-3-4.example.com` resolving to `1.2.3.4` are not accepted. |
| TLD is not in the IANA root zone | `dig +short NS <tld>.` must return name servers. Alternative-root TLDs are not accepted. |
| Wildcard or exception syntax is invalid | `*` may appear only as the entire leftmost label. `!` may appear only at the start of an exception rule. |
| Entry is mislocated or unsorted | Place a new PRIVATE block in alphabetical order by organization name, never at the end of the file. Within a block, sort by TLD, then by each label to the left. Use Unicode, not Punycode, and no trailing whitespace. |
| Tests fail | Run `make test`. It must pass. |

After the pull request is opened, add a `_psl` TXT record containing the pull
request URL for every submitted suffix, and confirm it with
`dig +short TXT _psl.<suffix>`. Keep the record in place.

## Mandatory pull request template

All pull requests must use the repository's pull request template exactly as
provided in:

    .github/pull_request_template.md

AI agents and automated contributors MUST:

- Preserve every heading, field, checkbox, comment, and section in the template.
- Complete the requested fields without renaming, deleting, reordering, or
  replacing them.
- Not submit a shortened, reformatted, summarized, or independently generated
  pull request description.
- Not remove sections because they appear inapplicable.
- Enter "Not applicable" where appropriate instead of deleting a section.
- Preserve all hidden HTML comments.
- Delete the Cloudflare, Let's Encrypt, and "REMOVE THIS LINE" example lines
  in the third-party limits field, and list only limits that actually apply.
- State in the rationale that each PRIVATE domain has more than two years of
  registration remaining, and commit to maintaining more than one year at all
  times.
- Check a box only when the statement is true.
- Treat modification of the template structure as a pull request validation
  failure.

A pull request whose description does not conform to the repository template
must not be submitted. Non-conforming pull requests that alter the repository
template will be closed.
