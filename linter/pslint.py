#!/usr/bin/env python3
# -*- coding: utf-8 -*-#
#
# PSL linter written in python
#
# Copyright 2016 Tim Rühsen (tim dot ruehsen at gmx dot de). All rights reserved.
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and associated documentation files (the "Software"),
# to deal in the Software without restriction, including without limitation
# the rights to use, copy, modify, merge, publish, distribute, sublicense,
# and/or sell copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
# FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
# DEALINGS IN THE SOFTWARE.

import sys
import codecs
import unicodedata

nline = 0
line = ""
orig_line = ""
warnings = 0
errors = 0
skip_order_check = False

def warning(msg):
	global warnings, orig_line, nline
	print('%d: warning: %s%s' % (nline, msg, ": \'" + orig_line + "\'" if orig_line else ""))
	warnings += 1

def error(msg):
	global errors, orig_line, nline
	print('%d: error: %s%s' % (nline, msg, ": \'" + orig_line + "\'" if orig_line else ""))
	errors += 1
#	skip_order_check = True

def print_psl(list):
	for domain in list:
		print(".".join(str(label) for label in reversed(domain)))

def psl_key(s):
	if s[0] == '*':
		return 0
	if s[0] == '!':
		return 1
	return 2

def check_order(group):
	"""Check the correct order of a domain group"""
	global skip_order_check

	try:
		if skip_order_check or len(group) < 2:
			skip_order_check = False
			return

		# check if the TLD is the identical within the group
		if any(group[0][0] != labels[0] for labels in group):
			warning('Domain group TLD is not consistent')

		# sort by # of labels, label-by-label (labels are in reversed order)
		sorted_group = sorted(group, key = lambda labels: (len(labels), psl_key(labels[-1][0]), labels))

		if group != sorted_group:
			warning('Incorrectly sorted group of domains')
			print("  " + str(group))
			print("  " + str(sorted_group))
			print("Correct sorting would be:")
			print_psl(sorted_group)

	finally:
		del group[:]


def lint_psl(infile):
	"""Parses PSL file and performs syntax checking"""
	global orig_line, nline

	PSL_FLAG_EXCEPTION = (1<<0)
	PSL_FLAG_WILDCARD = (1<<1)
	PSL_FLAG_ICANN = (1<<2) # entry of ICANN section
	PSL_FLAG_PRIVATE = (1<<3) # entry of PRIVATE section
	PSL_FLAG_PLAIN = (1<<4) #just used for PSL syntax checking

	line2number = {}
	line2flag = {}
	group = []
	section = 0
	icann_sections = 0
	private_sections = 0

	lines = [line.strip('\n') for line in infile]

	for line in lines:
		nline += 1

		# check for leading/trailing whitespace
		stripped = line.strip()
		if stripped != line:
			line = line.replace('\t','\\t')
			line = line.replace('\r','^M')
			orig_line = line
			warning('Leading/Trailing whitespace')
		orig_line = line
		line = stripped

		# empty line (end of sorted domain group)
		if not line:
			# check_order(group)
			continue

		# check for section begin/end
		if line[0:2] == "//":
			# check_order(group)

			if section == 0:
				if line == "// ===BEGIN ICANN DOMAINS===":
					section = PSL_FLAG_ICANN
					icann_sections += 1
				elif line == "// ===BEGIN PRIVATE DOMAINS===":
					section = PSL_FLAG_PRIVATE
					private_sections += 1
				elif line[3:11] == "===BEGIN":
					error('Unexpected begin of unknown section')
				elif line[3:9] == "===END":
					error('End of section without previous begin')
			elif section == PSL_FLAG_ICANN:
				if line == "// ===END ICANN DOMAINS===":
					section = 0
				elif line[3:11] == "===BEGIN":
					error('Unexpected begin of section: ')
				elif line[3:9] == "===END":
					error('Unexpected end of section')
			elif section == PSL_FLAG_PRIVATE:
				if line == "// ===END PRIVATE DOMAINS===":
					section = 0
				elif line[3:11] == "===BEGIN":
					error('Unexpected begin of section')
				elif line[3:9] == "===END":
					error('Unexpected end of section')

			continue # processing of comments ends here

		# No rule must be outside of a section
		if section == 0:
			error('Rule outside of section')

		group.append(list(reversed(line.split('.'))))

		# decode UTF-8 input into unicode, needed only for python 2.x
		try:
			if sys.version_info[0] < 3:
				line = line.decode('utf-8')
			else:
				line.encode('utf-8')
		except (UnicodeDecodeError, UnicodeEncodeError):
			orig_line = None
			error('Invalid UTF-8 character')
			continue

		# rules must be NFC coded (Unicode's Normal Form Kanonical Composition)
		if unicodedata.normalize("NFKC", line) != line:
			error('Rule must be NFKC')

		# each rule must be lowercase (or more exactly: not uppercase and not titlecase)
		if line != line.lower():
			error('Rule must be lowercase')

		# strip leading wildcards
		flags = section
		# while line[0:2] == '*.':
		if line[0:2] == '*.':
			flags |= PSL_FLAG_WILDCARD
			line = line[2:]

		if line[0] == '!':
			flags |= PSL_FLAG_EXCEPTION
			line = line[1:]
		else:
			flags |= PSL_FLAG_PLAIN

		# wildcard and exception must not combine
		if flags & PSL_FLAG_WILDCARD and flags & PSL_FLAG_EXCEPTION:
			error('Combination of wildcard and exception')
			continue

		labels = line.split('.')

		if flags & PSL_FLAG_EXCEPTION and len(labels) > 1:
			domain = ".".join(str(label) for label in labels[1:])
			if not domain in line2flag:
				error('Exception without previous wildcard')
			elif not line2flag[domain] & PSL_FLAG_WILDCARD:
				error('Exception without previous wildcard')

		for label in labels:
			if not label:
				error('Leading/trailing or multiple dot')
				continue

			if label[0:4] == 'xn--':
				error('Punycode found')
				continue

			if '--' in label:
				error('Double minus found')
				continue

			# allowed are a-z,0-9,- and unicode >= 128 (maybe that can be finetuned a bit !?)
			for c in label:
				if not c.isalnum() and c != '-' and ord(c) < 128:
					error('Illegal character')
					break

		if line in line2flag:
			'''Found existing entry:
			   Combination of exception and plain rule is contradictionary
			     !foo.bar + foo.bar
			   Doublette, since *.foo.bar implies foo.bar:
			      foo.bar + *.foo.bar
			   Allowed:
			     !foo.bar + *.foo.bar
			'''
			error('Found doublette/ambiguity (previous line was %d)' % line2number[line])

		line2number[line] = nline
		line2flag[line] = flags

	orig_line = None

	if section == PSL_FLAG_ICANN:
		error('ICANN section not closed')
	elif section == PSL_FLAG_PRIVATE:
		error('PRIVATE section not closed')

	if icann_sections < 1:
		warning('No ICANN section found')
	elif icann_sections > 1:
		warning('%d ICANN sections found' % icann_sections)

	if private_sections < 1:
		warning('No PRIVATE section found')
	elif private_sections > 1:
		warning('%d PRIVATE sections found' % private_sections)

def usage():
	"""Prints the usage"""
	print('usage: %s PSLfile' % sys.argv[0])
	print('or     %s -        # To read PSL from STDIN' % sys.argv[0])
	exit(1)


def main():
	"""Check syntax of a PSL file"""
	if len(sys.argv) < 2:
		usage()

	with sys.stdin if sys.argv[-1] == '-' else open(sys.argv[-1], 'r', encoding='utf-8', errors="surrogateescape") as infile:
		lint_psl(infile)

	return errors != 0


if __name__ == '__main__':
	sys.exit(main())
