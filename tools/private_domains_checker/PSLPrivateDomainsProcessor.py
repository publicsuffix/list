import datetime
import json
import os
import pandas as pd
import requests
import whois


def check_dns_status(domain):
    def make_request():
        try:
            url = f"https://dns.google/resolve?name={domain}&type=NS"
            response = requests.get(url)
            json_response = response.json()
            if json_response.get("Status") == 3:
                return "NXDOMAIN"
            else:
                return "ok"
        except Exception as e:
            return "ERROR"

    dns_status = make_request()
    if dns_status == "ERROR":  # Give it another try
        dns_status = make_request()
    return dns_status


def get_whois_data(domain):
    try:
        w = whois.whois(domain)
        whois_domain_status = w.status
        whois_expiry = w.expiration_date
        if isinstance(whois_expiry, list):
            whois_expiry = whois_expiry[0]
        whois_status = "ok"
    except Exception as e:
        whois_domain_status = None
        whois_expiry = None
        whois_status = "ERROR"
    return whois_domain_status, whois_expiry, whois_status


def check_psl_txt_record(domain):
    def make_request():
        try:
            url = f"https://dns.google/resolve?name=_psl.{domain}&type=TXT"
            response = requests.get(url)
            json_response = response.json()
            txt_records = json_response.get("Answer", [])
            for record in txt_records:
                if "github.com/publicsuffix/list/pull/" in record.get("data", ""):
                    return "valid"
            return "invalid"
        except Exception as e:
            return "ERROR"

    psl_txt_status = make_request()
    if psl_txt_status == "ERROR":  # Give it another try
        psl_txt_status = make_request()
    return psl_txt_status


class PSLPrivateDomainsProcessor:
    def __init__(self):
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
            "psl_txt_status"
        ]
        self.df = pd.DataFrame(columns=self.columns)
        self.icann_domains = set()

    def fetch_psl_data(self):
        print("Fetching PSL data from URL...")
        response = requests.get(self.psl_url)
        psl_data = response.text
        print("PSL data fetched.")
        return psl_data

    def parse_domain(self, domain):
        # Remove any leading '*.' parts
        domain = domain.lstrip('*.')

        # Split the domain into parts
        parts = domain.split('.')

        # Traverse the domain parts from the top-level domain upwards
        for i in range(len(parts)):
            candidate = '.'.join(parts[i:])
            if candidate in self.icann_domains:
                continue
            elif '.'.join(parts[i + 1:]) in self.icann_domains:
                # convert punycode to ASCII to support IDN domains
                return candidate.encode('idna').decode('ascii')

        # If no valid domain is found, raise an error
        raise ValueError(f"No valid top-level domain found in the provided domain: {domain}")

    def parse_psl_data(self, psl_data):
        print("Parsing PSL data...")

        lines = psl_data.splitlines()
        process_icann = False
        process_private = False
        private_domains = []

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
                private_domains.append(stripped_line)

        print(f"Private domains to be processed: {len(private_domains)}\n"
              f"ICANN domains: {len(self.icann_domains)}")

        # Parse each domain
        private_domains = [self.parse_domain(domain) for domain in private_domains]

        # Remove duplicates
        private_domains = list(set(private_domains))
        print("Private domains in the publicly registrable name space: ", len(private_domains))

        return private_domains

    def process_domains(self, domains):
        data = []
        for domain in domains:

            whois_domain_status, whois_expiry, whois_status = get_whois_data(domain)
            dns_status = check_dns_status(domain)
            psl_txt_status = check_psl_txt_record(domain)

            print(
                f"{domain} - DNS Status: {dns_status}, Expiry: {whois_expiry}, PSL TXT Status: {psl_txt_status}")

            data.append({
                "psl_entry": domain,
                "top_level_domain": domain,
                "whois_domain_status": json.dumps(whois_domain_status),
                "whois_domain_expiry_date": whois_expiry,
                "whois_status": whois_status,
                "dns_status": dns_status,
                "psl_txt_status": psl_txt_status
            })

        self.df = pd.DataFrame(data, columns=self.columns)

    def save_results(self):
        sorted_df = self.df.sort_values(by="psl_entry")
        sorted_df.to_csv("data/all.csv", index=False)

    def save_invalid_results(self):
        # Save nxdomain.csv
        nxdomain_df = self.df[self.df["dns_status"] != "ok"].sort_values(by="psl_entry")
        nxdomain_df.to_csv("data/nxdomain.csv", index=False)

        # Save expired.csv
        today_str = datetime.datetime.utcnow().strftime("%Y-%m-%d")
        expired_df = self.df[
            self.df["whois_domain_expiry_date"].notnull() &
            (self.df["whois_domain_expiry_date"].astype(str).str[:10] < today_str)
        ].sort_values(by="psl_entry")
        expired_df.to_csv("data/expired.csv", index=False)

        # Save missing_psl_txt.csv
        missing_psl_txt_df = self.df[self.df["psl_txt_status"] == "invalid"].sort_values(by="psl_entry")
        missing_psl_txt_df.to_csv("data/missing_psl_txt.csv", index=False)

    def save_hold_results(self):
        hold_df = self.df[
            self.df["whois_domain_status"].str.contains("hold", case=False, na=False)
        ].sort_values(by="psl_entry")
        hold_df.to_csv("data/hold.csv", index=False)

    def run(self):
        psl_data = self.fetch_psl_data()
        domains = self.parse_psl_data(psl_data)
        self.process_domains(domains)
        self.save_results()
        self.save_invalid_results()
        self.save_hold_results()


if __name__ == "__main__":
    processor = PSLPrivateDomainsProcessor()
    processor.run()
