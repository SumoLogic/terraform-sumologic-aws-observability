terraform {
  required_version = ">= 1.5.7"

  required_providers {
    sumologic = {
      version = ">= 3.3.0, < 4.0.0"
      source  = "SumoLogic/sumologic"
    }
    time = {
      source  = "hashicorp/time"
      version = ">= 0.11.1"
    }
  }
}
