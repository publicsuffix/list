import re

def check_private_domains(file):
    with open(file) as f:
        before_private = True
        prev_domain = ""
        wrong_order_domains = []
        for index, line in enumerate(f,1):
            if before_private:
                if line.startswith("// ===BEGIN PRIVATE DOMAINS==="):
                    before_private = False
                else:
                    continue
            if re.search("^\/\/ .* *: *https:\/\/.*", line):
                domain = line.removeprefix("// ").lower().strip()
                if prev_domain and domain < prev_domain:
                    wrong_order_domains.append((prev_domain, domain))
                wrong_domains = check_submission_domains(file, index)
                print(domain + ": " + str(wrong_domains)) if wrong_domains is not None else print("fine")
                prev_domain = domain
    return wrong_order_domains
            

def check_submission_domains(file, index):
    submission_started = False
    wrong_order_domains = []
    for line in file[index:]:
        if line.startswith("//") and submission_started:
            return wrong_order_domains
        else:
            submission_started = True
            if not suffix_is_less(file[index].strip(), file[index+1].strip()):
                wrong_order_domains.append((file[index], file[index+1]))
    return

def suffix_is_less(suffix_1, suffix_2):
    if suffix_1[-1] > suffix_2[-1]:
        return False
    elif suffix_1[-1] == suffix_2[-1]:
        if len(suffix_1) > 1:
            if len(suffix_2) <= 1:
                return False
            else:
                return suffix_is_less(suffix_1[:-1], suffix_2[:-1])
    return True
        

def psl_is_ordered(psl):
    for i in range(len(psl)-1):
        if not suffix_is_less(psl[i], psl[i+1]):
            print(psl[i], psl[i+1])


def main():
    # suffix_list = construct_psl("public_suffix_list.dat")
    # psl_is_ordered(suffix_list)
    for item in check_private_domains("public_suffix_list.dat"):
        print(item)



if __name__ == '__main__':
    main()