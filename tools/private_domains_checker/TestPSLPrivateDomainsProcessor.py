import unittest
import uuid

from PSLPrivateDomainsProcessor import PSLPrivateDomainsProcessor, check_dns_status, get_whois_data, check_psl_txt_record


class TestPSLPrivateDomainsProcessor(unittest.TestCase):

    def setUp(self):
        self.processor = PSLPrivateDomainsProcessor()
        # Populate icann_domains for testing
        self.processor.icann_domains = {
            "com", "co.uk", "ac.uk", "net", "org"
        }

    def test_parse_domain_icann_domain(self):
        # Test cases where domains should be parsed correctly
        test_cases = [
            ("*.example.com", "example.com"),
            ("sub.example.com", "example.com"),
            ("*.sub.example.com", "example.com"),
            ("example.com", "example.com"),
            ("example.co.uk", "example.co.uk"),
            ("sub.example.co.uk", "example.co.uk"),
            ("*.example.co.uk", "example.co.uk"),
            ("*.sub.example.co.uk", "example.co.uk"),
            ("abc.ac.uk", "abc.ac.uk"),
            ("a.b.com", "b.com")
        ]

        for domain, expected in test_cases:
            with self.subTest(domain=domain):
                result = self.processor.parse_domain(domain)
                self.assertEqual(expected, result)

    def test_parse_domain_no_icann(self):
        # Test case where no valid ICANN domain is found
        self.processor.icann_domains.remove("com")
        with self.assertRaises(ValueError):
            self.processor.parse_domain("example.com")

    def test_parse_domain_edge_cases(self):
        # Additional edge case testing
        self.assertEqual("example.org", self.processor.parse_domain("sub.example.org"))
        self.assertEqual("example.com", self.processor.parse_domain("example.com"))
        self.assertEqual("example.ac.uk", self.processor.parse_domain("sub.example.ac.uk"))

    def test_parse_domain_invalid(self):
        # Test invalid domains which should raise ValueError
        invalid_domains = ["invalid.test", "*.invalid.test", "sub.invalid.test"]
        for domain in invalid_domains:
            with self.subTest(domain=domain):
                with self.assertRaises(ValueError):
                    self.processor.parse_domain(domain)

    def test_check_dns_status(self):
        # Test with a known good domain
        self.assertEqual("ok", check_dns_status("mozilla.org"))
        # Test with a likely non-existent domain
        random_domain = "nxdomain-" + str(uuid.uuid4()) + ".edu"
        self.assertEqual("NXDOMAIN", check_dns_status(random_domain))

    def test_check_psl_txt_record(self):
        # Test with a known domain having a valid _psl TXT record
        self.assertEqual("valid", check_psl_txt_record("cdn.cloudflare.net"))
        # Test with a domain without a _psl TXT record
        random_domain = "invalid-" + str(uuid.uuid4()) + ".edu"
        self.assertEqual("invalid", check_psl_txt_record(random_domain))

    def test_get_whois_data(self):
        whois_data = get_whois_data("example.com")
        self.assertEqual("ok", whois_data[2])


if __name__ == "__main__":
    unittest.main()
