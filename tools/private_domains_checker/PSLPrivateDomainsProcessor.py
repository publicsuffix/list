import datetime
import json
import time

import pandas as pd
import requests
import whoisdomain as whois


def make_dns_request(domain, record_type):
    """
    Makes DNS requests to both Google and Cloudflare DNS APIs.

    Args:
        domain (str): The domain to query.
        record_type (str): The type of DNS record to query.

    Returns:
        list: A list containing the JSON responses from Google and Cloudflare.
    """
    urls = [
        f"https://dns.google/resolve?name={domain}&type={record_type}",
        f"https://cloudflare-dns.com/dns-query?name={domain}&type={record_type}"
    ]

    headers = {
        "accept": "application/dns-json"
    }

    responses = []
    for url in urls:
        try:
            response = requests.get(url, headers=headers)
            if response.status_code == 200:
                json_response = response.json()
                # print(f"URL: {url}, Response: {json_response}")
                responses.append(json_response)
            else:
                # print(f"URL: {url}, Status Code: {response.status_code}")
                responses.append(None)
        except Exception as e:
            print(f"URL: {url}, DNS Exception: {e}")
            responses.append(None)

    return responses


def check_dns_status(domain):
    """
    Checks the DNS status of a domain using Google and Cloudflare DNS APIs.

    Args:
        domain (str): The domain to check.

    Returns:
        str: The DNS status of the domain.
    """

    def make_request():
        responses = make_dns_request(domain, "NS")
        if None in responses:
            return "ERROR"

        google_status = responses[0].get("Status")
        cloudflare_status = responses[1].get("Status")

        print(f"Google Status: {google_status}, Cloudflare Status: {cloudflare_status}")

        if google_status == cloudflare_status:
            if google_status == 3:
                return "NXDOMAIN"
            else:
                return "ok"
        else:
            return "INCONSISTENT"

    for _ in range(5):
        dns_status = make_request()
        print(f"Attempt {_ + 1}, DNS Status: {dns_status}")
        if dns_status not in ["ERROR", "INCONSISTENT"]:
            return dns_status
        time.sleep(1)
    return "INCONSISTENT"


def check_psl_txt_record(domain):
    """
    Checks the _psl TXT record for a domain using Google and Cloudflare DNS APIs.

    Args:
        domain (str): The domain to check.

    Returns:
        str: The _psl TXT record status of the domain.
    """
    # Prepare the domain for the TXT check
    domain = domain.lstrip('*.').lstrip('!').encode('idna').decode('ascii')

    def make_request():
        responses = make_dns_request(f"_psl.{domain}", "TXT")
        if None in responses:
            return "ERROR"

        google_txt = responses[0].get("Answer", [])
        cloudflare_txt = responses[1].get("Answer", [])

        google_txt_records = [record.get("data", "") for record in google_txt]
        cloudflare_txt_records = [record.get("data", "").strip('"') for record in cloudflare_txt]

        print(
            f"_psl TXT Records (Google): {google_txt_records},  _psl TXT Records (Cloudflare): {cloudflare_txt_records}")

        if google_txt_records == cloudflare_txt_records:
            for record in google_txt_records:
                if "github.com/publicsuffix/list/pull/" in record:
                    return "valid"
            return "invalid"
        else:
            return "INCONSISTENT"

    for _ in range(5):
        psl_txt_status = make_request()
        print(f"Attempt {_ + 1}, PSL TXT Status: {psl_txt_status}")
        if psl_txt_status not in ["ERROR", "INCONSISTENT"]:
            return psl_txt_status
        time.sleep(1)
    return "INCONSISTENT"


def get_whois_data(domain):
    """
    Retrieves WHOIS data for a domain using the whoisdomain package.

    Args:
        domain (str): The domain to query.

    Returns:
        tuple: A tuple containing WHOIS domain status, expiry date, and WHOIS status.
    """
    try:
        d = whois.query(domain)
        whois_domain_status = d.statuses
        whois_expiry = d.expiration_date
        whois_status = "ok"
    except Exception as e:
        print(f"WHOIS Exception: {e}")
        whois_domain_status = None
        whois_expiry = None
        whois_status = "ERROR"
    return whois_domain_status, whois_expiry, whois_status


class PSLPrivateDomainsProcessor:
    """
    A class to process PSL private section domains, check their status, and save the results.
    """

    def __init__(self):
        """
        Initializes the PSLPrivateDomainsProcessor with default values and settings.
        """
        self.psl_url = "https://raw.githubusercontent.com/publicsuffix/list/master/public_suffix_list.dat"
        self.psl_icann_marker = "// ===BEGIN ICANN DOMAINS==="
        self.psl_private_marker = "// ===BEGIN PRIVATE DOMAINS==="
        self.columns = [
            "psl_entry",
            "top_level_domain",
            "dns_status",
            "whois_status",
            "whois_domain_expiry_date",
            "whois_domain_status",
            "psl_txt_status",
            "expiry_check_status"
        ]
        self.df = pd.DataFrame(columns=self.columns)
        self.icann_domains = set()

    def fetch_psl_data(self):
        """
        Fetches the PSL data from the specified URL.

        Returns:
            str: The raw PSL data.
        """
        print("Fetching PSL data from URL...")
        response = requests.get(self.psl_url)
        psl_data = response.text
        print("PSL data fetched.")
        return psl_data

    def parse_domain(self, domain):
        """
        Parses and normalizes a domain.

        Args:
            domain (str): The domain to parse.

        Returns:
            str: The normalized domain.

        Raises:
            ValueError: If no valid top-level domain is found.
        """
        domain = domain.lstrip('*.')  # wildcards (*)
        domain = domain.lstrip('!')  # bangs (!)

        parts = domain.split('.')

        for i in range(len(parts)):
            candidate = '.'.join(parts[i:])
            if candidate in self.icann_domains:
                continue
            elif '.'.join(parts[i + 1:]) in self.icann_domains:
                return candidate.encode('idna').decode('ascii')

        raise ValueError(f"No valid top-level domain found in the provided domain: {domain}")

    def parse_psl_data(self, psl_data):
        """
        Parses the fetched PSL data and separates ICANN and private domains.

        Args:
            psl_data (str): The raw PSL data.

        Returns:
            tuple: A tuple containing the unparsed private domains and the parsed private domains.
        """
        print("Parsing PSL data...")

        lines = psl_data.splitlines()
        process_icann = False
        process_private = False
        raw_private_domains = []
        parsed_private_domains = []

        for line in lines:
            stripped_line = line.strip()
            if stripped_line == self.psl_icann_marker:
                process_icann = True
                process_private = False
                continue
            elif stripped_line == self.psl_private_marker:
                process_icann = False
                process_private = True
                continue

            if stripped_line.startswith('//') or not stripped_line:
                continue

            if process_icann:
                self.icann_domains.add(stripped_line)
            elif process_private:
                raw_private_domains.append(stripped_line)
                parsed_private_domains.append(stripped_line)

        print(f"Private domains to be processed: {len(parsed_private_domains)}\n"
              f"ICANN domains: {len(self.icann_domains)}")

        parsed_private_domains = [self.parse_domain(domain) for domain in parsed_private_domains]
        raw_private_domains = list(set(raw_private_domains))
        parsed_private_domains = list(set(parsed_private_domains))
        print("Private domains in the publicly registrable name space: ", len(parsed_private_domains))

        return raw_private_domains, parsed_private_domains

    def process_domains(self, raw_domains, domains):
        """
        Processes each domain, performing DNS, WHOIS, and _psl TXT record checks.

        Args:
            raw_domains (list): A list of unparsed domains to process.
            domains (list): A list of domains to process.
        """
        data = []
        for raw_domain, domain in zip(raw_domains, domains):
            whois_domain_status, whois_expiry, whois_status = get_whois_data(domain)
            dns_status = check_dns_status(domain)
            psl_txt_status = check_psl_txt_record(raw_domain)

            if whois_status == "ERROR":
                expiry_check_status = "ERROR"
            else:
                expiry_check_status = "ok" if whois_expiry and whois_expiry >= (
                        datetime.datetime.utcnow() + datetime.timedelta(days=365 * 2)) else "FAIL_2Y"

            print(
                f"{domain} - DNS Status: {dns_status}, Expiry: {whois_expiry}, "
                f"PSL TXT Status: {psl_txt_status}, Expiry Check: {expiry_check_status}")

            data.append({
                "psl_entry": domain,
                "top_level_domain": domain,
                "whois_domain_status": json.dumps(whois_domain_status),
                "whois_domain_expiry_date": whois_expiry,
                "whois_status": whois_status,
                "dns_status": dns_status,
                "psl_txt_status": psl_txt_status,
                "expiry_check_status": expiry_check_status
            })

        self.df = pd.DataFrame(data, columns=self.columns)

    def save_results(self):
        """
        Saves all processed domain data to data/all.csv.
        """
        sorted_df = self.df.sort_values(by="psl_entry")
        sorted_df.to_csv("data/all.csv", index=False)

    def save_invalid_results(self):
        """
        Saves domains with invalid DNS or expired WHOIS data to data/nxdomain.csv and data/expired.csv.
        """
        nxdomain_df = self.df[self.df["dns_status"] != "ok"].sort_values(by="psl_entry")
        nxdomain_df.to_csv("data/nxdomain.csv", index=False)

        today_str = datetime.datetime.utcnow().strftime("%Y-%m-%d")
        expired_df = self.df[
            self.df["whois_domain_expiry_date"].notnull() &
            (self.df["whois_domain_expiry_date"].astype(str).str[:10] < today_str)
            ].sort_values(by="psl_entry")
        expired_df.to_csv("data/expired.csv", index=False)

    def save_hold_results(self):
        """
        Saves domains with WHOIS status containing any form of "hold" to data/hold.csv.
        """
        hold_df = self.df[
            self.df["whois_domain_status"].str.contains("hold", case=False, na=False)
        ].sort_values(by="psl_entry")
        hold_df.to_csv("data/hold.csv", index=False)

    def save_missing_psl_txt_results(self):
        """
        Saves domains with invalid _psl TXT records to data/missing_psl_txt.csv.
        """
        missing_psl_txt_df = self.df[self.df["psl_txt_status"] == "invalid"].sort_values(by="psl_entry")
        missing_psl_txt_df.to_csv("data/missing_psl_txt.csv", index=False)

    def save_expiry_less_than_2yrs_results(self):
        """
        Saves domains with WHOIS expiry date less than 2 years from now to data/expiry_less_than_2yrs.csv.
        """
        expiry_less_than_2yrs_df = self.df[self.df["expiry_check_status"] == "FAIL_2Y"].sort_values(by="psl_entry")
        expiry_less_than_2yrs_df.to_csv("data/expiry_less_than_2yrs.csv", index=False)

    def run(self):
        """
        Executes the entire processing pipeline.
        """
        psl_data = self.fetch_psl_data()
        raw_domains, domains = self.parse_psl_data(psl_data)
        self.process_domains(raw_domains, domains)
        self.save_results()
        self.save_invalid_results()
        self.save_hold_results()
        self.save_missing_psl_txt_results()
        self.save_expiry_less_than_2yrs_results()


if __name__ == "__main__":
    processor = PSLPrivateDomainsProcessor()
    processor.run()
