resource "google_compute_instance" "gce_front" {
  name         = var.gce_front
  machine_type = "e2-micro"
  zone         = "us-central1-a"
  tags = ["front"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }
  network_interface {
    network = var.vpc_name
    subnetwork  = var.subnet_name
  }
}

resource "google_compute_instance" "gce_back" {
  name         = var.gce_back
  machine_type = "e2-micro"
  zone         = "us-central1-a"
  tags = ["back"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
      network = var.vpc_name
      subnetwork  = var.subnet_name
      access_config {} # This will give the instance a public IP which is necessary for running the startup script
  }
  metadata_startup_script = <<-EOT
  #!/bin/bash
    apt update
    apt -y install apache2
    echo '<html><body><h1>Hello World!</h1></body></html>' | tee /var/www/html/index.html
  EOT

}