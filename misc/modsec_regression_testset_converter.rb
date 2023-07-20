#!/usr/bin/env ruby
# frozen_string_literal: true

require 'yaml'
require 'find'

max_paranoia_level = 2

crs_testcases = {
  'modsec-crs-rce' => 'REQUEST-932-APPLICATION-ATTACK-RCE',
  'modsec-crs-php' => 'REQUEST-933-APPLICATION-ATTACK-PHP',
  'modsec-crs-xss' => 'REQUEST-941-APPLICATION-ATTACK-XSS',
  'modsec-crs-lfi' => 'REQUEST-930-APPLICATION-ATTACK-LFI',
  'modsec-crs-rfi' => 'REQUEST-931-APPLICATION-ATTACK-RFI',
  'modsec-crs-sqli' => 'REQUEST-942-APPLICATION-ATTACK-SQLI',
  'modsec-crs-java' => 'REQUEST-944-APPLICATION-ATTACK-JAVA',
  'modsec-crs-generic' => 'REQUEST-934-APPLICATION-ATTACK-GENERIC',
  'modsec-crs-scanner-detection' => 'REQUEST-913-SCANNER-DETECTION'
}

def separate_negative_and_positive_tests(tests)
  result = { 'positive' => [], 'false-positive' => [] }

  tests.each do |t|
    if t['stages'][0]['stage']['output'].keys.include? 'no_log_contains' and t['stages'][0]['stage']['output']['no_log_contains'].match(/^id/i)
      result['false-positive'] << t
    else
      result['positive'] << t
    end
  end

  result
end

def get_rules_paranoia_level(test_set)
  result = {}

  File.read(".tmp/coreruleset/rules/#{test_set}.conf").gsub(/\\\n/, '').split("\n").each do |line|
    next unless line.match(/^[\s\t]*SecRule/i)

    id = line.match(/id:(\d+)/i)[1] if line.match(/id:(\d+)/i)
    paranoia = line.match(/paranoia-level\/(\d+)/i)[1] if line.match(/paranoia-level\/\d+/i)
    paranoia = line.match(/DETECTION_PARANOIA_LEVEL "@.*(\d+)"/i)[1] if line.match(/DETECTION_PARANOIA_LEVEL "@.*\d+"/i)
    next if paranoia.nil? || id.nil?

    result[id] = paranoia.to_i
  end

  result
end

def split_tests_by_req_method(tests)
  result = []
  test_title = tests['test_title']

  tests['stages'].each do |stage|
    case stage['stage']['input']['method']
    when nil
      result << parse_get_requests(stage['stage']['input'], test_title)
    when 'POST'
      result << parse_post_requests(stage['stage']['input'], test_title)
    else
      result << parse_get_requests(stage['stage']['input'], test_title)
    end
  end

  result.compact
end

def parse_get_requests(test_params, test_title)
  result = { 'method' => 'GET', 'payload' => '', 'title' => test_title }

  if test_params['uri'] && test_params['data']
    result['payload'] = "#{test_params['uri']}?#{test_params['data'].gsub(/\v/, '\\v')}".gsub(/^\//, '')
  elsif test_params['uri'].nil? && test_params['data']
    result['payload'] = "?#{test_params['data']}"
  elsif test_params['uri'] and test_params['data'].nil? and test_params['uri'].to_s.length > 16
    result['payload'] = test_params['uri'].gsub(/^\//, '')
  else
    return nil
  end

  result
end

def parse_post_requests(test_params, test_title)
  return nil if test_params['data'].nil?
  return nil if test_params['data'].length < 16

  result = { 'title' => test_title, 'method' => 'POST', 'payload' => test_params['data'].chomp }

  case test_params['headers']['Content-Type']
  when nil
    return result.merge({ 'placeholder' => 'RequestBody' })
  when /application\/x-www-form-urlencoded/i
    return result.merge({ 'placeholder' => 'RequestBody' })
  when /text\/plain/i
    return result.merge({ 'placeholder' => 'RequestBody' })
  when /application\/json/i
    return result.merge({ 'placeholder' => 'JSONRequest' })
  when /application\/xml/i
    return result.merge({ 'placeholder' => 'XMLBody' })
  when /multipart\/form-data/i
    payload = test_params['data'].gsub(/^---.*\n/, '').split("\n").last
    return result.merge({ 'placeholder' => 'HTMLMultipartForm', 'payload' => payload }) if payload.length > 15    
  end
  nil
end

def convert_and_save_get_testcases(testcases, gtw_testcase_name, test_set)
  result = []
  testcases.each { |t| split_tests_by_req_method(t).each { |x| result << x if x['method'] == 'GET' } }
  return if result.map { |x| x['payload'] }.empty?

  output_data = {
    'payload' => result.map { |x| x['payload'] }.uniq,
    'encoder' => ['Plain'],
    'placeholder' => ['URLPath'],
    'type' => gtw_testcase_name,
    'modsec_rule_name'=> test_set,
    'test_titles' => result.map { |x| x['title'] }.uniq.join(', ')
  }

  File.open("testcases/modsec-crs/#{gtw_testcase_name}.yml", 'w') { |file| file.write(output_data.to_yaml) }
end

def convert_and_save_post_testcases(testcases, gtw_testcase_name, test_set)
  result = []
  testcases.each { |t| split_tests_by_req_method(t).each { |x| result << x if x['method'] == 'POST' } }
  return if result.map { |x| x['payload'] }.empty?

  result.map { |x| x['placeholder'] }.uniq.sort.each do |placeholder|
    output_data = {
      'payload' => result.select { |x| x['placeholder'] == placeholder }.map { |x| x['payload'] }.uniq,
      'encoder' => ['Plain'],
      'placeholder' => [placeholder],
      'type' => gtw_testcase_name,
      'modsec_rule_name'=> test_set,
      'test_titles' => result.select { |x| x['placeholder'] == placeholder }.map { |x| x['title'] }.uniq.join(', ')
    }
    File.open("testcases/modsec-crs/#{gtw_testcase_name}_#{placeholder}.yml", 'w') { |file| file.write(output_data.to_yaml) }
  end
end

Dir.mkdir('testcases/modsec-crs') unless File.exist?('testcases/modsec-crs')
all_fp_testcases = []
crs_testcases.each do |gtw_testcase_name, test_set|
  current_test_set_data = []
  paranoia_level = get_rules_paranoia_level(test_set)

  Find.find(".tmp/coreruleset/tests/regression/tests/#{test_set}") do |file|
    next unless file.match(/.yaml$/)

    parsed_yaml_data = YAML.safe_load(File.read(file))
    next if parsed_yaml_data['meta']['enabled'] != true

    rule_id = parsed_yaml_data['meta']['name'].split('.')[0]
    next if paranoia_level[rule_id].nil? || paranoia_level[rule_id] >= max_paranoia_level
    parsed_yaml_data['tests'].each { |t| current_test_set_data << t }
  end

  current_separated_tests = separate_negative_and_positive_tests(current_test_set_data)
  convert_and_save_get_testcases(current_separated_tests['positive'], gtw_testcase_name, test_set)
  convert_and_save_post_testcases(current_separated_tests['positive'], gtw_testcase_name, test_set)
  current_separated_tests['false-positive'].each { |x| all_fp_testcases << x}
end

convert_and_save_get_testcases(all_fp_testcases, "fp_get_modsec_crs", 'fp_get_modsec_crs')
convert_and_save_post_testcases(all_fp_testcases, "fp_post_modsec_crs", 'fp_post_modsec_crs')
