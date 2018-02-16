#!/bin/bash

# test the create_dns.sh for expected output

cd $(dirname $0)
tests=$(grep -v '^#' <<EOF
# fromat:
# pullrequest number:filename:expected exitcode
600:everything_ok.txt:0
123:non_existing_domain.txt:1
123:domain_without_txt_record.txt:1
123:domain_with_incorrect_txt_record.txt:1
600:multiple_subdomains.txt:0
123:no_new_domains.txt:0
EOF
)

while IFS=':' read pullrequest file exitcode; do
    echo ../check_dns.sh --test-pullrequest=$pullrequest --test-file=$file
    ../check_dns.sh --test-pullrequest=$pullrequest --test-file=$file > /dev/null
    if [ "$?" -eq "$exitcode" ]; then
        echo "-> Test OK!"
    else
        echo Last test failed!
        exit 1
    fi
done <<< $tests
