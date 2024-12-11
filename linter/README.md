This directory contains a linter for the Public Suffix List.

Before you commit any changes to the PSL, please use the linter to check the syntax.

# Usage

Run this command in the root directory of the repository:

```
$ linter/pslint.py public_suffix_list.dat
```

> $? is set to 0 on success, else it is set to 1.


# Self Test

Every change on pslint.py should be followed by a self-test.

```
$ cd linter
$ ./pslint_selftest.sh
test_allowedchars: OK
test_dots: OK
test_duplicate: OK
test_exception: OK
test_punycode: OK
test_section1: OK
test_section2: OK
test_section3: OK
test_section4: OK
test_spaces: OK
test_wildcard: OK
```
