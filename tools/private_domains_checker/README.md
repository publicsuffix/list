# PSL Private Section Domains WHOIS Checker

## Overview

The `PSLPrivateDomainsProcessor` is a Python script designed to fetch data from the Public Suffix List (PSL) and check the domain status and expiry dates of the private section domains. 

It performs WHOIS checks on these domains and saves the results into CSV files for manual review.

## Requirements

- Python 3.x
- `requests`
- `pandas`
- `python-whois`

You can install the required packages using pip:

```sh
pip install -r requirements.txt
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

- `check_dns_status(domain)`: Checks the DNS status of a domain using Google's DNS API. It attempts to recheck DNS status if the initial check fails.
- `get_whois_data(domain)`: Retrieves WHOIS data for a domain. Note: WHOIS data parsing handles multiple expiration dates by selecting the first one.

### Class

#### PSLPrivateDomainsProcessor

- `fetch_psl_data()`: Fetches the PSL data from the specified URL.
- `parse_domain(domain)`: Parses and normalizes a domain.
- `parse_psl_data(psl_data)`: Parses the fetched PSL data and separates ICANN and private domains.
- `process_domains(domains)`: Processes each domain, performing DNS and WHOIS checks.
- `save_results()`: Saves all processed domain data to `data/all.csv`.
- `save_invalid_results()`: Saves domains with invalid DNS or expired WHOIS data to `data/nxdomain.csv` and `data/expired.csv`.
- `run()`: Executes the entire processing pipeline.

## Output

The script generates the following CSV files in the `data` directory:

- `all.csv`: Contains all processed domain data.
- `nxdomain.csv`: Contains domains that could not be resolved (`NXDOMAIN`).
- `expired.csv`: Contains domains with expired WHOIS records.

## Example

An example CSV entry:

| psl_entry      | top_level_domain | dns_status | whois_status | whois_domain_expiry_date | whois_domain_status |
| -------------- | ---------------- | ---------- | ------------ | ----------------------- | ------------------- |
| example.com    | example.com      | ok         | ok           | 2024-12-31              | "clientTransferProhibited" |

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
