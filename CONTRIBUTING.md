# Submitting Amendments

Before submitting any change to the list, please make sure to read the [Guidelines](https://github.com/publicsuffix/list/wiki/Guidelines).

A properly formatted and validated patch will decrease the review time, and increase the chances your request will be reviewed and perhaps accepted. Any patch that doesn't follow the Guidelines will be rejected or, in the best scenario, left pending for follow-up.

The most common time loss comes from not following the sorting guidelines
- Sorting / Placement needs to comply with [Guidelines](https://github.com/publicsuffix/list/wiki/Guidelines)
  - *PLEASE* order within the existing entries in the PRIVATE DOMAINS section so that your listed organization on your first comment line is alphabetically sorted
  - Do NOT append your PRIVATE DOMAINS entry to end of the file
  - If there are more than one domain within your PR, order your entries alphabetically, ascending by TLD, then SLD, then 3LD and deeper etc (if present)

Other Common mistakes that may cause the request to be rejected include:

- Invalid patch formatting, rule sorting or changeset position (see this: [Wiki:Formatting](https://github.com/publicsuffix/list/wiki/Format))
- Missing validation records 
- Lack of proper domain ownership or expiry dates less than 2Y away
- Attempts to work around vendor limits (see [#1245](https://github.com/publicsuffix/list/issues/1245) as an example)
- Submissions with TLDs non-compliant with [ICP-3](https://www.icann.org/resources/pages/unique-authoritative-root-2012-02-25-en) or on the [ICANN PSL](https://github.com/publicsuffix/list/wiki/Security-Considerations#icann-public-suffix-list)
- Insufficient or incomplete rationale (be verbose!)
- Smaller, private projects with <2000 stakeholders

Frequently, PR submissions overlook the sort ordering guidelines, adding to delay in processing and an increase in the time it takes to process requests.

Make sure to review with the [Guidelines](https://github.com/publicsuffix/list/wiki/Guidelines) before you open a new pull request.

Please also note that there is no guarantee of inclusion, nor we are able to provide an ETA for any inclusion request.  This is also true of projects that incorporate the PSL downline.  This is described, outlined and diagrammed [here](
https://github.com/publicsuffix/list/wiki/Guidelines#appropriate-expectations-on-derivative-propagation-use-or-inclusion).
