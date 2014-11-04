#!/usr/bin/env ruby
# Upload .deb packages to our apt repo

require 'socket'
require 'etc'
require 'tempfile'

class AptUpload
  UPLOAD_USER = 'apt-upload'
  UPLOAD_HOST = 'apt.ec2.shopify.com'

  VALID_ARCHES = %w(amd64)
  VALID_DISTS  = %w(lucid precise trusty utopic)

  def run(*args)
    usage if args.count != 1

    path = args.first
    Dir.entries(path).each do |file|
      upload_changes(File.join(path, file)) if file.end_with?('.changes')
    end
  end

  private

  def usage
    $stderr.puts "Usage: #{$0} <path>"
    exit(1)
  end

  def generate_upload_id
    timestamp = Time.now.utc.strftime("%Y%m%d_%H%M%S")
    username  = 'apt-builder'
    hostname  = Socket.gethostname.split('.', 2).first
    pid       = $$
    random    = rand(16**8)

    id = "%s_%s_%s_%d_%08x" % [timestamp, username, hostname, pid, random]
    raise "Upload ID contains odd characters? #{id.inspect}" if id =~ /[^A-Za-z0-9_-]/
    id
  end

  def upload_changes(file)
    upload_id    = generate_upload_id
    final_path   = "/data/uploads/pending/#{upload_id}"
    temp_path    = final_path + '.tmp'

    ssh_target   = UPLOAD_USER + '@' + UPLOAD_HOST
    rsync_target = ssh_target + ':' + temp_path + '/'

    Dir.mktmpdir(nil, File.dirname(File.dirname(file))) do |tempdir|
      files = process_changes(file, tempdir)
      File.chmod(0755, tempdir)

      puts
      puts "Uploading #{files.count} files to #{temp_path} on #{UPLOAD_HOST} ..."
      sh("rsync", "-dtP", "#{tempdir}/", rsync_target)

      puts
      puts "Queuing for upload ..."
      sh("ssh", ssh_target, "mv '#{temp_path}' '#{final_path}'")

      puts
      puts "Done uploading #{file}."
      sh("sudo", "touch", "#{File.dirname(file)}/upload.stamp")
    end
  end

  def sh(*command)
    begin
      system(*command)
    rescue Exception => e
      $stderr.puts "Error in command: #{command.inspect}"
      raise e
    end

    unless $?.success?
      raise "Command failed with status #{$?.inspect}: #{command.inspect}"
    end
  end

  def process_changes(in_file, tempdir)
    base_path = File.dirname(in_file)
    out_file  = File.join(tempdir, File.basename(in_file))
    files = [out_file]

    File.open(in_file) do |in_fh|
      File.open(out_file, 'w') do |out_fh|
        @announced = {}
        puts "Processing #{in_file}:"

        in_fh.each_line do |line|
          line.chomp!

          case line
          when /^Architecture: /
            line = $& + process_architectures($')
          when /^Distribution: /
            line = $& + process_distributions($')
          when /^\s [0-9a-f]{32,} \s \d+ (?:\s [a-z]+){0,2} \s/x
            file = $'
            if accept_file?(file)
              path = File.join(base_path, file)
              files << copy_file(path, tempdir)
            else
              line = nil
            end
          end

          out_fh.puts(line) unless line.nil?
        end
      end
    end

    files.compact.uniq
  end

  def process_architectures(arches)
    archlist = arches.split(/\s+/)

    if archlist.include?('source')
      raise "Source packages are not currently supported"
    end

    invalid = archlist - VALID_ARCHES
    raise "Invalid/unknown architecture(s): #{invalid.inspect}" unless invalid.empty?
    arches
  end

  def process_distributions(dists)
    distlist = dists.split(/\s+/)
    invalid  = distlist - VALID_DISTS
    raise "Invalid/unknown distribution(s): #{invalid.inspect}" unless invalid.empty?
    dists
  end

  def accept_file?(file)
    if file.end_with?('.deb')
      true
    elsif file.end_with?('.ddeb') # debug symbols
      false
    elsif file.end_with?('.dsc') || file.end_with?('.tar.gz')
      raise "Source packages are not currrently supported: #{file.inspect}"
    end
  end

  def copy_file(file, tempdir)
    target = File.join(tempdir, File.basename(file))
    return if File.exists?(target)

    puts "\t#{file}"
    FileUtils.copy(file, target)
    target
  end
end

AptUpload.new.run(*ARGV)
