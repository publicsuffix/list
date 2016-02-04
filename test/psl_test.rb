require 'test_helper'
require 'public_suffix'

# This test runs against the current PSL file and ensures
# the definitions satisfies the test suite.
describe "PSL" do

  def self.tests
    File.readlines(File.join(ROOT, "test/tests.txt")).map do |line|
      line = line.strip
      next if line.empty?
      next if line.start_with?("//")
      input, output = line.split(", ")

      # handle the case of eval("null"), it must be eval("nil")
      input  = "nil" if input  == "null"
      output = "nil" if output == "null"

      input  = eval(input)
      output = eval(output)
      [input, output]
    end
  end

  def self.define!
    defs = File.read(LIST_PATH)
    PublicSuffix::List.default = PublicSuffix::List.parse(defs)

    self.tests.each do |input, output|
      class_eval <<-RUBY, __FILE__, __LINE__+1
        it "xx" do
          domain = begin
            d = PublicSuffix.parse(#{input.inspect})
            [d.sld, d.tld].join(".")
          rescue
            nil
          end
          assert_equal(#{output.inspect}, domain, "Expected `%s` -> `%s`" % [#{input.inspect}, #{output.inspect}])
        end
      RUBY
    end
  end

  define!

end
