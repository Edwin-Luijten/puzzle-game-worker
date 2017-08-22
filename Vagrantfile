require 'yaml'

if File.file?('./vagrant/config.yml')
    conf = YAML.load_file('./vagrant/config.yml')
end

hostname = conf["hostname"]
ip_address = conf["ip_address"]

Vagrant.configure("2") do |config|
    config.vm.box = "minimal/xenial64"

    config.trigger.before :up do
        run "bash ./vagrant/hostupdate-up.sh #{ip_address} #{hostname}"
    end
    config.trigger.before :resume do
        run "bash ./vagrant/hostupdate-up.sh #{ip_address} #{hostname}"
    end
    config.trigger.before :reload do
        run "bash ./vagrant/hostupdate-down.sh #{ip_address}"
    end
    config.trigger.after :reload do
        run "bash ./vagrant/hostupdate-up.sh #{ip_address} #{hostname}"
    end
    config.trigger.before :suspend do
        run "bash ./vagrant/hostupdate-down.sh #{ip_address}"
    end
    config.trigger.before :halt do
        run "bash ./vagrant/hostupdate-down.sh #{ip_address}"
    end
    config.trigger.before :destroy do
        run "bash ./vagrant/hostupdate-down.sh #{ip_address}"
    end

    config.vm.network "private_network", ip: ip_address
    config.vm.network "public_network"

    config.vm.synced_folder "./build", "/app", type: "nfs"

    config.vm.provision "shell", path: "./vagrant/provision.sh", privileged: false

    config.vm.provider "virtualbox" do |v|
        v.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
    end
end