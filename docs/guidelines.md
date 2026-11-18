## Introduction

In order to make the Public Suffix List as current and accurate as possible, the volunteers who maintain it request that TLD registries put in place processes to keep their section of the list current and accurate. This will make sure that all sites within their TLD continue to function fully within modern browsers, in terms of the site owners' ability to set cookies and so on.

In addition, owners of privately-registered domains who themselves issue subdomains to mutually-untrusting parties may wish to be added to the PRIVATE section of the list. Other projects referring their users to request PSL additions should refer to these guidelines and qualify their referral with the user prior to doing so.

The general procedure to make changes is as follows:

1. Carefully read about the [list format](https://publicsuffix.org/list/#list-format) to see how the list rules are defined.
2. [View the list](https://publicsuffix.org/list/), find your top-level domain (if present), and then check the rules already there for accuracy.
3. If there are any additions, removals or amendments to be made, create a pull request for our GitHub repo.

Updates should be filed as pull requests. Issues can be submitted; however, a pull request is generally preferred, and is more rapidly actionable. Pull requests are automatically checked by our test suite, and in many cases, you can get immediate feedback whether your submission contains some invalid rules that may cause the patch to be rejected.

Submissions are discussed either in the respective issue or pull request. Each submission requires validation, authentication and approval:

- **Validation**: the patch is validated to make sure the domains are acceptable and the patch follows the [list format](https://publicsuffix.org/list/).
- **Authentication**: the patch is authenticated to make sure the changes are legit. The exact authorization procedure depends on the type of changes (see below).
- **Approval**: the patch is manually approved and merged. Generally, if the patch is validated and authenticated, the approval is granted. However, in some cases we reject the request or encourage the submitter to alter the patch if we feel that the change will negatively impact the experience of the PSL consumers.

### Resourcing and Level of Effort (LoE), impacts the pace of processing, validation or merge of a contribution

This project has a number of contributors, most all of whom are volunteering their spare time to process requests and maintain this resource. To best process things in the most efficient manner, the following are some factors and dimensions that can impact the order and pace of processing by the volunteers.

**Important: There are NO SERVICE LEVEL AGREEMENTS ON TIME nor any expectation of processing speed or urgency. Being thorough, clear, and accurate with your request is the best path. And patient.**

#### What type of change is being requested?
- Additions (if complete and thorough) typically require lower LoE than Modifications/Deletions
- Modifications/Deletions are higher LoE

All addition scenarios are not equal, and there are factors that can cause things to take longer or not happen (see "Information furnished in the PR template").

Contributor frequency (new vs repeat) submitting party might indicate experience or familiarity that may reduce the LoE, or may be posting frequent changes indicating circumstances of higher LoE from closer review.

#### Information furnished in the PR template (BE THOROUGH)
Questions and clarifications, research or review may or may not be needed, based upon the completeness, clarity, and accuracy of the submission.

- Does the PR have the template filled out completely and clearly in a way that answers the common questions?
  - Why is the PR being requested? Please be detailed, but not so detailed as to lose the reviewer(s) in the weeds.
  - What are example domains' use scenarios? Describe the desired outcome. Enumerate through example subdomains or the name itself and how these are desired to work as a result of the PR being merged. This is particularly important with respect to wildcards (*) and bangs (!) and ensuring the result will align with the request.
- How many domains are being requested within the PR?
- Are the domains being requested for internal purposes (e.g. organization Foo wants their dev/staging servers, which only they use/access) or is this for general utility?
  - Basically, does every user shipping the PSL benefit from the added size/processing? (Sure, your opinion might be "YES", but provide support for this).
- Does every domain in the _PRIVATE_ section have at least two years remaining in the registration term?
  - Modifications take time to reach software that uses the PSL, and so it's important that domain registrations cover at least two years. In general, registrations should be done for a longer term. Note that registration periods of less than a year may result in the automatic removal of the domain.

Third-party and community submissions can be submitted; however, the submitting party is expected to make contact with the respective domain name administrator, as there is a validation step in the DNS which is mandatory. (see [How to Help vs "Help"](https://github.com/publicsuffix/list/issues/1439))

#### Validation of submission(s): (ALWAYS)
This has a range of LoE. These are not mutually exclusive validation techniques, and the bias is the DNS validation as it is difficult or impossible to forge. The other methods create a burden of research for the volunteers and severely impact the pace of processing or prioritization.

- On the lower side of LoE, Zone validation via DNS and _psl TXT [RFC8553](https://datatracker.ietf.org/doc/rfc8553)-style record matching the PR# will help prove that there is a connection between the authoritative admin or domain owner and the submitted PR.
- Documented, publicly available policies or enumeration / explanation of the submission from an authoritative source
- Was this submitted directly from the registry or domain owner? (if Registry, does this match the IANA database in some verifiable manner?)

The first one is the most crucial. The other two are helpful. Having all three makes things go VERY quickly in the ICANN section.

Additionally, certain PRs or TLDs will undergo out-of-band validation in addition to the DNS Validation where deemed appropriate by the volunteers. Examples of this might be where "Official" nexus edu.xx, police.xx, gov.xx or mil.xx (or linguistic equivalent xx= respective ccTLD) changes within the ICANN/IANA section or subdomain requests within .BANK, .INSURANCE, .PHARMACY or similar TLDs are very likely to be scrutinized carefully.

#### Request size and complexity can also impact the LoE and pace of processing:

- Is this a modification that moves entries in whole or in part from one section to another section?
- Is this a request that has a lot of individual entry domains to validate?
- Did the requestor complete a thorough response to the Pull Request Template, answering all checkboxes honestly?
  - Are there a number of clarifying questions that the contributor or the volunteers must pose?
  - Was/Were submitted domain(s) (PRIVATE SECTION) registration term holding an expiration date shorter than a year from the current date?
  - Was the sorting of the entry/ies formatted according to the documented [Format](format.md)?
- Is the contributor responsive to questions?

## Validation and Non-Acceptance Factors

Our non-acceptance (wontfix) criteria are as follows:

* We do not accept entries for use as DNS wildcards, such that e.g. 1-2-3-4.foo.tld resolves as IP address 1.2.3.4. This basically projects the security properties of the IP address space onto the domain name space, and we don't feel that is safe. IP addresses can be dynamically allocated to multiple mutually-untrusting parties; domain names generally are not.

* We do not accept entries whose sole purpose is to circumvent rate limits of third parties (such as Let's Encrypt rate limits - use their [form](https://letsencrypt.org/docs/rate-limits/) instead).

* We do not accept entries that have the objective of getting around limitations that have been put in place by a vendor to protect internet users. The PSL is not a 'workaround', and Pull Requests that appear to be doing this should expect to be declined. Be thorough and candid with the rationale furnished with the request.

* To avoid challenges, security issues, competing root systems or unintentional behavior associated with interoperability that may come from competing namespace authority, this project follows ICP-3 and I* (meaning, for example, .onion was defined by IETF RFC despite not being in the IANA root zone) standards for top-level domains (TLDs). We do not accept entries with TLDs that fall into 'Alternative' naming systems. ICANN's Office of the CTO has published [OCTO 034 Challenges with Alternative Name Systems](https://www.icann.org/en/system/files/files/octo-034-27apr22-en.pdf) which explains the variety of different systems and approaches, as well as the hazards. This 'nope' rule is here to allow discussion and energy that develop around this topic to happen elsewhere.

* The PSL is a globally used resource. PR submitters should understand that tens of millions, perhaps hundreds of millions of devices and uses may incorporate the change being requested, and need to consider if the request authentically merits such widespread inclusion. Expanding the file size even in small ways increases the overhead for everyone. File Size and scale of impact of a request is a consideration. Requests being minimal is important.

* Projects that are smaller in scale or are temporary or seasonal in nature will likely be declined. Examples of this might be private-use, sandbox, test, lab, beta, or other exploratory nature changes or requests. It should be expected that despite whatever site or service referred a requestor to seek addition of their domain(s) to the list, projects not serving more than thousands of users are quite likely to be declined.

* We now require that domains submitted as private section entries have expiration dates more than 2 years beyond the submitting date of a PR. Please include a statement within the rationale that a] the domain name(s) submitted have at least 2 years of registration period left in them at the time of submitting the name, and b] include a commitment that you will maintain their registration in good standing with more than a year left in their term.

* PRs citing as rationale that a company or project referred them to the PSL as a solution or as a quality review sieve or enablement means for some of their products' core feature(s) may experience long delays or non-acceptance on requests. Wherever possible, such projects should maintain their own solutions and not make such referrals.

* These non-acceptance criteria are subject to change, and the criteria listed here do not necessarily reflect the comprehensive circumstances or situations that will cause a PR not to be accepted. Accordingly, **this list may be modified or updated at any time**, and the volunteer maintainers reserve the right to decline PRs where there is a belief they have potential to introduce issues adverse to security, stability, resilience or other factors, and may do so at their discretion.

## Authentication

### ICANN Domains

These are Top Level Domains (or their direct, affiliated subordinates) that are **delegated by the IANA**, per [ICP3](https://www.icann.org/resources/pages/unique-authoritative-root-2012-02-25-en), or new Top Level Domains with a contracting announcement from ICANN (these are in a special section within the ICANN Domains).

Changes here need to either come from a representative of the registry (authenticated in a similar manner to below) or be from public sources such as a registry website. As a general rule, the processing of _psl TXT entries in DNS using the method described in the PR allows for more rapid processing, as it allows verification via DNS ([RFC8553](https://datatracker.ietf.org/doc/rfc8553)) by tying the PR to the authority for the zone in question.

Priority/expedited processing is given to changes provided through the PSL ICANN JSON Automation, followed by requests that are validated to originate from the respective IANA listed contact which are complete and can be verified via DNS validation against a public resolver for the affected zone(s).

### Private Domains

For private domains we need to confirm that the person submitting the request is an authorized representative of the domain owner. **We will not accept patches submitted by third-party users of the service**.

Historically, we have been validating the authentication of the request by using the data in the WHOIS response and sending an email to the domain owner. However, this approach doesn't scale very well. Here's the current validation list, in order of preference.

#### [RFC8553](https://datatracker.ietf.org/doc/rfc8553) DNS Authentication

For each suffix included in the patch, add a corresponding DNS TXT record called `_psl` that contains the link to the PR.

For example, if you submit the following patch:

```diff
+// Company : http://www.example.com
+// Submitted by John Doe <john@example.com>
+alphaexample.com
+foo.betaexample.com
+*.gammaexample.com
```

Then we expect to find the following DNS records (assuming the pull request ID is 100):

```
$ dig TXT _psl.alphaexample.com
https://github.com/publicsuffix/list/pull/100

$ dig TXT _psl.foo.betaexample.com
https://github.com/publicsuffix/list/pull/100

$ dig TXT _psl.gammaexample.com
https://github.com/publicsuffix/list/pull/100
```
This validation is sufficient to rapidly allow the volunteers to associate the administrative control of a resource to the party contributing the PR.

LEAVE THESE _psl IN PLACE WITHIN YOUR ZONES POST-VALIDATION IN ORDER TO ANNOUNCE CONTINUED INCLUSION IS DESIRED. At some point in the future, automation will be used to remove stale entries from the PSL, and missing _psl records will be a flag used in determination of domains to remove from the PSL.

#### Email Authentication

In some circumstances, we may proceed with the email-based validation. However, the email must be clearly associated (generally in the public WHOIS) with all the suffixes included in the pull request. If the pull request contains multiple suffixes and the email doesn't match all of them, then the entire pull request will be rejected.

For large companies we may require a mailing list such as `security@example.com` or administrative email, rather than user-specific emails.

## Submitting Amendments

Follow this procedure to submit a change to the list.

### Understand the list

Carefully read about the [list format](https://publicsuffix.org/list/#list-format) to see how the list rules are defined.

[View the list](https://publicsuffix.org/list/), find your top-level domain (if present), and then check the rules already there for accuracy. If you want to submit a private domain, go to the private domain section and find the appropriate position.

### Understand the requirements

Read this page to properly understand the validation/authentication/approval requirements before submitting your request.

### Prepare the change

Prepare your change. Here are some rules to keep in mind:

- Follow the repository conventions

- For an IANA domain, update the header only if you have more relevant references.

- For a new private domain section, keep the changeset sorted by Company/Product name within the `===BEGIN PRIVATE DOMAINS===` and `===END PRIVATE DOMAINS===` block.

  For example, if your company is called "Example Company" and you want to list the suffixes `foobar.example`, the position of the change will be determined by the alphabetical ordering of "Example Company" and not `foobar.example`.

  Start the change with a header that includes the company information and the submitter email. You must be in control of the submitter email, as we may use it for the authentication. The email must belong to the entity that requests the change (note that you can only modify a private domain section if you are an authorized representative of the company).

   ```
   // Example Company : https://example.com/
   // Submitted by John Doe <security@example.com>
   foo.example.com
   ```

### Sort your submission correctly (IMPORTANT)
IF YOU ARE MAKING A NEW ENTRY - DO NOT PUT YOUR ADDITIONS AT THE END OF THE FILE; THAT IS INVALID AND MAY GET IT REJECTED.
- Place your new subsection in the appropriate section of the file, sorted by commented company name.
- Keep the list of suffixes (if more than one) in alphabetical order. Sort first by the TLD, then the first label to the left of the TLD, and so forth. For example, the following is sorted by TLD first (`com`, `invalid`, `net`, `org`, `test`), then within each TLD, sorted by the first label (`example`), then, for entries with more labels, each label is sorted, with shorter entries appearing first:

   ```
   beta.example.com
   alpha.beta.example.com   // alpha.beta has more labels than beta, and thus comes second.
   delta.example.invalid    // invalid sorts between com and net
   beta.example.net
   delta.example.net        // delta sorts after beta, as example and net are the same
   charlie.example.org      // org sorts after net, because sorting is by the right-most label, not the left-most
   alpha.example.test
   ```

### Submit the change

[Create a pull request](https://help.github.com/articles/creating-a-pull-request/) with the requested changes.

It's recommended to clone and edit the change locally, as you will be able to test the changes before submitting the request. However, if you feel more comfortable, just use the GitHub in-browser editing feature to submit the pull request.

Please provide the following information:

- When submitting private domains, add a brief description of why you want the suffix(es) to be added to the list
- When submitting private domains, provide expected input/output to help us validate the correctness of the request
- When submitting IANA suffixes, provide the authoritative evidence supporting the change (if you work for the registry, please make that clear as well)
- Provide extra information useful for the PSL team to handle the pull request

### Test the changes

If you make the changes locally, you can run the test suite before submitting the pull request. Make the change to the list, save the changes and run the following make command:

```
$ make test
```

Check the output for errors.

The tests are also run automatically using Travis CI when you create the pull request.

Please pay attention to the results of the tests. If the tests don't pass, amend your patch according to the error messages.

Patches that don't pass the tests will be rejected.

### Follow up

After submission, please follow up to the ticket and any correspondence email regarding questions, until the ticket is closed. Also make sure that the email you listed in the pull request is valid, as we may use it for validation or private follow-ups.

### Expediting Propagation in Browsers (Etc) Once a Pull Request is Merged

**TL;DR: Unfortunately, there is no way to expedite.**

Project volunteers often have cycles that could be addressing other people's Pull Requests lost to answering questions or request emails related to how to nudge browsers or other integrators of the PSL to update their stuff. So often, in fact, that it was necessary to write this documentation section.

The PSL is just a list.

As a submitter, the good news is that you have done what you can within what control by submitting a PR that was mergeable, and hopefully did so without iterations with the reviewer team. Project volunteers have done what they can control within the review/merging process. Merging the pull request into the PSL and updating it to include changes is the edge of the horizon of control for the project volunteers.

The bad news is that there is no reach this project or PSL volunteers have into any derivative project roadmaps other than the update of the PSL itself being detected, if those projects do so.

More detail below.

### Appropriate Expectations on Derivative Propagation, Use or Inclusion

Once the PSL update includes a submission, it is important to note that the **PSL maintainers do not control the frequency of updates or inclusion of the change in the products or services that utilize the list**, how the list is used there, and most importantly **if** that derivative work will include the update (**or not**).

#### Derivative Propagation Timing

Changes to the PSL may take time to percolate to clients, [particularly those listed on the website](https://publicsuffix.org/learn/). Given that some platforms have incorporated the PSL within the OS itself, updates may not happen until the next OS update.

![The Cascade timing on when PSL changes trickle varies](https://raw.githubusercontent.com/publicsuffix/publicsuffix.org/master/images/Cascading1xDownstream.svg)
In the illustration above, A, B, C, and D are third parties. These might be a browser, DNS infrastructure, Operating System, or other system or library that pulls and incorporates the PSL from snapshots. The horizontal axis is time, but not to scale in minutes, hours, days or years.

In some circumstances, where the list is included or incorporated in some static manner by a library, product or service, one may see timing delays before seeing the change within those works. This can occur based upon schedules, product road maps and update cycles beyond the control or influence of the PSL maintainers / volunteers.

Please note, the same cascade delay circumstances can compound if the initial PR was submitted in a manner that requires revision (or rollback). The following illustration shows the impact of submitting PR 1 that had some unintended consequences that get adjustment in PR 2, once the changes are seen to have not behaved as expected. The same illustration works for a rollback situation, where the requestor decides to remove the changes from the first PR.
![Cascade Compounds the Timeline on changes where second request comes.](https://raw.githubusercontent.com/publicsuffix/publicsuffix.org/master/images/Cascading2xDownstream.svg)

#### Derivative Use

The PSL resource has no governance on how it is used, in whole, in part, or some modified version by derivative works. This includes how vendors treat the addition and/or deletion of entries, or how they process update information as the PSL gets updated.

#### Inclusion or Non-Inclusion in Derivative Works

The Public Suffix List is a list. It is entirely the decision and control of those who use or incorporate the list what they choose to do or not do. If there is a means to contact those parties who maintain or control the derivative works, they are the appropriate place to follow up. It is constructive to allow for reasonable delay for the propagation of changes before doing so, regardless of the sense of urgency.

### This is only a list

Keep in mind as you submit a PR that this PSL is just a list, and you are just adding or updating entries. Neither the PSL nor its volunteers prescribe what browsers do with entries or their recognition/handling of entries, default behaviors, or other handling. Browsers do what browsers do, diversely, and the PSL is not the boss of them.
