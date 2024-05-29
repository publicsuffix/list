package parser

import (
	"strings"
)

// Exceptions are blocks of the PSL that would fail current validation
// and stylistic requirements, but are exempted due to predating those
// rules.
//
// These exceptions are deliberately built to be brittle: editing a
// block revokes its exemptions and requires the block to pass all
// modern validations (or the exceptions below need to be
// updated). This hopefully ratchets the PSL to always become more
// conformant with current policy, while not requiring that all
// existing lint be fixed immediately.
//
// See the bottom of this file for the exceptions themselves.

// downgradeToWarning reports whether e is a legacy exception to
// normal parsing and validation rules, and should be reported as a
// warning rather than a validation error.
func downgradeToWarning(e error) bool {
	switch v := e.(type) {
	case MissingEntityName:
		return sourceIsExempted(missingEntityName, v.Suffixes.Raw)
	case MissingEntityEmail:
		return sourceIsExempted(missingEmail, v.Suffixes.Raw)
	}
	return false
}

func sourceIsExempted(exceptions []string, source string) bool {
	for _, exc := range exceptions {
		if exc == source {
			return true
		}
	}
	return false
}

// dedent removes leading and trailing whitespace from all lines in
// s. It's just a helper to make multiline strings more readable when
// they don't need to include any indentation.
func dedent(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}

// missingEntityName are source code blocks that are allowed to lack
// an entity name.
var missingEntityName = []string{
	dedent(`// http://hoster.by/
	        of.by`),
}

// missingEmail are source code blocks in the private domains section
// that are allowed to lack email contact information.
var missingEmail = []string{
	dedent(`// 611coin : https://611project.org/
            611.to`),
	dedent(`// c.la : http://www.c.la/
            c.la`),
	dedent(`// co.ca : http://registry.co.ca/
            co.ca`),
	dedent(`// DynDNS.com : http://www.dyndns.com/services/dns/dyndns/
            dyndns.biz
            for-better.biz
            for-more.biz
            for-some.biz
            for-the.biz
            selfip.biz
            webhop.biz
            ftpaccess.cc
            game-server.cc
            myphotos.cc
            scrapping.cc
            blogdns.com
            cechire.com
            dnsalias.com
            dnsdojo.com
            doesntexist.com
            dontexist.com
            doomdns.com
            dyn-o-saur.com
            dynalias.com
            dyndns-at-home.com
            dyndns-at-work.com
            dyndns-blog.com
            dyndns-free.com
            dyndns-home.com
            dyndns-ip.com
            dyndns-mail.com
            dyndns-office.com
            dyndns-pics.com
            dyndns-remote.com
            dyndns-server.com
            dyndns-web.com
            dyndns-wiki.com
            dyndns-work.com
            est-a-la-maison.com
            est-a-la-masion.com
            est-le-patron.com
            est-mon-blogueur.com
            from-ak.com
            from-al.com
            from-ar.com
            from-ca.com
            from-ct.com
            from-dc.com
            from-de.com
            from-fl.com
            from-ga.com
            from-hi.com
            from-ia.com
            from-id.com
            from-il.com
            from-in.com
            from-ks.com
            from-ky.com
            from-ma.com
            from-md.com
            from-mi.com
            from-mn.com
            from-mo.com
            from-ms.com
            from-mt.com
            from-nc.com
            from-nd.com
            from-ne.com
            from-nh.com
            from-nj.com
            from-nm.com
            from-nv.com
            from-oh.com
            from-ok.com
            from-or.com
            from-pa.com
            from-pr.com
            from-ri.com
            from-sc.com
            from-sd.com
            from-tn.com
            from-tx.com
            from-ut.com
            from-va.com
            from-vt.com
            from-wa.com
            from-wi.com
            from-wv.com
            from-wy.com
            getmyip.com
            gotdns.com
            hobby-site.com
            homelinux.com
            homeunix.com
            iamallama.com
            is-a-anarchist.com
            is-a-blogger.com
            is-a-bookkeeper.com
            is-a-bulls-fan.com
            is-a-caterer.com
            is-a-chef.com
            is-a-conservative.com
            is-a-cpa.com
            is-a-cubicle-slave.com
            is-a-democrat.com
            is-a-designer.com
            is-a-doctor.com
            is-a-financialadvisor.com
            is-a-geek.com
            is-a-green.com
            is-a-guru.com
            is-a-hard-worker.com
            is-a-hunter.com
            is-a-landscaper.com
            is-a-lawyer.com
            is-a-liberal.com
            is-a-libertarian.com
            is-a-llama.com
            is-a-musician.com
            is-a-nascarfan.com
            is-a-nurse.com
            is-a-painter.com
            is-a-personaltrainer.com
            is-a-photographer.com
            is-a-player.com
            is-a-republican.com
            is-a-rockstar.com
            is-a-socialist.com
            is-a-student.com
            is-a-teacher.com
            is-a-techie.com
            is-a-therapist.com
            is-an-accountant.com
            is-an-actor.com
            is-an-actress.com
            is-an-anarchist.com
            is-an-artist.com
            is-an-engineer.com
            is-an-entertainer.com
            is-certified.com
            is-gone.com
            is-into-anime.com
            is-into-cars.com
            is-into-cartoons.com
            is-into-games.com
            is-leet.com
            is-not-certified.com
            is-slick.com
            is-uberleet.com
            is-with-theband.com
            isa-geek.com
            isa-hockeynut.com
            issmarterthanyou.com
            likes-pie.com
            likescandy.com
            neat-url.com
            saves-the-whales.com
            selfip.com
            sells-for-less.com
            sells-for-u.com
            servebbs.com
            simple-url.com
            space-to-rent.com
            teaches-yoga.com
            writesthisblog.com
            ath.cx
            fuettertdasnetz.de
            isteingeek.de
            istmein.de
            lebtimnetz.de
            leitungsen.de
            traeumtgerade.de
            barrel-of-knowledge.info
            barrell-of-knowledge.info
            dyndns.info
            for-our.info
            groks-the.info
            groks-this.info
            here-for-more.info
            knowsitall.info
            selfip.info
            webhop.info
            forgot.her.name
            forgot.his.name
            at-band-camp.net
            blogdns.net
            broke-it.net
            buyshouses.net
            dnsalias.net
            dnsdojo.net
            does-it.net
            dontexist.net
            dynalias.net
            dynathome.net
            endofinternet.net
            from-az.net
            from-co.net
            from-la.net
            from-ny.net
            gets-it.net
            ham-radio-op.net
            homeftp.net
            homeip.net
            homelinux.net
            homeunix.net
            in-the-band.net
            is-a-chef.net
            is-a-geek.net
            isa-geek.net
            kicks-ass.net
            office-on-the.net
            podzone.net
            scrapper-site.net
            selfip.net
            sells-it.net
            servebbs.net
            serveftp.net
            thruhere.net
            webhop.net
            merseine.nu
            mine.nu
            shacknet.nu
            blogdns.org
            blogsite.org
            boldlygoingnowhere.org
            dnsalias.org
            dnsdojo.org
            doesntexist.org
            dontexist.org
            doomdns.org
            dvrdns.org
            dynalias.org
            dyndns.org
            go.dyndns.org
            home.dyndns.org
            endofinternet.org
            endoftheinternet.org
            from-me.org
            game-host.org
            gotdns.org
            hobby-site.org
            homedns.org
            homeftp.org
            homelinux.org
            homeunix.org
            is-a-bruinsfan.org
            is-a-candidate.org
            is-a-celticsfan.org
            is-a-chef.org
            is-a-geek.org
            is-a-knight.org
            is-a-linux-user.org
            is-a-patsfan.org
            is-a-soxfan.org
            is-found.org
            is-lost.org
            is-saved.org
            is-very-bad.org
            is-very-evil.org
            is-very-good.org
            is-very-nice.org
            is-very-sweet.org
            isa-geek.org
            kicks-ass.org
            misconfused.org
            podzone.org
            readmyblog.org
            selfip.org
            sellsyourhome.org
            servebbs.org
            serveftp.org
            servegame.org
            stuff-4-sale.org
            webhop.org
            better-than.tv
            dyndns.tv
            on-the-web.tv
            worse-than.tv
            is-by.us
            land-4-sale.us
            stuff-4-sale.us
            dyndns.ws
            mypets.ws`),
	dedent(`// Hashbang : https://hashbang.sh
            hashbang.sh`),
	dedent(`// HostyHosting (hostyhosting.com)
            hostyhosting.io`),
	dedent(`// info.at : http://www.info.at/
            biz.at
            info.at`),
	dedent(`// .KRD : http://nic.krd/data/krd/Registration%20Policy.pdf
            co.krd
            edu.krd`),
	dedent(`// Michau Enterprises Limited : http://www.co.pl/
            co.pl`),
	dedent(`// Nicolaus Copernicus University in Torun - MSK TORMAN (https://www.man.torun.pl)
            torun.pl`),
	dedent(`// TASK geographical domains (www.task.gda.pl/uslugi/dns)
            gda.pl
            gdansk.pl
            gdynia.pl
            med.pl
            sopot.pl`),
	dedent(`// CoDNS B.V.
            co.nl
            co.no`),
	dedent(`// .pl domains (grandfathered)
            art.pl
            gliwice.pl
            krakow.pl
            poznan.pl
            wroc.pl
            zakopane.pl`),
	dedent(`// QA2
            // Submitted by Daniel Dent (https://www.danieldent.com/)
            qa2.com`),
}
