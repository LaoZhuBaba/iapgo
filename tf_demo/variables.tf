variable "vpc_name" {
  type = string
}

variable "subnet_name" {
  type = string
}

variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "gce_front" {
  type = string
}

variable "gce_back" {
  type = string
}

variable "api_services" {
  type = set(string)
  default = ["compute.googleapis.com", "iap.googleapis.com", "oslogin.googleapis.com"]
}
