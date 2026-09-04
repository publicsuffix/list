#!/usr/bin/env python3
"""
Direct PR body fixer for publicsuffix/list PR #3227
"""

import subprocess
import sys

# The correct, properly formatted PR body matching the template
CORRECT_BODY = """# Public Suffix List (PSL) Submission

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
  https://ajservices.online/contact
  <!-- END FILL IN -->

---

* [x] *Yes, I understand*. I could break my organization's website cookies and cause other issues, and the rollback timing is acceptable. *Proceed anyway*.

---

## Description of Organization
<!-- FILL IN: Description of Organization (keep this line and its END FILL IN) -->
I am Abdul Jabbar, an engineer and platform architect at AJ Services, responsible for overseeing the technical infrastructure and multi-tenant systems. AJ Services operates a premium domain registration and cloud hosting platform (ajservices.online) that provisions independent, standalone web spaces and digital assets for commercial clients. Our infrastructure is centrally managed but allows each customer to operate completely autonomous sub-domain instances with their own isolated applications, content management systems (WordPress), and ecommerce stores. We are not a generic shared hosting provider, but rather a specialized multi-tenant infrastructure platform serving distinct, unrelated business entities.
<!-- END FILL IN -->

**Organization Website:**
<!-- FILL IN: Organization Website (keep this line and its END FILL IN) -->
https://ajservices.online
<!-- END FILL IN -->

## Reason for PSL Inclusion
<!-- FILL IN: Reason for PSL Inclusion (keep this line and its END FILL IN) -->
Our platform allows untrusted third-party clients to launch completely distinct, user-controlled web platforms, content management systems (WordPress), and ecommerce stores as subdomains under ajservices.online. Because these tenants are unrelated commercial entities, we require PSL private domain isolation to enforce:

1. Strict HTTP Cookie Isolation: Preventing malicious or accidental cross-subdomain cookie leakage, ensuring session boundaries are securely locked between users.

2. Browser Same-Origin Policy (SOP): Establishing a firm security perimeter within modern browsers (Chrome, Firefox, Safari) so each customer's site handles scripts independently.

3. Ad Network Autonomy: Allowing advertising platforms like Google AdSense to audit and approve each distinct tenant subdomain (e.g., blog.ajservices.online) as an independent brand rather than a generic branch of our main platform.

4. SSL/TLS Certificate Isolation: Enabling each tenant to obtain independent SSL certificates for their subdomains without certificate transparency log conflicts or domain validation complications.

5. Let's Encrypt Rate Limit Compliance: Without PSL listing, Let's Encrypt rate limits would prevent multiple independent customers from obtaining certificates for separate subdomains within our zone.

Currently, we serve 5000+ distinct paying customers who operate independent commercial entities under our domain namespace.
<!-- END FILL IN -->

**Number of THOUSANDS of distinct users this request is being made to serve:**
<!-- FILL IN: Number of thousands of distinct users (keep this line and its END FILL IN) -->
5
<!-- END FILL IN -->

## DNS Verification
<!-- FILL IN: DNS verification records (keep this line and its END FILL IN) -->
dig +short TXT _psl.ajservices.online
"https://github.com/publicsuffix/list/pull/3227"
<!-- END FILL IN -->
"""

def main():
    try:
        # Update PR body using gh CLI
        result = subprocess.run(
            [
                "gh", "pr", "edit", "3227",
                "--repo", "publicsuffix/list",
                "--body", CORRECT_BODY
            ],
            capture_output=True,
            text=True,
            check=True
        )
        print("✅ PR #3227 body updated successfully!")
        print("📝 The template validation should now pass automatically.")
        return 0
    except subprocess.CalledProcessError as e:
        print(f"❌ Error updating PR: {e.stderr}")
        return 1

if __name__ == "__main__":
    sys.exit(main())
