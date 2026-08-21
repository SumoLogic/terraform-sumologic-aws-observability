locals {
  region_bucket_map = {
    "us-east-1"      = "appdevzipfiles-us-east-1"
    "us-east-2"      = "appdevzipfiles-us-east-2"
    "us-west-1"      = "appdevzipfiles-us-west-1"
    "us-west-2"      = "appdevzipfiles-us-west-2"
    "ap-south-1"     = "appdevzipfiles-ap-south-1"
    "ap-northeast-2" = "appdevzipfiles-ap-northeast-2"
    "ap-southeast-1" = "appdevzipfiles-ap-southeast-1"
    "ap-southeast-2" = "appdevzipfiles-ap-southeast-2"
    "ap-northeast-1" = "appdevzipfiles-ap-northeast-1"
    "ca-central-1"   = "appdevzipfiles-ca-central-1"
    "eu-central-1"   = "appdevzipfiles-eu-central-1"
    "eu-west-1"      = "appdevzipfiles-eu-west-1"
    "eu-west-2"      = "appdevzipfiles-eu-west-2"
    "eu-west-3"      = "appdevzipfiles-eu-west-3"
    "eu-north-1"     = "appdevzipfiles-eu-north-1s"
    "sa-east-1"      = "appdevzipfiles-sa-east-1"
    "ap-east-1"      = "appdevzipfiles-ap-east-1s"
    "af-south-1"     = "appdevzipfiles-af-south-1s"
    "eu-south-1"     = "appdevzipfiles-eu-south-1"
    "me-south-1"     = "appdevzipfiles-me-south-1s"
    "me-central-1"   = "appdevzipfiles-me-central-1"
    "eu-central-2"   = "appdevzipfiles-eu-central-2ss"
    "ap-northeast-3" = "appdevzipfiles-ap-northeast-3s"
    "ap-southeast-3" = "appdevzipfiles-ap-southeast-3"
    "il-central-1"   = "appdevzipfiles-il-central-1"
    "ap-southeast-4" = "appdevzipfiles-ap-southeast-4s"
    "ap-southeast-6" = "appdevzipfiles-ap-southeast-6ss"
  }
}

resource "aws_iam_role" "lambda_helper_role" {
  for_each = toset(local.any_existing_bucket_source ? ["lambda_helper"] : [])

  name = "SumoLogic-AWSO-LambdaHelper-${random_string.aws_random.id}"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  tags = var.aws_resource_tags
}

resource "aws_iam_role_policy" "lambda_helper_policy" {
  for_each = toset(local.any_existing_bucket_source ? ["lambda_helper"] : [])

  name = "SumoLogic-AWSO-LambdaHelper-Policy"
  role = aws_iam_role.lambda_helper_role["lambda_helper"].id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "arn:${data.aws_partition.current.partition}:logs:*:*:*"
      },
      {
        Effect = "Allow"
        Action = [
          "sns:CreateTopic",
          "sns:DeleteTopic",
          "sns:SetTopicAttributes",
          "sns:Subscribe",
          "sns:Unsubscribe",
          "sns:ListSubscriptionsByTopic",
        ]
        Resource = "arn:${data.aws_partition.current.partition}:sns:${local.aws_region}:${local.aws_account_id}:sumo-s3-notif-*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetBucketNotification",
          "s3:PutBucketNotification",
          "s3:GetBucketPolicy",
          "s3:PutBucketPolicy",
          "s3:DeleteBucketPolicy",
        ]
        Resource = [for b in local.existing_bucket_names : "arn:${data.aws_partition.current.partition}:s3:::${b}"]
      },
    ]
  })
}

resource "aws_lambda_function" "lambda_helper" {
  for_each = toset(local.any_existing_bucket_source ? ["lambda_helper"] : [])

  function_name = "SumoLogic-AWSO-Helper-${random_string.aws_random.id}"
  handler       = "main.handler"
  runtime       = "python3.14"
  timeout       = 300
  memory_size   = 128
  role          = aws_iam_role.lambda_helper_role["lambda_helper"].arn
  s3_bucket     = local.region_bucket_map[local.aws_region]
  s3_key        = "sumologic-aws-observability/functions/sumo-app-utils/v3.0.0/sumo-app-utils.zip"
  tags          = var.aws_resource_tags
}
