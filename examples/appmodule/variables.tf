variable "sumologic_environment" {
  type        = string
  description = "Enter au, ca, ch, de, eu, esc, fed, jp, kr, us1 or us2. For more information on Sumo Logic deployments visit https://help.sumologic.com/APIs/General-API-Information/Sumo-Logic-Endpoints-and-Firewall-Security"

  validation {
    condition = contains([
      "au",
      "ca",
      "ch",
      "de",
      "eu",
      "esc",
      "fed",
      "jp",
      "kr",
      "us1",
    "us2"], var.sumologic_environment)
    error_message = "The value must be one of au, ca, ch, de, eu, esc, fed, jp, kr, us1 or us2."
  }
}

variable "sumologic_access_id" {
  type        = string
  description = "Sumo Logic Access ID. Visit https://help.sumologic.com/Manage/Security/Access-Keys#Create_an_access_key"

  validation {
    condition     = can(regex("\\w+", var.sumologic_access_id))
    error_message = "The SumoLogic access ID must contain valid characters."
  }
}

variable "sumologic_access_key" {
  type        = string
  description = "Sumo Logic Access Key. Visit https://help.sumologic.com/Manage/Security/Access-Keys#Create_an_access_key"
  #sensitive = true

  validation {
    condition     = can(regex("\\w+", var.sumologic_access_key))
    error_message = "The SumoLogic access key must contain valid characters."
  }
}

variable "sumo_api_endpoint" {
  type        = string
  description = "Sumo Logic API endpoint URL. E.g. https://api.us2.sumologic.com/api/"
  default     = ""

  validation {
    condition = var.sumo_api_endpoint == "" || contains([
      "https://api.au.sumologic.com/api/",
      "https://api.ca.sumologic.com/api/",
      "https://api.ch.sumologic.com/api/",
      "https://api.de.sumologic.com/api/",
      "https://api.eu.sumologic.com/api/",
      "https://api.esc.sumologic.com/api/",
      "https://api.fed.sumologic.com/api/",
      "https://api.jp.sumologic.com/api/",
      "https://api.sumologic.com/api/",
      "https://api.us2.sumologic.com/api/",
    "https://api.kr.sumologic.com/api/"], var.sumo_api_endpoint)
    error_message = "Argument \"sumo_api_endpoint\" must be one of the values specified at https://help.sumologic.com/APIs/General-API-Information/Sumo-Logic-Endpoints-and-Firewall-Security."
  }
}
