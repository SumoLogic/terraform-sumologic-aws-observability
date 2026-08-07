# This TF is solely used to create common AWS S3 bucket.
# 1. Create an AWS S3 Bucket.
# 2. Add CloudTrail Policy
# 3. Add ELB policy

resource "aws_s3_bucket" "s3_bucket" {
  for_each = toset(local.create_common_bucket ? ["s3_bucket"] : [])

  bucket        = local.common_bucket_name
  force_destroy = local.common_force_destroy
  tags          = var.aws_resource_tags
}

resource "aws_s3_bucket_policy" "dump_access_logs_to_s3" {
  for_each = toset(local.create_common_bucket ? ["s3_bucket"] : [])

  bucket = aws_s3_bucket.s3_bucket["s3_bucket"].id

  policy = templatefile("${path.module}/templates/s3_bucket_policy.tmpl", {
    BUCKET_NAME   = local.common_bucket_name
    AWS_PARTITION = data.aws_partition.current.partition
  })
}

# Default s3 bucket acl is private, if you want to update uncomment the following block
# For more details refer https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_acl
# resource "aws_s3_bucket_acl" "s3_bucket_acl" {
#   for_each = toset(local.create_common_bucket ? ["s3_bucket_acl"] : [])
#   bucket = aws_s3_bucket.s3_bucket["s3_bucket"].id
#   acl    = "private"
# }

resource "aws_sns_topic" "sns_topic" {
  for_each = toset(local.create_common_sns_topic ? ["sns_topic"] : [])

  name = "SumoLogic-Aws-Observability-Module-${random_string.aws_random.id}"
  policy = templatefile("${path.module}/templates/sns_topic_policy.tmpl", {
    BUCKET_NAME    = local.common_bucket_name,
    AWS_REGION     = local.aws_region,
    SNS_TOPIC_NAME = "SumoLogic-Aws-Observability-Module-${random_string.aws_random.id}",
    AWS_ACCOUNT    = local.aws_account_id
    AWS_PARTITION  = data.aws_partition.current.partition
  })
  tags = var.aws_resource_tags
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  for_each = toset(local.create_common_sns_topic ? ["bucket_notification"] : [])

  bucket = aws_s3_bucket.s3_bucket["s3_bucket"].id

  topic {
    topic_arn = aws_sns_topic.sns_topic["sns_topic"].arn
    events    = ["s3:ObjectCreated:Put"]
  }
}

# ──── Existing-bucket policy and notification management ────
# When create_bucket=false, the module must still configure the user-supplied
# existing bucket with the correct service policy and S3→SNS notifications.

data "aws_s3_bucket_policy" "existing" {
  for_each = local.existing_bucket_configs
  bucket   = each.key
}

locals {
  managed_sids = [
    "AWSCloudTrailAclCheck",
    "AWSCloudTrailWrite",
    "AWSBucketExistenceCheck",
    "AWSALBLogDeliveryAclCheck",
    "AddALBLogsStatement",
  ]

  cloudtrail_policy_statements = [
    {
      Sid       = "AWSCloudTrailAclCheck"
      Effect    = "Allow"
      Principal = { Service = "cloudtrail.amazonaws.com" }
      Action    = "s3:GetBucketAcl"
    },
    {
      Sid       = "AWSCloudTrailWrite"
      Effect    = "Allow"
      Principal = { Service = "cloudtrail.amazonaws.com" }
      Action    = "s3:PutObject"
      Condition = { StringEquals = { "s3:x-amz-acl" = "bucket-owner-full-control" } }
    },
    {
      Sid       = "AWSBucketExistenceCheck"
      Effect    = "Allow"
      Principal = { Service = "cloudtrail.amazonaws.com" }
      Action    = "s3:ListBucket"
    },
  ]

  elb_policy_statements = [
    {
      Sid       = "AWSALBLogDeliveryAclCheck"
      Effect    = "Allow"
      Principal = { Service = "delivery.logs.amazonaws.com" }
      Action    = "s3:GetBucketAcl"
    },
    {
      Sid       = "AddALBLogsStatement"
      Effect    = "Allow"
      Principal = { Service = "logdelivery.elasticloadbalancing.amazonaws.com" }
      Action    = "s3:PutObject"
    },
  ]

  existing_bucket_merged_policies = {
    for bucket_name, cfg in local.existing_bucket_configs : bucket_name => jsonencode({
      Version = "2012-10-17"
      Statement = concat(
        [
          for s in try(jsondecode(data.aws_s3_bucket_policy.existing[bucket_name].policy), { Statement = [] }).Statement :
          s if !contains(local.managed_sids, try(s.Sid, ""))
        ],
        [
          for s in local.cloudtrail_policy_statements : merge(s, {
            Resource = s.Action == "s3:PutObject" ? "arn:${data.aws_partition.current.partition}:s3:::${bucket_name}/*" : "arn:${data.aws_partition.current.partition}:s3:::${bucket_name}"
          }) if cfg.needs_cloudtrail
        ],
        [
          for s in local.elb_policy_statements : merge(s, {
            Resource = s.Action == "s3:PutObject" ? "arn:${data.aws_partition.current.partition}:s3:::${bucket_name}/*" : "arn:${data.aws_partition.current.partition}:s3:::${bucket_name}"
          }) if cfg.needs_elb || cfg.needs_classic_lb
        ],
      )
    })
  }

  existing_bucket_sns_topics = {
    for bucket_name, cfg in local.existing_bucket_configs : bucket_name => compact([
      cfg.needs_cloudtrail ? module.cloudtrail_module["cloudtrail_module"].aws_sns_topic["sns_topic"].arn : "",
      cfg.needs_elb ? module.elb_module["elb_module"].aws_sns_topic["sns_topic"].arn : "",
      cfg.needs_classic_lb ? module.classic_lb_module["classic_lb_module"].aws_sns_topic["sns_topic"].arn : "",
    ])
  }
}

resource "aws_s3_bucket_policy" "existing" {
  for_each = local.existing_bucket_configs
  bucket   = each.key
  policy   = local.existing_bucket_merged_policies[each.key]
}

resource "aws_s3_bucket_notification" "existing" {
  for_each = local.existing_bucket_configs
  bucket   = each.key

  dynamic "topic" {
    for_each = local.existing_bucket_sns_topics[each.key]
    content {
      topic_arn = topic.value
      events    = ["s3:ObjectCreated:Put"]
    }
  }

  depends_on = [aws_s3_bucket_policy.existing]
}