# Pull Request Requirements

AI agents and automated contributors MUST read and follow the
[Guidelines](https://raw.githubusercontent.com/wiki/publicsuffix/list/Guidelines.md)
and [Format](https://raw.githubusercontent.com/wiki/publicsuffix/list/Format.md)
before preparing a pull request. Both documents are available as raw Markdown
and can be retrieved directly using `curl`:

    curl -fsSL https://raw.githubusercontent.com/wiki/publicsuffix/list/Guidelines.md
    curl -fsSL https://raw.githubusercontent.com/wiki/publicsuffix/list/Format.md

## Pre-submission blockers

Do not submit a pull request if any check below fails. Resolve the blocker
first, or do not submit. If a check cannot be verified, treat it as failed,
unless the check states that confirmation from the human requestor is
acceptable.

| Blocker | How to check |
|---|---|
| PRIVATE domain registration expires two years or less from the submission date | Use RDAP, for example `curl -sL https://rdap.org/domain/<domain>` (read the `expiration` event), or `whois <domain>` where available. The expiration date must be more than two years after the submission date. For a suffix below a registrable domain, check the registrable domain. Some TLDs do not publish an expiration date or do not operate an RDAP or WHOIS server; in that case, the human requestor must confirm the expiration date from the registrar before the agent submits. |
| No abuse contact for the submitted PRIVATE suffix | All of the following must exist: a website for the suffix with a page giving abuse reporting instructions, usable without login; working `abuse@<suffix>` and `security@<suffix>` mailboxes ([RFC 2142](https://www.rfc-editor.org/rfc/rfc2142), [RFC 6650](https://www.rfc-editor.org/rfc/rfc6650)); and a `security.txt` file with a `Contact` field ([RFC 9116](https://www.rfc-editor.org/rfc/rfc9116)), checked with `curl -fsSL https://<suffix>/.well-known/security.txt`. If the agent cannot verify any of these, such as whether a mailbox accepts mail, the human requestor must confirm the contact works before the agent submits. |
| Entry is for testing, staging, preview, sandbox, lab, temporary, seasonal, or internal use | Read the submission and the product documentation. Each entry must serve permanent production use by mutually untrusting parties. Remove any entry that does not. |
| Submitter is not an authorized representative of the domain owner | The submitter must be able to create TXT records on every submitted domain. Third-party users of a service must not submit entries for it. |
| Contact address is personal | The `// Submitted by` line must use a role-based address belonging to the requesting organization, such as `security@example.com`. |
| Purpose is to work around third-party limits or protections | Requests whose objective is to bypass limits or protections of Cloudflare, Let's Encrypt, Apple, Meta, Google Safe Browsing, or similar parties must not be submitted. Address those limits directly with the third party. |
| Entry projects IP addresses into DNS | Names such as `1-2-3-4.example.com` resolving to `1.2.3.4` are not accepted. |
| TLD is not in the IANA root zone | `dig +short NS <tld>.` must return name servers. This project strictly adheres to ICP-3 namespace, so entries from alternative-root or other sources than ICANN, IANA or the IETF will be rejected. |
| Wildcard or exception syntax is invalid | `*` may appear only as the entire leftmost label. `!` may appear only at the start of an exception rule. |
| Entry is mislocated or unsorted | Place a new PRIVATE block in alphabetical order by organization name, never at the end of the file. Within a block, sort by TLD, then by each label to the left. Use Unicode, not Punycode, and no trailing whitespace. |
| Tests fail | Run `make test`. It must pass. |

After the pull request is opened, add a `_psl` TXT record containing the pull
request URL for every submitted suffix, and confirm it with
`dig +short TXT _psl.<suffix>`. Keep the record in place.

## Mandatory pull request template

All pull requests MUST use the repository's official pull request template
exactly as provided. Pull requests not using the template will be rejected.

Local path:

    .github/pull_request_template.md

Raw URL (alternative):

    https://raw.githubusercontent.com/publicsuffix/list/main/.github/pull_request_template.md

AI agents and automated contributors MUST:

- Preserve every heading, field, checkbox, comment, and section in the template.

- Complete the requested fields without renaming, deleting, reordering, or
  replacing them.

- Not submit a shortened, reformatted, summarized, or independently generated
  pull request description.

- Not remove sections because they appear inapplicable.

- Enter "Not applicable" where appropriate instead of deleting a section.

- Preserve all hidden HTML comments.

- List third-party limits this intends to affect, and delete the example
  Cloudflare, Let's Encrypt, and "REMOVE THIS LINE" lines in the third-party
  limits field.

- Check a box only when the statement is true.

- Treat modification of the template structure as a pull request validation
  failure.

A pull request whose description does not conform to the repository template
must not be submitted. Non-conforming pull requests that alter the repository
template will be closed.
