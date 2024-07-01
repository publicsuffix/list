# PSL Private Section Domains WHOIS Checker

## Overview

The `PSLPrivateDomainsProcessor` is a Python script designed to fetch data from the Public Suffix List (PSL) and check the domain status, expiry dates, and `_psl` TXT records of the private section domains. 

It performs WHOIS checks on these domains and saves the results into CSV files for manual review.

## Requirements

- Python 3.x
- `requests`
- `pandas`
- `whoisdomain`

You can install the required packages using pip:

```sh
pip install -r requirements.txt
```

Ensure that `whois` is installed on your operating system.

```sh
sudo apt install whois  # Debian/Ubuntu
sudo yum install whois  # Fedora/Centos/Rocky
```

## Usage

`PSLPrivateDomainsProcessor.py`: The main script containing the `PSLPrivateDomainsProcessor` class and functions for DNS and WHOIS checks.

Run the script using Python:

```sh
cd private_domains_checker
mkdir data
python PSLPrivateDomainsProcessor.py
```

## Main Components

### Functions

- `make_dns_request(domain, record_type)`: Makes DNS requests to both Google and Cloudflare DNS APIs.
- `check_dns_status(domain)`: Checks the DNS status of a domain using Google and Cloudflare DNS APIs.
- `get_whois_data(domain)`: Retrieves WHOIS data for a domain using the whoisdomain package.
- `check_psl_txt_record(domain)`: Checks the `_psl` TXT record for a domain using Google and Cloudflare DNS APIs.

### Class

#### PSLPrivateDomainsProcessor

- `fetch_psl_data()`: Fetches the PSL data from the specified URL.
- `parse_domain(domain)`: Parses and normalizes a domain.
- `parse_psl_data(psl_data)`: Parses the fetched PSL data and separates ICANN and private domains.
- `process_domains(raw_domains, domains)`: Processes each domain, performing DNS, WHOIS, and `_psl` TXT record checks.
- `save_results()`: Saves all processed domain data to `data/all.csv`.
- `save_invalid_results()`: Saves domains with invalid DNS or expired WHOIS data to `data/nxdomain.csv` and `data/expired.csv`.
- `save_hold_results()`: Saves domains with WHOIS status containing any form of "hold" to `data/hold.csv`.
- `save_missing_psl_txt_results()`: Saves domains with invalid `_psl` TXT records to `data/missing_psl_txt.csv`.
- `save_expiry_less_than_2yrs_results()`: Saves domains with WHOIS expiry date less than 2 years from now to `data/expiry_less_than_2yrs.csv`.
- `run()`: Executes the entire processing pipeline.

## Output

The script generates the following CSV files in the `data` directory:

- `all.csv`: Contains all processed domain data.
- `nxdomain.csv`: Contains domains that could not be resolved (`NXDOMAIN`).
- `expired.csv`: Contains domains with expired WHOIS records.
- `hold.csv`: Contains domains with WHOIS status indicating any kind of "hold".
- `missing_psl_txt.csv`: Contains domains with invalid `_psl` TXT records.
- `expiry_less_than_2yrs.csv`: Contains domains with WHOIS expiry date less than 2 years from now.

## Example

An example CSV entry:

| psl_entry      | top_level_domain | dns_status | whois_status | whois_domain_expiry_date | whois_domain_status          | psl_txt_status | expiry_check_status |
| -------------- | ---------------- | ---------- | ------------ | ----------------------- | ---------------------------- | -------------- | ------------------- |
| example.com    | example.com      | ok         | ok           | 2024-12-31              | "clientTransferProhibited"   | "valid"        | ok                  |

## Publicly Registrable Namespace Determination

The script determines the publicly registrable namespace from private domains by using the ICANN section. 

Here's how it works:

1. **ICANN Domains Set**: ICANN domains are stored in a set for quick lookup.
2. **Domain Parsing**: For each private domain, the script splits the domain into parts. It then checks if any suffix of these parts exists in the ICANN domains set.
3. **Normalization**: The private domain is normalized to its publicly registrable form using the ICANN domains set.

Examples:

- **Input**: PSL private domain entry `"*.example.com"`
  - **Process**: 
    - Remove leading `'*.'`: `"example.com"`
    - Check `"com"` against the ICANN domains set: Found
  - **Output**: `"example.com"`

- **Input**: PSL private domain entry `"sub.example.co.uk"`
  - **Process**:
    - Check `"example.co.uk"` against the ICANN domains set: Not found
    - Check `"co.uk"` against the ICANN domains set: Found
  - **Output**: `"example.co.uk"`

The output is then used for checking WHOIS data.

## License

This tool is licensed under the MIT License.