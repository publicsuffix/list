Reason of this Pull Request
====

<!--
If you haven't yet, please read our guidelines:
https://github.com/publicsuffix/list/wiki/Guidelines#submit-the-change

Please tell us what you do (i.e. DynDNS, Hosting, etc)
and why your domain(s) should be listed in the PSL
(i.e. Cookie Security, Let's Encrypt issuance, etc).

If you'd like an example of what an excellent PR looks like
see https://github.com/publicsuffix/list/pull/615
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
