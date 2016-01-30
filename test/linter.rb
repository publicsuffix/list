module Linter
  extend self

  def blank_line?(line)
    line.strip == ""
  end

  def empty_line?(line)
    line == ""
  end

  def comment_line?(line)
    string = line.strip
    string.start_with?("//")
  end

  def suffix_line?(line)
    non_comment_line?(line) && non_blank_line?(line)
  end

  def non_blank_line?(line); !blank_line?(line); end
  def non_empty_line?(line); !empty_line?(line); end
  def non_comment_line?(line); !comment_line?(line); end
  def non_suffix_line?(line); !suffix_line?(line); end


  # If any line in lines has a leading space returns the index,
  # nil otherwise.
  def leading_space_index(lines)
    lines.find_index.each do |line|
      line.start_with?(" ")
    end
  end

  # If any suffix line is not lowercase returns the index,
  # nil otherwise.
  def suffix_non_lowercase_index(lines)
    lines.find_index.each do |line|
      non_comment_line?(line) && line =~ /[A-Z]/
    end
  end

  # If any suffix line has a leading dot returns the index,
  # nil otherwise.
  def suffix_with_leading_dot_index(lines)
    lines.find_index.each do |line|
      string = line.strip
      non_comment_line?(string) && string =~ /\A\./
    end
  end

  # If any suffix line has a space in the rule returns the index,
  # nil otherwise.
  def suffix_with_space_index(lines)
    lines.find_index.each do |line|
      string = line.strip
      non_comment_line?(string) && string =~ /\s/
    end
  end

end
