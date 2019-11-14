<!-- #### READ THIS FIRST ####

If you haven't yet, please read our guidelines:
https://github.com/publicsuffix/list/wiki/Guidelines#submit-the-change

If you'd like an example of what an excellent PR looks like
see https://github.com/publicsuffix/list/pull/615
-->

* [X] Description of Organization
* [X] Reason for PSL Inclusion
* [X] DNS verification via dig
* [X] Run Syntax Checker (make test)

<!--

As you complete each item in the checklist please mark it with an X

Example:

* [x] Description of Organization

-->

Description of Organization
====

Organization Website:
https://v.ua

<!--
Please tell us who you are and represent (i.e. individual, non-profit volunteer, engineer at a business)
and what you do (i.e. DynDNS, Hosting, etc)
-->

Reason for PSL Inclusion
====
The third level domains available to registration in the domain zone V.UA
<!--
Please tell us why your domain(s) should be listed in the PSL
(i.e. Cookie Security, Let's Encrypt issuance, etc).
-->

DNS Verification via dig
=======
```
dig +short TXT _psl.v.ua
"https://github.com/publicsuffix/list/pull/917"
```

<!--
For each domain you'd like to add to the list please create
a DNS verification record pointing to your pull request.

For example, if you'd like to add example.com and example.net
you would need to provide the following verifications:

Note that XXXX is replaced with the number of your pull request.
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
