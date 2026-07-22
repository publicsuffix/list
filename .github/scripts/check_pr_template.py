"""Validate that a pull request body still matches the PSL PR template.

The committed pull request template is the single source of truth. It marks the
areas a submitter completes with delimiter comments, which must be kept in place
(they are invisible on GitHub):

    <!-- FILL IN: <name> -->
    ...the submitter's content goes here...
    <!-- END FILL IN -->

A FILL IN block must be filled in; a FILL IN (CAN BE EMPTY) block — such as the
third-party-limits list — may be left empty. Everything outside the fill-in
regions and outside HTML comments is the fixed, visible template text.

The rules are:

  * fixed visible text preserved — every non-blank visible line outside the
    fill-in regions must still be present (a submitter may add text around it);
  * fill-in sections present — the FILL IN markers must be kept;
  * required fields filled in — a FILL IN block (but not one marked
    FILL IN (CAN BE EMPTY)) must contain content;
  * all boxes ticked — every checklist affirmation must be marked [x];
  * instructions removed — no "REMOVE THIS LINE" text may remain.

Fixed text is compared as rendered (whitespace-normalised, comments stripped),
so removing an instructional comment or reflowing whitespace is fine; only the
FILL IN markers themselves are required.

It reads the PR body from the PR_BODY environment variable, prints a single
pass/fail line (no itemised report), appends it to $GITHUB_STEP_SUMMARY, records
result=pass|fail in $GITHUB_OUTPUT, and exits non-zero when the body is invalid
so the workflow's job fails.
"""

import os
import re
import sys

# A submitter must delete every line carrying this token.
SENTINEL = "REMOVE THIS LINE"

COMMENT_RE = re.compile(r"<!--.*?-->", re.DOTALL)
CHECKBOX_RE = re.compile(r"^\s*[-*]\s*\[( |x|X)\]\s*(.*\S)\s*$")
CHECKBOX_STATE_RE = re.compile(r"\[[ xX]\]")
# The optional "(...)" after the name is a human note (e.g. "keep this line")
# and is ignored; the field name is everything up to it.
SLOT_RE = re.compile(
    r"<!--\s*FILL IN\s*(\(CAN BE EMPTY\))?\s*:\s*([^(]*?)\s*(?:\([^)]*\))?\s*-->"
    r"(.*?)<!--\s*END FILL IN\s*-->",
    re.DOTALL,
)


def strip_comments(text):
    return COMMENT_RE.sub("", text)


def normalize(text):
    """Collapse runs of whitespace so comparisons tolerate reflowing."""
    return re.sub(r"\s+", " ", text).strip()


def shorten(text, limit=70):
    text = normalize(text)
    return text if len(text) <= limit else text[: limit - 1] + "…"


def neutralize(line):
    """Normalise a line and blank out checkbox tick state for comparison."""
    return normalize(CHECKBOX_STATE_RE.sub("[ ]", line))


def visible_lines(text):
    """Non-blank rendered lines: comments removed, whitespace/ticks normalised."""
    text = strip_comments(text.replace("\r\n", "\n"))
    return [neutralize(line) for line in text.splitlines() if line.strip()]


def fill_in_fields(text):
    """Map each FILL IN section name to (raw content, required)."""
    return {
        match.group(2).strip(): (match.group(3), match.group(1) is None)
        for match in SLOT_RE.finditer(text)
    }


def fixed_lines(template):
    """Visible template lines outside the fill-in regions."""
    return visible_lines(SLOT_RE.sub("\n", template))


def required_checkboxes(template):
    """Checklist affirmation labels the submitter must tick."""
    labels = []
    for line in strip_comments(template).splitlines():
        match = CHECKBOX_RE.match(line)
        if match and match.group(1) == " ":
            labels.append(match.group(2))
    return labels


def checkbox_state(body, label):
    """True if ticked, False if present but unticked, None if absent."""
    target = normalize(label)
    state = None
    for line in strip_comments(body).splitlines():
        match = CHECKBOX_RE.match(line)
        if not match or normalize(match.group(2)) != target:
            continue
        if match.group(1) in ("x", "X"):
            return True
        state = False
    return state


def check(body, template):
    """Return a list of problems; an empty list means the body is valid."""
    problems = []
    body_lines = visible_lines(body)
    fields = fill_in_fields(body)

    # Fixed text is preserved when every template line outside the fill-in
    # regions still appears in the body. Matching is by substring, so a
    # submitter may add text around a fixed line as long as it stays intact.
    for line in fixed_lines(template):
        if not any(line in body_line for body_line in body_lines):
            problems.append(
                f"Required text is missing or was modified: “{shorten(line)}”"
            )

    # The FILL IN markers must stay, and a required section must have content.
    for name, (_, required) in fill_in_fields(template).items():
        if name not in fields:
            problems.append(f"Fill-in section is missing its markers: “{name}”")
        elif required and not visible_lines(fields[name][0]):
            problems.append(f"Fill-in field is empty: “{name}”")

    if any(SENTINEL in body_line for body_line in body_lines):
        problems.append(f"A “{SENTINEL}” instruction must be removed.")

    for label in required_checkboxes(template):
        if checkbox_state(body, label) is False:
            problems.append(f"Checklist item is not checked: “{shorten(label)}”")

    return problems


def render_report(problems):
    status = (
        "✅ PR template correctly filled out."
        if not problems
        else "❌ PR template incorrectly filled out."
    )
    return f"## PR template check\n\n{status}\n"


def main(argv=None):
    argv = sys.argv[1:] if argv is None else argv
    template_path = (
        argv[0]
        if argv
        else os.path.join(
            os.path.dirname(__file__), "..", "pull_request_template.md"
        )
    )
    with open(template_path, encoding="utf-8") as handle:
        template = handle.read()

    body = os.environ.get("PR_BODY", "") or ""
    problems = check(body, template)
    report = render_report(problems)
    print(report)

    summary_path = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary_path:
        with open(summary_path, "a", encoding="utf-8") as handle:
            handle.write(report)

    output_path = os.environ.get("GITHUB_OUTPUT")
    if output_path:
        with open(output_path, "a", encoding="utf-8") as handle:
            handle.write("result=" + ("fail" if problems else "pass") + "\n")

    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
