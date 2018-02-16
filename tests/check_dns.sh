#!/bin/bash
#
# check the changes in the psl for correct dns entries
# we will look at the domains nameservers if possible, to aviod wating until
# changed records are propagated

# constants
REPO="https://github.com/publicsuffix/list"
PULL_REQUEST="$TRAVIS_PULL_REQUEST"
TEMPFILE=$(mktemp)

# check for arguments
DEBUG=false
TEST=false
for arg in $*; do
    case $arg in
        -v)
            DEBUG=true
            ;;
        --test-file=*)
            TEST=true
            TEST_FILE=${arg/--test-file=/}
            ;;
        --test-pullrequest=*)
            TEST=true
            TEST_PULLREQUEST=${arg/--test-pullrequest=/}
            ;;
    esac
done

if ! $TEST; then
    # get only the lines added by pull request without comments
    git diff master public_suffix_list.dat \
        | sed -n '/^@@/,/^diff/{/^+/s/^+//p}' \
        | grep -v ^// \
        > $TEMPFILE
else
    # use testing data
    $DEBUG && echo TESTING MODE
    if [ -f "$TEST_FILE" ]; then
        cp "$TEST_FILE" $TEMPFILE
        if grep -qE '^[0-9]+$' <<< "$TEST_PULLREQUEST"; then  # if numeric
            PULL_REQUEST="$TEST_PULLREQUEST"
        else
            echo "ERROR: test-pullrequest missing or invalid."
            exit 2
        fi
    else
        echo "ERROR: test-file missing or not a file."
        exit 2
    fi
fi

# ask for pullrequest if not specified
# i.e. if user is checking if he has done everything right
if [ "$TRAVIS" != "true" ] && [ "x$PULL_REQUEST" == "x" ]; then
    echo -n "Pull request number not specified, please fill in: "
    read PULL_REQUEST
fi



# work on domains from $TEMPFILE
while read domain; do
    # kill empty lines
    if [ "x$domain" == "x" ]; then
        continue
    fi

    $DEBUG && echo "!! $domain"

    # try to find nameservers for domain
    # this speeds the process up a lot if there where any mistakes,
    # otherwise changes would have to propagate at first.
    domain_to_lookup="${domain/\*.}"
    while : ; do
        $DEBUG && echo "   checking nameservers for $domain_to_lookup"
        nameservers=$(dig +short NS "$domain_to_lookup")

        if [ "x$nameservers" != "x" ]; then
            break;
        fi

        domain_to_lookup=$(sed 's/^[^\.]*\.//' <<< "$domain_to_lookup")

        # break if domain_to_lookup is not a domain
        if ! grep -q '\.' <<< $domain_to_lookup; then
            break;
        fi
    done

    # or use default if none are there
    nameservers="${nameservers:-"8.8.8.8 4.4.4.4"}"
    $DEBUG && echo "   nameservers:" $nameservers

    # check domain against nameservers
    expected_result="\"${REPO}/pull/${PULL_REQUEST}\""
    $DEBUG && echo "   expected result: $expected_result"
    ok=false
    for nameserver in $nameservers; do
        dig_result=$(dig +short TXT "_psl.$domain_to_lookup" "$nameserver")
        $DEBUG && echo "   query result from $nameserver: ${dig_result:-NONE}"
        if [ "$dig_result" == "$expected_result" ]; then
            $DEBUG && echo "   -> OK!"
            ok=true
            break
        fi
    done

    # fail for issues
    if ! $ok; then
        echo "ERROR: $domain returned invalid TXT-record: ${dig_result:-NONE}"
        exit 1
    fi
done < $TEMPFILE

# cleanup
rm $TEMPFILE

# if we get to here, everything is fine
echo "OK: All new domains verified"
exit 0
