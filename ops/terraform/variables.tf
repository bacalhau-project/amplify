variable "gcp_project" {
  type = string
}
variable "region" {
  type = string
}
variable "zone" {
  type = string
}
variable "machine_type" {
  type = string
}
variable "instance_count" {
  type = string
}
variable "volume_size_gb" {
  type = number
}
variable "boot_disk_size_gb" {
  type    = number
  default = 10
}
variable "amplify_version" {
  type = string
}
# allows deploying amplify from a specific branch instead of a release
variable "amplify_branch" {
  type = string
  default = ""
}
variable "amplify_port" {
  type = string
}
variable "log_level" {
  type    = string
  default = "debug"
}
variable "ipfs_version" {
  type = string
}
// Version number, omit the 'v' prefix
variable "otel_collector_version" {
  type    = string
  default = ""
}
variable "ssh_access_cidrs" {
  type    = set(string)
  default = []
}
variable "ingress_cidrs" {
  type    = set(string)
  default = []
}