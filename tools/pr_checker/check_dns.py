import click
import dns.message
import dns.name
import dns.query
import dns.resolver
import re


def read_rules(psl_filename):
    """Read rules from a file into a set.

    >>> sorted(read_rules('test_psl_01.dat'))
    ['!main.bar.example.org', '*.bar.example.org', '*.bd', 'a.example.com', 'at', 'b.example.com.', 'example.com', 'example.org', 'foo.example.org']
    """
    rules = set()

    with open(psl_filename) as f:
        for line in f:
            line = line.strip()

            if line == "":
                continue

            if line.startswith("//"):
                continue

            rules.add(line)
    return rules


def rule2fqdn(rule):
    """Return the domain name for a rule.

    Removes wildcards and exception qualifiers.

    >>> rule2fqdn("alpha.beta.example.com")
    'alpha.beta.example.com'
    >>> rule2fqdn("*.hokkaido.jp")
    'hokkaido.jp'
    >>> rule2fqdn("!pref.hokkaido.jp")
    'pref.hokkaido.jp'
    """

    if rule.startswith("*."):
        rule = rule[2:]

    if rule.startswith("!"):
        rule = rule[1:]

    if any(illegal_char in rule for illegal_char in "!*"):
        print(rule)
        assert False

    return rule


def check_dns_pr(rule, pr_id):
    """Check _psl DNS entry for a rule.

    >>> check_dns_pr("tests.arcane.engineering", 123456)
      Rule: tests.arcane.engineering
        Checking TXT entry for _psl.tests.arcane.engineering.
        DNS answer: "https://github.com/publicsuffix/list/pull/123456" -> PR 123456
    True
    >>> check_dns_pr("tests.arcane.engineering", 666)
      Rule: tests.arcane.engineering
        Checking TXT entry for _psl.tests.arcane.engineering.
        DNS answer: "https://github.com/publicsuffix/list/pull/123456" -> PR 123456
        DNS _psl entry incorrect expected PR 666 != 123456.
    False
    >>> check_dns_pr("foo.arcane.engineering", 666)
      Rule: foo.arcane.engineering
        Checking TXT entry for _psl.foo.arcane.engineering.
        No answer from nameserver for '_psl.foo.arcane.engineering.'.
    False
    """

    print(f"  Rule: {rule}")
    name = dns.name.from_text(rule)
    pslname = dns.name.from_text("_psl." + rule2fqdn(rule))
    print(f"    Checking TXT entry for {pslname}")
    try:
        # resolver = dns.resolver.Resolver()
        # resolver.nameservers = ["213.133.100.102"]
        # answer = resolver.resolve(pslname, "TXT")
        answer = dns.resolver.resolve(pslname, "TXT")
    except dns.resolver.NoNameservers as e:
        print(f"    No nameserver found for '{pslname}'.")
        return False
    except dns.resolver.NXDOMAIN:
        print(f"    No _psl entry for '{name}'.")
        return False
    except dns.resolver.NoAnswer:
        print(f"    No answer from nameserver for '{pslname}'.")
        return False

    for rdata in answer:
        if match := re.match(
            r"\"https://github.com/publicsuffix/list/pull/(\d+)\"", str(rdata)
        ):
            dns_pr_id = int(match[1])
            print(f"    DNS answer: {match[0]} -> PR {dns_pr_id}")
            if dns_pr_id == pr_id:
                return True
            else:
                print(
                    f"    DNS _psl entry incorrect expected PR {pr_id} != {dns_pr_id}."
                )
                return False
    print("No DNS entry with pull request URL found.")
    return False


def psl_diff(current_filename, pull_request_filename):
    """Check _psl DNS entry for a rule.

    >>> added, removed = psl_diff("test_psl_01.dat", "test_psl_02.dat")
    >>> sorted(added)
    ['be', 'com']
    >>> sorted(removed)
    ['*.bd', 'a.example.com', 'b.example.com.']
    """
    current_rules = read_rules(current_filename)
    pull_request_rules = read_rules(pull_request_filename)

    removed = current_rules.difference(pull_request_rules)
    added = pull_request_rules.difference(current_rules)

    return (added, removed)


@click.command()
@click.argument("current_filename")
@click.argument("pull_request_filename")
@click.argument("pr_id", type=click.INT)
def main(current_filename, pull_request_filename, pr_id):
    """This script compares two PSL files and checks the _psl DNS records for the changed rules.

    It can be tested using doctests by running `python -m doctest check_dns.py`
    """
    added, removed = psl_diff(current_filename, pull_request_filename)

    if not all(map(lambda rule: check_dns_pr(rule, pr_id), added.union(removed))):
        exit(1)


if __name__ == "__main__":
    main()
