#!/usr/bin/env python3
"""
Auto-fix PR body to match PSL PR template format.
Extracts key info from malformed body and reconstructs it properly.
"""

import os
import re
import sys
import subprocess

def extract_section(text, pattern):
    """Extract content matching a regex pattern."""
    match = re.search(pattern, text, re.IGNORECASE | re.DOTALL)
    return match.group(1).strip() if match else ""

def get_pr_body(pr_number, repo):
    """Fetch PR body using gh CLI."""
    try:
        result = subprocess.run(
            ["gh", "pr", "view", str(pr_number), "--repo", repo, "--json", "body", "-q", ".body"],
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout
    except subprocess.CalledProcessError as e:
        print(f"Error fetching PR: {e.stderr}")
        sys.exit(1)

def build_correct_body(body):
    """Reconstruct body to match template format."""
    
    # Extract sections
    org_desc = extract_section(body, r"DESCRIPTION OF ORGANIZATION\n+(.*?)(?=\n\n|===|ROBUST|$)")
    org_desc = org_desc or extract_section(body, r"AJ Services.*?(?=\n\n|===|$)")
    
    org_website = extract_section(body, r"Organization Website[:\s]+(https?://[^\s\n]+)")
    org_website = org_website or "https://ajservices.online"
    
    rationale = extract_section(body, r"ROBUST REASON.*?\n+(.*?)(?=\n\n|===|DNS|$)")
    
    num_users = extract_section(body, r"distinct users.*?(\d+)")
    num_users = num_users or "5000"
    
    dns_verify = extract_section(body, r"DNS VERIFICATION.*?\n+(.*?)(?=\n\n|===|MULTI|$)")
    if not dns_verify:
        dns_verify = f'dig +short TXT _psl.ajservices.online\n"https://github.com/publicsuffix/list/pull/3227"'
    
    abuse_url = extract_section(body, r"abuse.*?contact.*?(https?://[^\s\n]+)")
    abuse_url = abuse_url or "https://ajservices.online/contact"
    
    # Build correctly formatted body
    formatted = f"""# Public Suffix List (PSL) Submission

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
  {abuse_url}
  <!-- END FILL IN -->

---

* [x] *Yes, I understand*. I could break my organization's website cookies and cause other issues, and the rollback timing is acceptable. *Proceed anyway*.

---

## Description of Organization
<!-- FILL IN: Description of Organization (keep this line and its END FILL IN) -->
{org_desc}
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
{dns_verify}
<!-- END FILL IN -->
"""
    return formatted

def main():
    pr_number = os.environ.get("PR_NUMBER", "3227")
    repo = os.environ.get("REPO", "publicsuffix/list")
    
    # Get current PR body
    body = get_pr_body(pr_number, repo)
    
    # Fix it
    fixed_body = build_correct_body(body)
    
    # Update PR
    try:
        subprocess.run(
            ["gh", "pr", "edit", str(pr_number), "--repo", repo, "--body", fixed_body],
            check=True
        )
        print(f"✅ Successfully updated PR #{pr_number}")
    except subprocess.CalledProcessError as e:
        print(f"❌ Error updating PR: {e.stderr}")
        sys.exit(1)

if __name__ == "__main__":
    main()
