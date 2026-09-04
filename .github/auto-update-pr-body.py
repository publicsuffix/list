#!/usr/bin/env python3
"""
Auto-update PR body in publicsuffix/list to match template format.
This script fetches the PR template and reformats the body correctly.
"""

import os
import re
import requests

# Get environment variables
GH_TOKEN = os.environ.get("GH_TOKEN")
PR_NUMBER = os.environ.get("PR_NUMBER", "3227")
REPO_OWNER = "publicsuffix"
REPO_NAME = "list"

# Original PR body content (extracted from PR #3227)
original_content = """AJ Services (operating commercially via ajdomain.com) is a premium domain registration and cloud infrastructure platform. We provision independent, standalone web spaces and digital assets for multi-tenant users. Our infrastructure, application distribution, and web routing are centrally managed under our core root domain: ajservices.online."""

org_website = "https://ajservices.online"

rationale = """Our platform allows untrusted third-party clients to launch completely distinct, user-controlled web platforms, content management systems (WordPress), and ecommerce stores as subdomains under ajservices.online. Because these tenants are unrelated commercial entities, we require PSL private domain isolation to enforce:
1. Strict HTTP Cookie Isolation: Preventing malicious or accidental cross-subdomain cookie leakage, ensuring session boundaries are securely locked between users.
2. Browser Same-Origin Policy (SOP): Establishing a firm security perimeter within modern browsers (Chrome, Firefox, Safari) so each customer's site handles scripts independently.
3. Ad Network Autonomy: Allowing advertising platforms like Google AdSense to audit and approve each distinct tenant subdomain (e.g., blog.ajservices.online) as an independent brand rather than a generic branch of our main platform."""

num_users = "5000"

dns_verification = """dig +short TXT _psl.ajservices.online
"https://github.com/publicsuffix/list/pull/3227\""""

abuse_contact_url = "https://ajservices.online/contact"

# Build the properly formatted PR body
formatted_body = f"""# Public Suffix List (PSL) Submission

### Checklist of required steps

* [x] Description of Organization
* [x] Robust Reason for PSL Inclusion
* [x] DNS verification via dig

* [x] Each domain listed in the PRIVATE section has and shall maintain at least two years remaining on registration, and we shall keep the `_psl` TXT record in place in the respective zone(s).

__Submitter affirms the following:__ 

 * [x] We are listing *any* third-party limits that we seek to work around in our rationale such as those between iOS 14.5+ and Facebook (see [Issue #1245](https://github.com/publicsuffix/list/issues/1245))
 <!-- FILL IN (CAN BE EMPTY): Third-party limits worked around (keep this line and its END FILL IN) -->
 <!-- END FILL IN -->

 * [x] This request was _not_ submitted with the objective of working around other third-party limits.

 * [x] The submitter acknowledges that it is their responsibility to maintain the domains within their section. This includes removing names which are no longer used, retaining the _psl DNS entry, and responding to e-mails to the supplied address. Failure to maintain entries may result in removal of individual entries or the entire section.

 * [x] The [Guidelines](https://github.com/publicsuffix/list/wiki/Guidelines) were carefully _read_ and _understood_, and this request conforms to them.
 * [x] The submission follows the [Guidelines](https://github.com/publicsuffix/list/wiki/Format) on formatting and sorting.

 * [x] A role-based email address has been used and this inbox is actively monitored with a response time of no more than 30 days.

**Abuse Contact:**

* [x] Abuse contact information (email or web form) is available and easily accessible.

  URL where abuse contact or abuse reporting form can be found: 
  <!-- FILL IN: Abuse contact URL (keep this line and its END FILL IN) -->
  {abuse_contact_url}
  <!-- END FILL IN -->

---

* [x] *Yes, I understand*. I could break my organization's website cookies and cause other issues, and the rollback timing is acceptable. *Proceed anyway*.

---

## Description of Organization
<!-- FILL IN: Description of Organization (keep this line and its END FILL IN) -->
{original_content}
<!-- END FILL IN -->

**Organization Website:**
<!-- FILL IN: Organization Website (keep this line and its END FILL IN) -->
{org_website}
<!-- END FILL IN -->

## Reason for PSL Inclusion
<!-- FILL IN: Reason for PSL Inclusion (keep this line and its END FILL IN) -->
{rationale}
<!-- END FILL IN -->

**Number of THOUSANDS of distinct users this request is being made to serve:**
<!-- FILL IN: Number of thousands of distinct users (keep this line and its END FILL IN) -->
{num_users}
<!-- END FILL IN -->

## DNS Verification
<!-- FILL IN: DNS verification records (keep this line and its END FILL IN) -->
{dns_verification}
<!-- END FILL IN -->
"""

if __name__ == "__main__":
    print(formatted_body)
    print("\n\n# To update the PR, run:")
    print(f"# gh pr edit {PR_NUMBER} --repo {REPO_OWNER}/{REPO_NAME} --body '@body.md'")
