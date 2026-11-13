# Add fields to source if Metric source already exists
resource "null_resource" "AddFieldsToMetricSource" {
  for_each = toset(local.update_metrics_source ? ["add_fields_to_source"] : [])
  triggers = {
    access_id  = var.sumologic_access_id
    access_key = var.sumologic_access_key
    env        = var.sumologic_environment
    source_url = var.cloudwatch_metrics_source_url
  }
  provisioner "local-exec" {
    when    = create
    command = "python3 ${path.module}/attach_fields_to_source.py"
    environment = {
      SumoAccessID   = self.triggers.access_id
      SumoAccessKey  = self.triggers.access_key
      SumoDeployment = self.triggers.env
      SourceApiUrl   = self.triggers.source_url
      Fields         = jsonencode(local.metrics_fields)
    }
  }
}

# Add fields to source if CloudTrail source already exists
resource "null_resource" "AddFieldsToCloudTrailSource" {
  for_each = toset(local.update_cloudtrail_source ? ["add_fields_to_source"] : [])
  triggers = {
    access_id  = var.sumologic_access_id
    access_key = var.sumologic_access_key
    env        = var.sumologic_environment
    source_url = var.cloudtrail_source_url
  }
  provisioner "local-exec" {
    when    = create
    command = "python3 ${path.module}/attach_fields_to_source.py"
    environment = {
      SumoAccessID   = self.triggers.access_id
      SumoAccessKey  = self.triggers.access_key
      SumoDeployment = self.triggers.env
      SourceApiUrl   = self.triggers.source_url
      Fields         = jsonencode(local.cloudtrail_fields)
    }
  }
}

# Add fields to source if Application LoadBalancer source already exists
resource "null_resource" "AddFieldsToELBSource" {
  for_each = toset(local.update_elb_source ? ["add_fields_to_source"] : [])
  triggers = {
    access_id  = var.sumologic_access_id
    access_key = var.sumologic_access_key
    env        = var.sumologic_environment
    source_url = var.elb_log_source_url
  }
  provisioner "local-exec" {
    when    = create
    command = "python3 ${path.module}/attach_fields_to_source.py"
    environment = {
      SumoAccessID   = self.triggers.access_id
      SumoAccessKey  = self.triggers.access_key
      SumoDeployment = self.triggers.env
      SourceApiUrl   = self.triggers.source_url
      Fields         = jsonencode(local.elb_fields)
    }
  }
}

# Add fields to source if Classic LoadBalancer source already exists
resource "null_resource" "AddFieldsToCLBSource" {
  for_each = toset(local.update_classic_lb_source ? ["add_fields_to_source"] : [])
  triggers = {
    access_id  = var.sumologic_access_id
    access_key = var.sumologic_access_key
    env        = var.sumologic_environment
    source_url = var.classic_lb_log_source_url
  }
  provisioner "local-exec" {
    when    = create
    command = "python3 ${path.module}/attach_fields_to_source.py"
    environment = {
      SumoAccessID   = self.triggers.access_id
      SumoAccessKey  = self.triggers.access_key
      SumoDeployment = self.triggers.env
      SourceApiUrl   = self.triggers.source_url
      Fields         = jsonencode(local.classic_lb_fields)
    }
  }
}

# Add fields to source if CloudWatch Logs source already exists
resource "null_resource" "AddFieldsToLogSource" {
  for_each = toset(local.update_logs_source ? ["add_fields_to_source"] : [])
  triggers = {
    access_id  = var.sumologic_access_id
    access_key = var.sumologic_access_key
    env        = var.sumologic_environment
    source_url = var.cloudwatch_logs_source_url
  }
  provisioner "local-exec" {
    when    = create
    command = "python3 ${path.module}/attach_fields_to_source.py"
    environment = {
      SumoAccessID   = self.triggers.access_id
      SumoAccessKey  = self.triggers.access_key
      SumoDeployment = self.triggers.env
      SourceApiUrl   = self.triggers.source_url
      Fields         = jsonencode(local.cloudwatch_logs_fields)
    }
  }
}

# Add fields to the existing Host Metrics sources of the EC2 instances in this account/region.
# Terraform equivalent of the host_metrics_add_fields CloudFormation template. No source is created.
data "aws_instances" "host_metrics" {
  for_each = toset(var.add_fields_to_host_metrics_sources ? ["host_metrics"] : [])

  instance_state_names = ["running"]
}

resource "null_resource" "AddFieldsToHostMetricsSources" {
  for_each = toset(var.add_fields_to_host_metrics_sources ? ["add_fields_to_source"] : [])
  triggers = {
    access_id    = var.sumologic_access_id
    access_key   = var.sumologic_access_key
    env          = var.sumologic_environment
    fields       = jsonencode(local.host_metrics_fields)
    version      = var.host_metrics_fields_version
    script       = "${path.module}/attach_fields_to_source.py"
    instance_ids = jsonencode(data.aws_instances.host_metrics["host_metrics"].ids)
  }

  # The instance list changes with autoscaling; it is only used by the destroy step, which
  # removes the fields again. Change host_metrics_fields_version to refresh it.
  lifecycle {
    ignore_changes = [triggers["instance_ids"]]
  }

  provisioner "local-exec" {
    when    = create
    command = "python3 ${self.triggers.script}"
    environment = {
      Action         = "add"
      SumoAccessID   = self.triggers.access_id
      SumoAccessKey  = self.triggers.access_key
      SumoDeployment = self.triggers.env
      InstanceIds    = self.triggers.instance_ids
      Fields         = self.triggers.fields
    }
  }

  provisioner "local-exec" {
    when    = destroy
    command = "python3 ${self.triggers.script}"
    environment = {
      Action         = "remove"
      SumoAccessID   = self.triggers.access_id
      SumoAccessKey  = self.triggers.access_key
      SumoDeployment = self.triggers.env
      InstanceIds    = self.triggers.instance_ids
      Fields         = self.triggers.fields
    }
  }
}
