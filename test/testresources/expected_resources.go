package testresources

// ExpectedResources returns TF state addresses that should exist given the variable config.
// Used with AssertResourceExistence to assert presence without brittle exact counts.
func ExpectedResources(vars map[string]interface{}) []string {
	resources := []string{}

	createCollector := getBool(vars, "create_collector", true)
	if createCollector {
		resources = append(resources,
			"module.collection-module.sumologic_collector.collector",
			"module.collection-module.aws_iam_role.sumologic_iam_role",
			"module.collection-module.aws_iam_policy.sumologic_iam_policy",
		)
	}

	collectELB := getBool(vars, "collect_elb", true)
	if collectELB {
		resources = append(resources,
			`module.collection-module.module.elb_module["elb_module"].sumologic_polling_source.source`,
			`module.collection-module.module.elb_module["elb_module"].aws_sns_topic.sns_topic`,
			`module.collection-module.module.elb_module["elb_module"].aws_sns_topic_policy.sns_topic_policy`,
			`module.collection-module.module.elb_module["elb_module"].aws_sns_topic_subscription.sns_subscription`,
		)
		createELBBucket := getNestedBool(vars, "elb_details", "bucket_details", "create_bucket", true)
		if createELBBucket {
			resources = append(resources,
				`module.collection-module.module.elb_module["elb_module"].aws_s3_bucket.s3_bucket`,
			)
		}
	}

	collectCLB := getBool(vars, "collect_classic_lb", true)
	if collectCLB {
		resources = append(resources,
			`module.collection-module.module.classic_lb_module["classic_lb_module"].sumologic_polling_source.source`,
			`module.collection-module.module.classic_lb_module["classic_lb_module"].aws_sns_topic.sns_topic`,
			`module.collection-module.module.classic_lb_module["classic_lb_module"].aws_sns_topic_policy.sns_topic_policy`,
			`module.collection-module.module.classic_lb_module["classic_lb_module"].aws_sns_topic_subscription.sns_subscription`,
		)
		createCLBBucket := getNestedBool(vars, "classic_lb_details", "bucket_details", "create_bucket", true)
		if createCLBBucket {
			resources = append(resources,
				`module.collection-module.module.classic_lb_module["classic_lb_module"].aws_s3_bucket.s3_bucket`,
			)
		}
	}

	collectCloudTrail := getBool(vars, "collect_cloudtrail", true)
	if collectCloudTrail {
		resources = append(resources,
			`module.collection-module.module.cloudtrail_module["cloudtrail_module"].sumologic_polling_source.source`,
			`module.collection-module.module.cloudtrail_module["cloudtrail_module"].aws_sns_topic.sns_topic`,
			`module.collection-module.module.cloudtrail_module["cloudtrail_module"].aws_sns_topic_policy.sns_topic_policy`,
			`module.collection-module.module.cloudtrail_module["cloudtrail_module"].aws_sns_topic_subscription.sns_subscription`,
			`module.collection-module.module.cloudtrail_module["cloudtrail_module"].aws_cloudtrail.cloudtrail`,
		)
	}

	collectLogsMode := getStr(vars, "collect_logs_cloudwatch", "Kinesis Firehose Log Source")
	switch collectLogsMode {
	case "Kinesis Firehose Log Source":
		resources = append(resources,
			`module.collection-module.module.kinesis_firehose_for_logs_module["kinesis_firehose_for_logs_module"].sumologic_kinesis_log_source.source`,
			`module.collection-module.module.kinesis_firehose_for_logs_module["kinesis_firehose_for_logs_module"].aws_kinesis_firehose_delivery_stream.logs_delivery_stream`,
			`module.collection-module.module.kinesis_firehose_for_logs_module["kinesis_firehose_for_logs_module"].aws_iam_role.firehose_role`,
		)
	case "Lambda Log Forwarder":
		resources = append(resources,
			`module.collection-module.module.cloudwatch_logs_lambda_log_forwarder_module["cloudwatch_logs_lambda_log_forwarder_module"].sumologic_http_source.source`,
			`module.collection-module.module.cloudwatch_logs_lambda_log_forwarder_module["cloudwatch_logs_lambda_log_forwarder_module"].aws_lambda_function.log_forwarder`,
		)
	}

	collectMetricsMode := getStr(vars, "collect_metric_cloudwatch", "Kinesis Firehose Metrics Source")
	switch collectMetricsMode {
	case "Kinesis Firehose Metrics Source":
		resources = append(resources,
			`module.collection-module.module.kinesis_firehose_for_metrics_source_module["kinesis_firehose_for_metrics_source_module"].sumologic_kinesis_log_source.source`,
			`module.collection-module.module.kinesis_firehose_for_metrics_source_module["kinesis_firehose_for_metrics_source_module"].aws_kinesis_firehose_delivery_stream.metrics_delivery_stream`,
			`module.collection-module.module.kinesis_firehose_for_metrics_source_module["kinesis_firehose_for_metrics_source_module"].aws_cloudwatch_metric_stream.metric_stream`,
		)
	}

	// Common Sumo Logic fields always created by testSource module
	for _, field := range []string{"account", "region", "accountid", "namespace", "loadbalancer",
		"loadbalancername", "apiname", "tablename", "instanceid", "clustername",
		"cacheclusterid", "functionname", "networkloadbalancer", "dbidentifier"} {
		resources = append(resources, "sumologic_field."+field)
	}

	return resources
}

// getBool extracts a bool variable from the vars map with a default.
func getBool(vars map[string]interface{}, key string, def bool) bool {
	v, ok := vars[key]
	if !ok {
		return def
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "false" && val != "False" && val != "0"
	}
	return def
}

// getStr extracts a string variable from the vars map with a default.
func getStr(vars map[string]interface{}, key, def string) string {
	v, ok := vars[key]
	if !ok {
		return def
	}
	s, _ := v.(string)
	return s
}

// getNestedBool reads vars[topKey] as a map[string]interface{} and then looks up
// midKey → bottomKey as a bool, falling back to def.
func getNestedBool(vars map[string]interface{}, topKey, midKey, bottomKey string, def bool) bool {
	top, ok := vars[topKey]
	if !ok {
		return def
	}
	topMap, ok := top.(map[string]interface{})
	if !ok {
		return def
	}
	mid, ok := topMap[midKey]
	if !ok {
		return def
	}
	midMap, ok := mid.(map[string]interface{})
	if !ok {
		return def
	}
	return getBool(midMap, bottomKey, def)
}
