import click
import dns.message
import dns.name
import dns.query
import dns.resolver
import re


def read_rules(psl_filename):
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
    if rule.startswith("*."):
        rule = rule[2:]

    if rule.startswith("!"):
        rule = rule[1:]

    if any(illegal_char in rule for illegal_char in "!*"):
        print(rule)
        assert False

    return rule


def check_dns_pr(rule, pr_id):
    print(f"  Rule: {rule}")
    name = dns.name.from_text(rule)
    pslname = dns.name.from_text("_psl." + rule2fqdn(rule))
    print(f"    Checking TXT entry for {pslname}")
    try:
        answer = dns.resolver.resolve(pslname, "TXT")
    except dns.resolver.NoNameservers as e:
        print(f"    No nameserver found for {pslname}.")
        return False
    except dns.resolver.NXDOMAIN:
        print(f"    No _psl entry for {name}.")
        return False
    except dns.resolver.NoAnswer:
        print(f"    No answer from nameserver for {pslname}.")
        return False

    for rdata in answer:
        if match := re.match(r"\"https://github.com/publicsuffix/list/pull/(\d+)\"", str(rdata)):
            dns_pr_id = int(match[1])
            print(f"    DNS answer: {match[0]} -> PR {dns_pr_id}")
            if dns_pr_id == pr_id:
                return True
            else:
                print(f"    DNS _psl entry incorrect expected PR {pr_id} != {dns_pr_id}.")
                return False
    print("No DNS entry with pull request URL found.")
    return False


@click.command()
@click.argument('current_filename')
@click.argument('pull_request_filename')
@click.argument('pr_id', type=click.INT)
def main(current_filename, pull_request_filename, pr_id):
    current_rules = read_rules(current_filename)
    pull_request_rules = read_rules(pull_request_filename)

    all_good = True
    removed = current_rules.difference(pull_request_rules)
    print("The following rules have been removed:")
    for rule in removed:
        if not check_dns_pr(rule, pr_id): all_good = False

    added = pull_request_rules.difference(current_rules)

    print("The following rules have been added:")
    for rule in added:
        if not check_dns_pr(rule, pr_id): all_good = False

    if not all_good:
        exit(1)

if __name__ == "__main__":
    main()
