provider "google" {
  project = var.gcp_project
  region  = var.region
  zone    = var.zone
}

terraform {
  backend "gcs" {
    # this bucket lives in the bacalhau-cicd google project
    # https://console.cloud.google.com/storage/browser/bacalhau-global-storage;tab=objects?project=bacalhau-cicd
    bucket = "bacalhau-global-storage"
    prefix = "terraform/amplify"
  }
}

// A single Google Cloud Engine instance
resource "google_compute_instance" "amplify_vm" {
  name         = "amplify-vm-${terraform.workspace}-${count.index}"
  count        = var.instance_count
  machine_type = var.machine_type
  zone         = var.zone

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
      size  = var.boot_disk_size_gb
    }
  }

  metadata_startup_script = <<-EOF
#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

sudo mkdir -p /terraform_node

##############################
# export the terraform variables ready for scripts to use
# we write these to a file so the amplify startup script
# called by systemd can also source them
##############################

sudo tee /terraform_node/variables > /dev/null <<'EOI'
export TERRAFORM_WORKSPACE="${terraform.workspace}"
export TERRAFORM_NODE_INDEX="${count.index}"
export TERRAFORM_NODE0_IP="${google_compute_address.ipv4_address[count.index].address}"
export IPFS_VERSION="${var.ipfs_version}"
export AMPLIFY_ENVIRONMENT="${terraform.workspace}"
export AMPLIFY_VERSION="${var.amplify_version}"
export AMPLIFY_BRANCH="${var.amplify_branch}"
export AMPLIFY_PORT="${var.amplify_port}"
export OTEL_COLLECTOR_VERSION="${var.otel_collector_version}"
EOI

##############################
# write the local files to the node filesystem
##############################

#########
# node scripts
#########

sudo mkdir -p /terraform_node

sudo tee /terraform_node/install-node.sh > /dev/null <<'EOI'
${file("${path.module}/remote_files/scripts/install-node.sh")}
EOI

sudo tee /terraform_node/start-amplify.sh > /dev/null <<'EOI'
${file("${path.module}/remote_files/scripts/start-amplify.sh")}
EOI

#########
# config files
#########


sudo mkdir -p /terraform_node

sudo tee /terraform_node/config.yaml > /dev/null <<'EOI'
${file("${path.module}/../../config.yaml")}
EOI

#########
# health checker
#########

sudo mkdir -p /var/www/health_checker

# this will be copied to the correct location once openresty has installed to avoid
# an interactive prompt warning about the file existing blocking the headless install
sudo tee /terraform_node/nginx.conf > /dev/null <<'EOI'
${file("${path.module}/remote_files/health_checker/nginx.conf")}
EOI

sudo tee /var/www/health_checker/livez.sh > /dev/null <<'EOI'
${file("${path.module}/remote_files/health_checker/livez.sh")}
EOI

sudo tee /var/www/health_checker/healthz.sh > /dev/null <<'EOI'
${file("${path.module}/remote_files/health_checker/healthz.sh")}
EOI

sudo tee /var/www/health_checker/network_name.txt > /dev/null <<EOI
${google_compute_network.amplify_network[0].name}
EOI

sudo tee /var/www/health_checker/address.txt > /dev/null <<EOI
${google_compute_address.ipv4_address[count.index].address}
EOI

sudo chmod u+x /var/www/health_checker/*.sh

#########
# systemd units
#########

sudo tee /etc/systemd/system/ipfs.service > /dev/null <<'EOI'
${file("${path.module}/remote_files/configs/ipfs.service")}
EOI

sudo tee /etc/systemd/system/amplify.service > /dev/null <<'EOI'
${file("${path.module}/remote_files/configs/amplify.service")}
EOI

sudo tee /etc/systemd/system/otel.service > /dev/null <<'EOI'
${file("${path.module}/remote_files/configs/otel.service")}
EOI

sudo tee /etc/systemd/system/promtail.service > /dev/null <<'EOI'
${file("${path.module}/remote_files/configs/promtail.service")}
EOI

##############################
# run the install script
##############################

sudo bash /terraform_node/install-node.sh 2>&1 | tee -a /tmp/amplify.log
EOF
  network_interface {
    network = google_compute_network.amplify_network[0].name
    access_config {
      nat_ip = google_compute_address.ipv4_address[count.index].address
    }
  }

  lifecycle {
    ignore_changes = [attached_disk]
  }
  allow_stopping_for_update = true
}

resource "google_compute_address" "ipv4_address" {
  count = var.instance_count
  region = var.region
  # keep the same ip addresses if we are production (because they are in DNS and the auto connect serve codebase)
  name  = "amplify-ipv4-address-${terraform.workspace}-${count.index}"
  lifecycle {
    prevent_destroy = true
  }
}

output "public_ip_address" {
  value = google_compute_instance.amplify_vm.*.network_interface.0.access_config.0.nat_ip
}

resource "google_compute_disk" "amplify_disk" {
  name  = "amplify-disk-${terraform.workspace}-${count.index}"
  count = var.instance_count
  type  = "pd-ssd"
  zone  = var.zone
  size  = var.volume_size_gb
  lifecycle {
    prevent_destroy = true
  }
}

resource "google_compute_disk_resource_policy_attachment" "attachment" {
  count = var.instance_count
  name  = google_compute_resource_policy.amplify_disk_backups[count.index].name
  disk  = google_compute_disk.amplify_disk[count.index].name
  zone  = var.zone
}

resource "google_compute_resource_policy" "amplify_disk_backups" {
  name   = "amplify-disk-backups-${terraform.workspace}-${count.index}"
  region = var.region
  count  = var.instance_count
  snapshot_schedule_policy {
    schedule {
      daily_schedule {
        days_in_cycle = 1
        start_time    = "23:00"
      }
    }
    retention_policy {
      max_retention_days    = 30
      on_source_disk_delete = "KEEP_AUTO_SNAPSHOTS"
    }
    snapshot_properties {
      labels = {
        amplify_backup = "true"
      }
      # this only works with Windows and looks like it's non-negotiable with gcp
      guest_flush = false
    }
  }
}

resource "google_compute_attached_disk" "default" {
  count    = var.instance_count
  disk     = google_compute_disk.amplify_disk[count.index].self_link
  instance = google_compute_instance.amplify_vm[count.index].self_link
  zone     = var.zone
}

resource "google_compute_firewall" "amplify_firewall" {
  name    = "amplify-ingress-firewall-${terraform.workspace}"
  network = google_compute_network.amplify_network[0].name

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports = [
      "4001",  // ipfs swarm
      "80",    // amplify API
      "13133", // otel collector health_check extension
      "55679", // otel collector zpages extension
      "44443", // nginx is healthy - for running health check scripts
      "44444", // nginx node health check scripts
    ]
  }

  source_ranges = var.ingress_cidrs
}

resource "google_compute_firewall" "amplify_ssh_firewall" {
  name    = "amplify-ssh-firewall-${terraform.workspace}"
  network =google_compute_network.amplify_network[0].name 

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    // Port 22   - Provides ssh access to the amplify server, for debugging
    ports = ["22"]
  }

  source_ranges = var.ssh_access_cidrs
}

resource "google_compute_network" "amplify_network" {
  name                    = "amplify-network-${terraform.workspace}"
  auto_create_subnetworks = true
  count                   = 1
}