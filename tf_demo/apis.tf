resource "google_project_service" "enable-api" {
  for_each = var.api_services

  service                    = each.value
  disable_dependent_services = false
  disable_on_destroy         = false
}