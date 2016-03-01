require 'test_helper'
require 'public_suffix'
require 'simpleidn'

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


  it "passes the tests" do
    # Parse the PSL and run the tests
    defs = File.read(LIST_PATH)
    PublicSuffix::List.default = PublicSuffix::List.parse(defs)

    failures = []
    self.class.tests.each do |input, output|
      domain = begin
        # The PublicSuffix gem doesn't handle unicode/punycode conversion (yet)
        if input =~ /xn--/
          SimpleIDN.to_ascii(PublicSuffix.domain(SimpleIDN.to_unicode(input)))
        else
          PublicSuffix.domain(input)
        end
      rescue PublicSuffix::Error
        nil
      end

      failures << [input, output, domain] if output != domain
    end

    message = "The following #{failures.size} tests fail:\n"
    failures.each { |i,o,d| message += "Expected %s to be %s, got %s\n" % [i.inspect, o.inspect, d.inspect] }
    assert_equal 0, failures.size, message
  end

end
