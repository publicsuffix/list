### The Public Suffix List Adheres to ICP-3 and IAB Guidance

The project does not accept requests for TLD additions that are not part of the one-root ICP-3 concept.

The PSL adheres to the [ICP-3](https://www.icann.org/resources/pages/unique-authoritative-root-2012-02-25-en) "A Unique, Authoritative Root for the DNS", and includes and enhances the [ICANN/IANA Public Suffix List](#icann-public-suffix-list).

There are exceptions to strict adherence to the ICP-3, but those are standards-driven or tied to authoritative policies that work within the ICP-3.

Some examples of exception are the .onion TLD (see [PR #386](https://github.com/publicsuffix/list/pull/386) and related [Issue #374](https://github.com/publicsuffix/list/issues/374) where there was IETF review resulting in RFC 7686 before allowance in PSL), 2012 round nTLDs that were approved TLD applications that have been furnished by ICANN on their contracted TLDs json file, and names defined by RFC(s) that are produced by the Internet Architecture Board (IAB) or Internet Engineering Task Force (IETF).

The project does not create new domain names, but rather documents existing names as defined by the IANA, ICANN, IETF or IAB, and then adds elegance and specificity within that universe without going beyond it.

### ICANN Security and Stability Advisory Committee (SSAC)

ICANN has a [Security and Stability Advisory Committee](https://www.icann.org/resources/pages/ssac-role-2018-02-06-en), with the role of advising the ICANN community and Board on matters relating to the security and integrity of the Internet's naming and address allocation systems.

They created a working group which convened 2014-2015 to review the use of Public Suffix Lists, and how to implement them within software, in security, and in language libraries and other systems. This ultimately resulted in SAC070, an advisory on the use of Public Suffix Lists.

Although it is slightly dated, and not current and entirely applicable to the Mozilla PSL, it does contain good practices and advise. The list maintainers have migrated management since here to GitHub, have introduced numerous validation and verification steps, and now have in place many of the suggestions and recommendations that were actionable as improvements.

Other suggestions and recommendations were made for those who integrate or use the PSL, and worth a read and consideration, as they were drawn from a pool of security experts that are familiar with architectural and procedural practices that merit some attention.

Please familiarize yourself with their findings and consider them in your use of these lists.

[SAC070 In Japanese / 日本語](https://www.icann.org/ja/system/files/files/sac-070-ja.pdf) [SAC070 In English](https://www.icann.org/en/system/files/files/sac-070-en.pdf)

### ICANN Public Suffix List

(Update May 2021) - ICANN Board of Directors held a special meeting (minutes) on May 12, 2021, where SAC070 advisories/recommendations were deemed satisfied. An outcome is that ICANN now operates a [Public Suffix List](https://data.iana.org/TLD/tlds-alpha-by-domain.txt).

[The ICANN Board of Directors May 12, 2021 Special Meeting Resolution 1.g](https://www.icann.org/resources/board-material/resolutions-2021-05-12-en#1.g) declared that the SAC070 advisory was closed and completed.

Specific attention should be drawn to recommendation 5, which called upon ICANN to publish a PSL. Because ICANN (via IANA) publishes a "Public Suffix List", which is the IANA TLD list (it doesn't descend below the first level), and this recommendation was found to be complete in the board resolution. The specific wording from the board resolution was, _"Whereas, on 1 December 2019, the IANA team started hosting an authoritative PSL for all TLDs in the root zone, completing Recommendation 5."_

### ICANN Office of the Chief Technology Officer (OCTO)

The ICANN OCTO team published a guide for TLD Administrators regarding this Public Suffix List, which is helpful reading. It includes actions that TLD Administrators should consider with respect to the Universal Acceptance and the impacts and benefits of monitoring the PSL and maintaining current records.

[OCTO-11, "The Public Suffix List: A Guide for TLD Administrators"](https://www.icann.org/en/system/files/files/octo-011-18may20-en.pdf)
