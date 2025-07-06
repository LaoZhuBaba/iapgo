resource "google_compute_network" "vpc" {
  name                    = var.vpc_name
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  name          = var.subnet_name
  region        = var.region
  network       = google_compute_network.vpc.name
  ip_cidr_range = "10.10.10.0/24"
}

resource "google_compute_firewall" "allow_ssh_via_iap" {
  name      = "allow-ingress-via-iap"
  network   = google_compute_network.vpc.name
  direction = "INGRESS"
  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
  source_ranges = ["35.235.240.0/20"] # IAP CIDR
  target_tags = ["front", "back"]
}

resource "google_compute_firewall" "allow_http_from_front" {
  name      = "allow-http-from-front"
  network   = google_compute_network.vpc.name
  direction = "INGRESS"
  allow {
    protocol = "tcp"
    ports    = ["80"]
  }
  source_tags = ["front"]
  target_tags = ["back"]
}

resource "google_compute_firewall" "allow_ssh_from_front" {
  name      = "allow-ssh-from-front"
  network   = google_compute_network.vpc.name
  direction = "INGRESS"
  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
  source_tags = ["front"]
  target_tags = ["back"]
}