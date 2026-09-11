package testresources

// ExpectedResources returns TF state addresses that should exist given the variable config.
// Used with AssertResourceExistence to assert presence without brittle exact counts.
func ExpectedResources(vars map[string]interface{}) []string {
	resources := []string{}

	collectELB := getBool(vars, "collect_elb", true)
	collectCLB := getBool(vars, "collect_classic_lb", true)
	collectCT := getBool(vars, "collect_cloudtrail", true)
	collectLogsMode := getStr(vars, "collect_logs_cloudwatch", "Kinesis Firehose Log Source")
	collectMetricsMode := getStr(vars, "collect_metric_cloudwatch", "Kinesis Firehose Metrics Source")
	anySource := collectELB || collectCLB || collectCT || (collectLogsMode != "None") || (collectMetricsMode != "None")
	needsIAMRole := collectELB || collectCLB || collectCT || (collectMetricsMode != "None")

	createCollector := getBool(vars, "create_collector", true)
	if createCollector && anySource {
		resources = append(resources,
			`module.collection-module.sumologic_collector.collector["collector"]`,
		)
	}
	if needsIAMRole && anySource {
		resources = append(resources,
			`module.collection-module.aws_iam_role.sumologic_iam_role["sumologic_iam_role"]`,
		)
	}

	if collectELB {
		resources = append(resources,
			`module.collection-module.module.elb_module["elb_module"].sumologic_elb_source.source`,
			`module.collection-module.module.elb_module["elb_module"].aws_sns_topic_subscription.subscription["subscription"]`,
			`module.collection-module.aws_iam_policy.elb_policy["elb_policy"]`,
		)
	}

	if collectCLB {
		resources = append(resources,
			`module.collection-module.module.classic_lb_module["classic_lb_module"].sumologic_elb_source.source`,
			`module.collection-module.module.classic_lb_module["classic_lb_module"].aws_sns_topic_subscription.subscription["subscription"]`,
			`module.collection-module.aws_iam_policy.classic_lb_policy["classic_lb_policy"]`,
		)
	}

	if collectCT {
		resources = append(resources,
			`module.collection-module.module.cloudtrail_module["cloudtrail_module"].sumologic_cloudtrail_source.source`,
			`module.collection-module.module.cloudtrail_module["cloudtrail_module"].aws_cloudtrail.cloudtrail["cloudtrail"]`,
			`module.collection-module.module.cloudtrail_module["cloudtrail_module"].aws_sns_topic_subscription.subscription["subscription"]`,
			`module.collection-module.aws_iam_policy.cloudtrail_policy["cloudtrail_policy"]`,
		)
	}

	switch collectLogsMode {
	case "Kinesis Firehose Log Source":
		resources = append(resources,
			`module.collection-module.module.kinesis_firehose_for_logs_module["kinesis_firehose_for_logs_module"].sumologic_http_source.source`,
			`module.collection-module.module.kinesis_firehose_for_logs_module["kinesis_firehose_for_logs_module"].aws_kinesis_firehose_delivery_stream.logs_delivery_stream`,
			`module.collection-module.module.kinesis_firehose_for_logs_module["kinesis_firehose_for_logs_module"].aws_iam_role.firehose_role`,
		)
	case "Lambda Log Forwarder":
		resources = append(resources,
			`module.collection-module.module.cloudwatch_logs_lambda_log_forwarder_module["cloudwatch_logs_lambda_log_forwarder_module"].sumologic_http_source.source`,
			`module.collection-module.module.cloudwatch_logs_lambda_log_forwarder_module["cloudwatch_logs_lambda_log_forwarder_module"].aws_lambda_function.logs_lambda_function`,
		)
	}

	switch collectMetricsMode {
	case "Kinesis Firehose Metrics Source":
		resources = append(resources,
			`module.collection-module.module.kinesis_firehose_for_metrics_source_module["kinesis_firehose_for_metrics_source_module"].sumologic_kinesis_metrics_source.source`,
			`module.collection-module.module.kinesis_firehose_for_metrics_source_module["kinesis_firehose_for_metrics_source_module"].aws_kinesis_firehose_delivery_stream.metrics_delivery_stream`,
			`module.collection-module.module.kinesis_firehose_for_metrics_source_module["kinesis_firehose_for_metrics_source_module"].aws_cloudwatch_metric_stream.metric_stream`,
			`module.collection-module.aws_iam_policy.cw_metrics_policy["cw_metrics_policy"]`,
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
