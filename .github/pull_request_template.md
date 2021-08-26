<!-- #### READ THIS FIRST ####

If you haven't yet, please read our guidelines:
https://github.com/publicsuffix/list/wiki/Guidelines#submit-the-change

If you'd like an example of what an excellent PR looks like
see https://github.com/publicsuffix/list/pull/615
-->

* [ ] Description of Organization
* [ ] Reason for PSL Inclusion
* [ ] DNS verification via dig
* [ ] Run Syntax Checker (make test)

* [ ] Each domain listed in the PRIVATE section has and shall maintain at least two years remaining on registration, and we shall keep the _PSL txt record in place


__Submitter affirms the following:__ 
  * [ ] We are listing any third party limits that we seek to work around in our rationale such as those between IOS 14.5+ and Facebook (see [Issue #1245](https://github.com/publicsuffix/list/issues/1245) as a well-documented example)
  * [ ] This request was _not_ submitted with the objective of working around other third party limits
  * [ ] The [Guidelines](https://github.com/publicsuffix/list/wiki/Guidelines) were carefully _read_ and _understood_, and this request conforms
  * [ ] The submission follows the [guidelines](https://github.com/publicsuffix/list/wiki/Format) on formatting

---
__For Private section requests that are submitting entries for domains that match their organization website's primary domain:__

``` 
Seriously, carefully read the downline flow of the PSL and the guidelines.
Your request could very likely alter the cookie and certificate (as well as other) behaviours on your 
core domain name in ways that could be problematic for your business.

Rollback is really not predicatable, as those who use or incorporate the PSL do what they do, and when.
It is not within the PSL volunteers' control to do anything about that.  

The volunteers are busy with new requests, and rollbacks are lowest priority, so if something gets broken 
it will stay that way for an indefinitely long while.
```
(Link: [about propogation/expectations](https://github.com/publicsuffix/list/wiki/Guidelines#appropriate-expectations-on-derivative-propagation-use-or-inclusion))

 * [ ] Yes, I understand.  I could break my organization's website cookies etc. and the rollback timing, etc is acceptable.  Proceed.
---


<!--

As you complete each item in the checklist please mark it with an X

Example:

* [x] Description of Organization

-->

Description of Organization
====

Organization Website: <!-- https://example.com -->

<!--
Please tell us who you are and represent (i.e. individual, 
non-profit volunteer, engineer at a business) and what you 
do (i.e. DynDNS, Hosting, etc)

For the org description, there is less interest in the 
promotional / marketing information about the org and more 
a focus on having concise description of the core focus of 
the submitting org, specifically with context/connection 
to this request.
-->

Reason for PSL Inclusion
====

<!--
Please tell us why your domain(s) should be listed in the PSL
(i.e. Cookie Security, Let's Encrypt issuance, IOS/Facebook, 
Cloudflare etc) and clearly confirm that any private section 
names hold registration term longer than 2 years and shall 
maintain more than 1 year term in order to remain listed.

Please also include the numbers of any past Issue # or PR # 
specifically related to this submission or section.
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

make test
=========

<!--
Please verify that you followed the correct syntax and nothing broke

git clone https://github.com/publicsuffix/list.git
cd list
make test

Simply let us know that you ran the test
-->


