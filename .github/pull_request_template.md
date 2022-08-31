Public Suffix List (PSL) Pull Request (PR) Template
====

Each PSL PR needs to have a description, rationale, indication of DNS validation and syntax checking, as well as a number of acknowlegements from the submitter.  This template must be included with each PR, and the submitting party MUST provide responses to all of the elements in order to be considered.

<!-- #### READ THIS FIRST ####

If you haven't yet, please read our guidelines:
https://github.com/publicsuffix/list/wiki/Guidelines#submit-the-change

Also, read them again, as many skip that part and 
get confused about why their PR is delayed or does
not get accepted when theirs didn't follow them.

If you'd like an example of what an excellent PR looks like
see https://github.com/publicsuffix/list/pull/615
-->
### Checklist of required steps

* [ ] Description of Organization
* [ ] Reason for PSL Inclusion
* [ ] DNS verification via dig
* [ ] Run Syntax Checker (make test)

* [ ] Each domain listed in the PRIVATE section has and shall maintain at least two years remaining on registration, and we shall keep the \_PSL txt record in place in the respective zone(s) in the affected section

__Submitter affirms the following:__ 
<!--
Third-party Limits are used elsewhere, such as at Cloudflare, Let's 
Encrypt, Apple or others, and having an entry in the PSL alters 
the manner in which those third-party systems or products treat 
a given domain name or sub-domains within it.

To be clear, it is appropriate to address how those limits impact 
your domain(s) directly with that third-party, and it is inappropriate 
to submit entries to the PSL in order to work around those limits or 
restrictions.
-->
  * [ ] We are listing *any* third-party limits that we seek to work around in our rationale such as those between IOS 14.5+ and Facebook (see [Issue #1245](https://github.com/publicsuffix/list/issues/1245) as a well-documented example)

<!--
If there are third party limits that the PR seeks to overcome, those
must be listed within the rationale section of this request, and 
provide a good level of detail the effort that was made to work directly 
with the third part(y|ies) in attempting to address this within their 
rationale responsse below.
-->

  * [ ] This request was _not_ submitted with the objective of working around other third-party limits

<!--
The guidelines describe which section to place the entry, what the 
order of commented org placement, order of sorting of entries. 
(hint: TLD then SLD, Ascending sort)   Although it seems pedantic, 
the sorting and formatting rules help ensure all of the automation 
that uses the PSL operates correctly.  Typically both are solved or
neither.
-->

  * [ ] The [Guidelines](https://github.com/publicsuffix/list/wiki/Guidelines) were carefully _read_ and _understood_, and this request conforms
  * [ ] The submission follows the [guidelines](https://github.com/publicsuffix/list/wiki/Format) on formatting and sorting

<!-- 
Sorting and formatting of the entries is outlined in the guidelines 
and non-conforming requests are one of the largest sources of delay,
so getting this right initially will aid successfully having it 
proceed.  Miss-located entries and trailing spaces should be avoided.
-->

---

For Private section requests that are submitting entries for domains that match their organization website's primary domain, please understand that this can have impacts that may not match the desired outcome and take a long time to rollback, if at all.

To ensure that requested changes are entirely intentional, make sure that you read the affectation and propagation expectations, that you understand them, and confirm this understanding. 

PR Rollbacks have lower priority, and the volunteers are unable to control when or if browsers or other parties using the PSL will refresh or update.

<!-- 
Seriously, carefully read the downline flow of the PSL and the 
guidelines. Your request could very likely alter the cookie and 
certificate (as well as other) behaviours on your core domain name in 
ways that could be problematic for your business.

Rollback is really not predicatable, as those who use or incorporate 
the PSL do what they do, and when. It is not within the PSL volunteers' 
control to do anything about that.  

The volunteers are busy with new requests, and rollbacks are lowest 
priority, so if something gets broken by your PR, it will potentially 
stay that way for an indefinite period of time (typically long).
-->

(Link: [about propagation/expectations](https://github.com/publicsuffix/list/wiki/Guidelines#appropriate-expectations-on-derivative-propagation-use-or-inclusion))

 * [ ] *Yes, I understand*.  I could break my organization's website cookies etc. and the rollback timing, etc is acceptable.  *Proceed*.
---


<!--

As you complete each item in the checklist please mark it with an X

Example:

* [x] Description of Organization

-->

Description of Organization
====

<!--
PROVIDE AT LEAST THREE SENTENCES (the more the better) but
avoid the promotional stuff about how wonderful it is, and 
please do not copy and paste the mission statement or 
elevator pitch from your org's website.

Also tell us who you (submitter) are and represent (i.e. 
individual, non-profit volunteer, engineer at a business) 
and what you do (i.e. DynDNS, Hosting, etc), and what your 
role is as submitter with respect to the org and the 
submission.

For the org description, there is less interest in the 
promotional / marketing information about the org and more 
a focus on having concise description of the core focus of 
the submitting org, specifically with context/connection 
to this request.
-->

Organization Website: 
<!-- 
Provide the website address of 
the Org as a full URL ie https://example.com 
-->

Reason for PSL Inclusion
====

<!--
Please tell us why your domain(s) should be listed in the PSL
(i.e. Cookie Security, Let's Encrypt issuance, IOS/Facebook, 
Cloudflare etc) and clearly confirm that any private section 
names hold registration term longer than 2 years and shall 
maintain more than 1 year term in order to remain listed.

If you are attempting to work around third party limits, use 
this area to describe how and detail the mannner in which you 
have first attempted to engage those third parties on the 
matter.

Please also include the numbers of any past Issue # or PR # 
specifically related to this submission or section.

Three or more sentences here that describe the purpose for 
which your PR should be included in the PSL.  There is no 
upper limit, but six paragraphs seems like a rational stop.
-->

Number of users this request is being made to serve:
<!--
Identify if this is current or an estimate.
-->


DNS Verification via dig
=======

<!--
For each domain you'd like to add to the list please create
a DNS verification record pointing to your pull request.

For example, if you'd like to add example.com and example.net
you would need to provide the following verifications:

```
dig +short TXT _psl.example.com
"https://github.com/publicsuffix/list/pull/XXXX"
```

```
dig +short TXT _psl.example.net
"https://github.com/publicsuffix/list/pull/XXXX"
```

Note that XXXX is replaced with the number of your pull request.

We ask that you leave this record in place while you want 
your entry to remain in the PSL, so that future (TBD) 
automation can remove entries where the record is not present.

-->

Results of Syntax Checker (`make test`)
=========

<!--
Please verify that you followed the correct syntax and nothing broke

git clone https://github.com/publicsuffix/list.git
cd list
make test

Simply let us know that you ran the test and those results
-->


