import re
import sys

def check_private_domains(file):
    with open(file) as f:
        before_private = True
        wrong_order_submissions = []
        content = f.readlines()
        incorrect_domains = {}
        for index, line in enumerate(content,0):
            if before_private:
                if line.startswith("// ===BEGIN PRIVATE DOMAINS==="):
                    before_private = False
                else:
                    continue
            if re.search("^\/\/ .* *: *https?:\/\/.*", line):
                wrong_domains, next_index = title_is_less(content, index)
                if wrong_domains:
                    wrong_order_submissions.append(wrong_domains)
                incorrect_domains[line.removeprefix("//").split(":")[0].strip()
                                  ] = check_submission_domains(content, index,
                                                                next_index)
    return wrong_order_submissions, incorrect_domains


def title_is_less(content, starting_index):
    domain = content[starting_index].removeprefix("// ").lower().strip()
    wrong_order = []
    end_index = -1
    for index, line in enumerate(content[starting_index+1:], starting_index+1):
        if re.search("^\/\/ .* *: *https?:\/\/.*", line):
            end_index = index
            next_domain = line.removeprefix("//").lower().strip()
            if domain > next_domain:
                wrong_order = (domain.strip(), next_domain.strip())
            return wrong_order, end_index
    return None, -1

            

def check_submission_domains(content, start_index, end_index):
    wrong_order_domains = []
    for index, line in enumerate(content[start_index:end_index], start_index):
        if line.startswith("//"):
                continue
        else:
            if not suffix_is_less(content[index].strip().split("."),
                                   content[index+1].strip().split(".")):
                wrong_order_domains.append((content[index].strip(), 
                                            content[index+1].strip()))
    return wrong_order_domains

def suffix_is_less(suffix_1, suffix_2):
    if not suffix_2[-1] or not [suffix_1][-1]:
        return True
    if suffix_1[-1] > suffix_2[-1]:
        return False
    elif suffix_1[-1] == suffix_2[-1]:
        if len(suffix_1) > 1:
            if len(suffix_2) <= 1:
                return False
            else:
                return suffix_is_less(suffix_1[:-1], suffix_2[:-1])
    return True

def main(effective_tld_filename):
    wrong_order_submissions, incorrect_domains = check_private_domains(
                                                    effective_tld_filename)
    if wrong_order_submissions:
        print(f"The following submissions were entered in the wrong order:")
        for submission in wrong_order_submissions:
            print(f"\t{submission}")
    if incorrect_domains:
        for submission in incorrect_domains:
            if incorrect_domains[submission]:
                print(f"The submission for {submission} contains domains in an improper order, which are:")
                for domain in incorrect_domains[submission]:
                    print(f"\t{domain}")



if __name__ == '__main__':
    main(sys.argv[1])