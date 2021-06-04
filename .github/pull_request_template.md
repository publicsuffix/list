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

Organization Website: https://adoveo.com

Adoveo is a digital marketing company which provide creation of interactive landing pages for high end brands in FMCG, Retail and charity. We need to become a part of the list to make it possible for our customers own their subdomains under adoveo.com.


Reason for PSL Inclusion
====

With iOS14.5 launch, in accordance with their AppTrackingTransparency framework, Facebook will start processing pixel conversion events from iOS 14 devices using Aggregated Event Measurement to preserve user privacy and help running effective campaigns. Specifically, Facebook would associate events triggered from a website to be associated with a Domain, the corresponding merchant's owned Facebook Ad Account, and the corresponding Pixel.

Our customers needs to verify their website's domain with Facebook and this would be done at eTLD+1 level. In our case, it would be boutir.com

However, as with our model, each merchant operates separately and independently. adoveo.com products, data, events are totally separated from those from bbb.adoveo.com. But now we cannot do it, as Facebook only verify eTLD+1 (i.e. adoveo.com) and would not verify and associate events with aaa.adoveo.com separately with bbb.adoveo.com. Our customers from aaa.adoveo.com and bbb.adoveo.com cannot pass through the verification step, and cannot effectively associate events triggered from their website to their domain.

If adoveo.com can be listed in the PSL, Facebook can separately verify aaa.adoveo.com and bbb.adoveo.com, and different customers can associate events and track separately with different domain aaa.adoveo.com and bbb.adoveo.com

DNS Verification via dig
=======

dig +short TXT _psl.example.net
https://github.com/publicsuffix/list/pull/XXXX


make test
=========

# TOTAL: 5
# PASS:  5
# SKIP:  0
# XFAIL: 0
# FAIL:  0
# XPASS: 0
# ERROR: 0


