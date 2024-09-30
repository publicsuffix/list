package parser

import "slices"

// Exceptions are parts of the PSL that would fail current validation
// and stylistic requirements, but are exempted due to predating those
// rules.
//
// See the bottom of this file for the exceptions themselves.

// exemptFromContactInfo reports whether the block owned by entity is
// exempt from the requirement to have a contact email address.
func exemptFromContactInfo(entity string) bool {
	return slices.Contains(missingEmail, entity)
}

// exemptFromSorting reports whether the block owned by entity is
// exempt from the sorting requirement that normally applies in the
// private domains section.
func exemptFromSorting(entity string) bool {
	return slices.Contains(incorrectSort, entity)
}

// missingEmail are source code blocks in the private domains section
// that are allowed to lack email contact information.
var missingEmail = []string{
	"611 blockchain domain name system",
	"c.la",
	"co.ca",
	"DynDNS.com",
	"Hashbang",
	"HostyHosting",
	"info.at",
	".KRD",
	"Michau Enterprises Limited",
	"Nicolaus Copernicus University in Torun - MSK TORMAN",
	"TASK geographical domains",
	"CoDNS B.V.",
	".pl domains (grandfathered)",
	"QA2",
}

// incorrectSort are entities in the private domains section that are
// allowed to be in the wrong sort order.
var incorrectSort = []string{
	"AAA workspace",
	"University of Banja Luka",
	"University of Bielsko-Biala regional domain",
	"No longer operated by CentralNic, these entries should be adopted and/or removed by current operators",
	"Africa.com Web Solutions Ltd",
	"iDOT Services Limited",
	"Radix FZC",
	"US REGISTRY LLC",
	"co.com Registry, LLC",
	"Roar Domains LLC",
	"BRS Media",
	"c.la",
	"Clever Cloud",
	"co.ca",
	"Co & Co",
	"i-registry s.r.o.",
	"CDN77.com",
	"Cloud DNS Ltd",
	"Daplie, Inc",
	"Datto, Inc.",
	"Bip",
	"bitbridge.net",
	"ddnss.de",
	"Definima",
	"DigitalOcean App Platform",
	"DigitalOcean Spaces",
	"DigitalPlat",
	"dnstrace.pro",
	"ECG Robotics, Inc",
	"Fedora",
	"Frusky MEDIA&PR",
	"RavPage",
	"CDDO",
	"GOV.UK Platform as a Service",
	"GOV.UK Pay",
	"Helio Networks",
	"Häkkinen.fi",
	"is-a.dev",
	"I-O DATA DEVICE, INC.",
	"KUROKU LTD",
	"Katholieke Universiteit Leuven",
	".KRD",
	"Lokalized",
	"May First - People Link",
	"mcpe.me",
	"NFSN, Inc.",
	"NFT.Storage",
	"No-IP.com",
	"NodeArt",
	"One.com",
	".pl domains (grandfathered)",
	"Pantheon Systems, Inc.",
	"PE Ulyanov Kirill Sergeevich",
	"Rad Web Hosting",
	"Raidboxes GmbH",
	"Redgate Software",
	"Redstar Consultants",
	"Russian Academy of Sciences",
	"QA2",
	"QCX",
	"QNAP System Inc",
	"Senseering GmbH",
	"Smallregistry by Promopixel SARL",
	"staticland",
	"Storebase",
	"Strapi",
	"Strategic System Consulting (eApps Hosting)",
	"Sony Interactive Entertainment LLC",
	"SourceLair PC",
	"SpaceKit",
	"SpeedPartner GmbH",
	"Spreadshop (sprd.net AG)",
	"Studenten Net Twente",
	"UNIVERSAL DOMAIN REGISTRY",
	".US",
	"VeryPositive SIA",
	"V.UA Domain Administrator",
}
