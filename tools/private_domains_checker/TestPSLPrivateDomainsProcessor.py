import unittest
import uuid
from unittest.mock import patch

from PSLPrivateDomainsProcessor import PSLPrivateDomainsProcessor, check_dns_status, get_whois_data


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
                self.assertEqual(result, expected)

    def test_parse_domain_no_icann(self):
        # Test case where no valid ICANN domain is found
        self.processor.icann_domains.remove("com")
        with self.assertRaises(ValueError):
            self.processor.parse_domain("example.com")

    def test_parse_domain_edge_cases(self):
        # Additional edge case testing
        self.assertEqual(self.processor.parse_domain("sub.example.org"), "example.org")
        self.assertEqual(self.processor.parse_domain("example.net"), "example.net")
        self.assertEqual(self.processor.parse_domain("sub.example.ac.uk"), "example.ac.uk")

    def test_parse_domain_invalid(self):
        # Test invalid domains which should raise ValueError
        invalid_domains = ["invalid.domain", "*.invalid.domain", "sub.invalid.domain"]
        for domain in invalid_domains:
            with self.subTest(domain=domain):
                with self.assertRaises(ValueError):
                    self.processor.parse_domain(domain)

    @patch('requests.get')
    def test_check_dns_status(self, mock_get):
        mock_response_ok = {
            "Status": 0,
            "Answer": [
                {"name": "example.com."}
            ]
        }
        mock_response_nxdomain = {
            "Status": 3
        }
        mock_get.side_effect = [
            MockResponse(mock_response_ok, 200),
            MockResponse(mock_response_nxdomain, 200)
        ]

        self.assertEqual(check_dns_status("example.com"), "ok")
        random_domain = "example" + str(uuid.uuid4()) + ".edu"
        self.assertEqual(check_dns_status(random_domain), "NXDOMAIN")

    def test_get_whois_data(self):
        whois_data = get_whois_data("example.com")
        self.assertEqual("ok", whois_data[2])


class MockResponse:
    def __init__(self, json_data, status_code):
        self.json_data = json_data
        self.status_code = status_code

    def json(self):
        return self.json_data


if __name__ == "__main__":
    unittest.main()
