Public Suffix List (PSL) Pull Request (PR) Template
====

Each PSL PR needs to have a description, rationale, indication of DNS validation and syntax checking, as well as a number of acknowledgements from the submitter. This template must be included with each PR, and the submitting party MUST provide responses to all of the elements in order to be considered.


### Checklist of required steps

* [x] Description of Organization
* [x] Robust Reason for PSL Inclusion
* [x] DNS verification via dig
* [x] Run Syntax Checker (make test)

* [x] Each domain listed in the PRIVATE section has and shall maintain at least two years remaining on registration, and we shall keep the \_PSL txt record in place in the respective zone(s) in the affected section

__Submitter affirms the following:__

  * [x] We are listing *any* third-party limits that we seek to work around in our rationale such as those between IOS 14.5+ and Facebook (see [Issue #1245](https://github.com/publicsuffix/list/issues/1245) as a well-documented example)

  * [x] This request was _not_ submitted with the objective of working around other third-party limits


  * [x] The [Guidelines](https://github.com/publicsuffix/list/wiki/Guidelines) were carefully _read_ and _understood_, and this request conforms
  * [x] The submission follows the [guidelines](https://github.com/publicsuffix/list/wiki/Format) on formatting and sorting


---

For Private section requests that are submitting entries for domains that match their organization website's primary domain, please understand that this can have impacts that may not match the desired outcome and take a long time to rollback, if at all.

To ensure that requested changes are entirely intentional, make sure that you read the affectation and propagation expectations, that you understand them, and confirm this understanding.

PR Rollbacks have lower priority, and the volunteers are unable to control when or if browsers or other parties using the PSL will refresh or update.


(Link: [about propagation/expectations](https://github.com/publicsuffix/list/wiki/Guidelines#appropriate-expectations-on-derivative-propagation-use-or-inclusion))

 * [x] *Yes, I understand*.  I could break my organization's website cookies etc. and the rollback timing, etc is acceptable.  *Proceed*.
---

Description of Organization
====


Organization Website:
https://tinohost.com

Reason for PSL Inclusion
====

We operate a number of public ingress points to our cloud to shard the customer websites evenly across multiple clusters we operate, every website is assigned a unique subdomain under a sharded ingress mask *.tino.page (e.g. some-website.staging.tino.page, the staging part is the wildcard).

At the moment, ~10k websites don't have a registered domain name assigned to them, so they are only available through their respective third-level subdomains of tino.page. We would like to isolate those websites from affecting each other and the parent domain by cookies, make CSP-bypass attacks a bit more difficult etc.

In order not to bloat the list and allow future scalability more easily, we are adding this as a single wildcard entry instead of the individual ingress subdomains.

Number of users this request is being made to serve: about 10000



DNS Verification via dig
=======

```
dig +short TXT _psl.tino.page
"https://github.com/publicsuffix/list/pull/1775"
```



Results of Syntax Checker (`make test`)
=========
