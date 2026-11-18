# Format

## Entry Specification

- The list is a set of rules, with one rule per line.
- Each line is only read up to the first whitespace; entire lines can also be commented using //.
- Each line which is not entirely whitespace or begins with a comment contains a rule.
- Each rule lists a public suffix, with the subdomain portions separated by dots (.) as usual. There is no leading dot.
- The wildcard character * (asterisk, specifically ```U+002A * 2a ASTERISK```) matches any valid sequence of characters in a hostname part. Wildcards in the PSL follow the syntax defined in [RFC1034](https://tools.ietf.org/html/rfc1034), section 4.3.3 (pp24-25), and are restricted to appear only in the leftmost position and must wildcard an entire label.
- If a hostname matches more than one rule in the file, the longest matching rule (the one with the most levels) will be used.
- An exclamation mark (!) at the start of a rule marks an exception to a previous wildcard rule. An exception rule takes priority over any other matching rule.
- The list uses **Unicode, not Punycode** forms on strings, and the **file is encoded using UTF-8**.
- **Entries should not have trailing whitespace**.

The following characters are used explicitly; **_please only use these and avoid Unicode variants of the following_**:

- Space `" "` (Dec/Hex: 32/20),
- Exclamation `"!"` (Dec/Hex: 33/21),
- Forward Slash `"/"` (Dec/Hex: 47/2F),
- Period/Dot `"."` (Dec/Hex: 46/2E), and
- Asterisk `"*"` (Dec/Hex: 42/2A).
- **When in doubt, for the above characters, only use the ASCII characters between Dec/Hex 32/20 and 126/7E**
- For header lines, material after the dual slashes is more forgiving as it is comment-based, but only use the standard `"@"` in email addresses, please.

**Examples of valid entries and ! or \* / wildcard usage:**
|Entry|Valid/Invalid|Why|
|-----|-----|-----|
|\*.foo|Valid|Correct Use of Wildcard|
|!specificsite.foo|Valid|Correct use of Exclamation to indicate exception to previous wildcard rule|
|\*.bar.foo|Valid|Correct Use of Wildcard|
|\*.예|Valid|Correct Use of Wildcard|
|\*.예.예|Valid|Correct Use of Wildcard|

**Examples of invalid entries and ! or \* / wildcard usage:**
|Entry|Valid/Invalid|Why|
|-----|-----|-----|
|\*.\*.bar.foo|Invalid|Multiple Wildcard|
|bar.*.foo|Invalid|Nested Wildcard (must be in leftmost position, only)|
|*bar.foo|Invalid|Wildcard Non-delimited by dot|
|예.\*.foo|Invalid|Nested Wildcard|
|ǃspecificsite.예.예|Invalid|Used latin letter retroflex click Unicode char (01C3) instead of exclamation (33)|

## Right-to-left sorting

Keep the list of suffixes (if more than one) in alphabetical order. Sort first by the TLD, then the first label to the left of the TLD, and so forth. For example, the following is sorted by TLD first (`com`, `invalid`, `net`, `org`, `test`), then within each TLD, sorted by the first label (`example`), then, for entries with more labels, each label is sorted, with shorter entries appearing first:

```
beta.example.com
alpha.beta.example.com   // alpha.beta has more labels than beta, and thus comes second.
delta.example.invalid    // invalid sorts between com and net
beta.example.net
delta.example.net        // delta sorts after beta, as example and net are the same
charlie.example.org      // org sorts after net, because sorting is by the right-most label, not the left-most
alpha.example.test
```

This is a simple Ruby script (called `psl-sort`) you can use to sort entries in a file:

```ruby
#!/usr/bin/env ruby

input  = STDIN.read.split("\n")
output = input.sort_by { |x| x.split(".").reverse }

puts output.join("\n")
```

and the same in Python:

```python
#!/usr/bin/env python3

import sys
lines = sys.stdin.readlines()
sorted_lines = sorted(lines, key=lambda x: list(reversed(x.split("."))))
for line in sorted_lines:
    print(line.strip())
```

Usage:

```
$ cat file.txt | psl-sort
```

## Example

Here is an example (incomplete) list section. The rules are numbered, but the numbers would not appear in the real file:

1. com
2. *.foo.com
3. *.jp
// Hosts in .hokkaido.jp can't set cookies below level 4...
4. *.hokkaido.jp
5. *.tokyo.jp
// ...except hosts in pref.hokkaido.jp, which can set cookies at level 3.
6. !pref.hokkaido.jp
7. !metro.tokyo.jp

The example above would be interpreted as follows, in the case of cookie-setting, and using "foo" and "bar" as generic hostnames:

1:

- Cookies **may** be set for foo.com.

(WARNING: Many libraries interpret `*.foo.com` as implying `foo.com`. So this would behave differently. See: https://github.com/publicsuffix/list/issues/694)

2:

- Cookies may not be set for bar.foo.com.
- Cookies **may** be set for example.bar.foo.com.

3:

- Cookies **may** be set for foo.bar.jp.
- Cookies **_may not_** be set for bar.jp.

4:

- Cookies **may** be set for foo.bar.hokkaido.jp.
- Cookies **_may not_** be set for bar.hokkaido.jp.

5:

- Cookies **may** be set for foo.bar.tokyo.jp.
- Cookies **_may not_** be set for bar.tokyo.jp.

6:

- Cookies **may** be set for pref.hokkaido.jp because the exception overrides the previous rule.

7:

- Cookies **may** be set for metro.tokyo.jp, because the exception overrides the previous rule.

## Formal algorithm

Here is an algorithm for determining the Public Suffix of a domain. (Note: it may not be the most efficient algorithm.) The domain, as well as all rules from the Public Suffix List, must be canonicalized in the normal way for hostnames - lowercase, Punycode (RFC 3492) - prior to being compared.

## Definitions

- The Public Suffix List consists of a series of lines, separated by \n.
- Each line is only read up to the first whitespace; entire lines can also be commented using //.
- Each line which is not entirely whitespace or begins with a comment contains a rule.
- A rule may begin with a "!" (exclamation mark). If it does, it is labelled as an "exception rule" and then treated as if the exclamation mark is not present.
- A domain or rule can be split into a list of labels using the separator "." (dot). The separator is not part of any of the labels. Empty labels are not permitted.
- A domain is said to match a rule if and only if all of the following conditions are met:
- When the domain and rule are split into corresponding labels, the domain contains as many labels as or more labels than the rule.
- Beginning with the right-most labels of both the domain and the rule, and continuing for all labels in the rule, one finds that for every pair, either they are identical or the label from the rule is "\*".

## Algorithm

- Match domain (See [Note 1](#note-1)) against all rules and take note of the matching ones.
- If no rules match, the prevailing rule is "\*". (See [Note 2](#note-2))
- If more than one rule matches, the prevailing rule is the one which is an exception rule.
- If there is no matching exception rule, the prevailing rule is the one with the most labels.
- If the prevailing rule is an exception rule, modify it by removing the leftmost label.
- The public suffix is the set of labels from the domain which match the labels of the prevailing rule, using the matching algorithm above. A trailing dot shall be added if the domain included one. (See [Note 3](#note-3))
- The registered or registrable domain is the public suffix plus one additional label.

##### Note 1
Querying the PSL with a domain which is not fully qualified cannot yield accurate results.

##### Note 2
There are different use-cases which require slightly different behavior. When the PSL is used to determine if a string is likely a domain, finding no matching rule is taken to mean that the input was not a valid domain.

##### Note 3
Callers of the algorithm often cannot determine if the domain to be queried is an FQDN but may treat `example.com` and `example.com.` as distinct. To support this, the algorithm should retain that distinction. This means looking up `example.com` should yield `com` while looking up `example.com.` should yield `com.` as the public suffix.

## Divisions

The Public Suffix List is subdivided, using markers in the comments, into two sections, labelled as `ICANN` domains and `PRIVATE` domains.

`ICANN` domains (_Perhaps IANA would have been a better label, but changing it may break integrations that are built_) are those delegated by ICANN or part of the IANA root zone database. The authorized registry may express further policies on how they operate the TLD, such as subdivisions within it. Updates to this section can be submitted by anyone, but if they are not an authorized representative of the registry then they will need to back up their claims of error with documentation from the registry's website or coordinate with the NIC Administrators to have them enter in `_psl` TXT record leaf validation.

`PRIVATE` domains are amendments submitted by the domain holder, as an expression of how they operate their domain security policy. Updates to this section are only accepted from authorized representatives of the domain registrant. This is so we can be certain they know what they are getting into.

While some applications, such as browsers when considering cookie-setting, treat all entries the same, other applications may wish to treat ICANN domains and PRIVATE domains differently. For example, Certification Authorities checking for wildcard misissuance would not issue a "\*.com" wildcard cert ("com" is in the ICANN domains list) but could legitimately issue a "\*.appspot.com" wildcard cert to the domain owner, in this case Google ("appspot.com" is in the PRIVATE domains list).
