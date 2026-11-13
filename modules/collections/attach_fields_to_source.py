from sumologic import SumoLogic
import json
import os
import sys
import time

HOST_METRICS_BATCH_SIZE = 10
HOST_METRICS_QUERY_WINDOW_MS = 60 * 60 * 1000


class SumoResource(object):

    def __init__(self, props, *args, **kwargs):
        access_id, access_key, deployment = props["SumoAccessID"], props["SumoAccessKey"], props["SumoDeployment"]
        self.deployment = deployment
        self.sumologic_cli = SumoLogic(access_id, access_key, self.api_endpoint)

    @property
    def api_endpoint(self):
        if self.deployment == "us1":
            return "https://api.sumologic.com/api"
        elif self.deployment in ["ca", "ch", "au", "de", "eu", "esc", "jp", "us2", "kr", "fed"]:
            return "https://api.%s.sumologic.com/api" % self.deployment
        else:
            return 'https://%s-api.sumologic.net/api' % self.deployment


class SumoLogicUpdateFields(SumoResource):
    """
        This Class helps you to add fields to an existing source. This class will not create a new source if not already present.
    """
    # This function can add field if it doesnt exist
    # If field already exists, it'll update value associated with the field
    def add_fields_to_source(self, collector_id, source_id, fields):
        if collector_id and source_id:
            try:
                sv, etag = self.sumologic_cli.source(collector_id, source_id)
            except:
                return f'''Status: "Collector, source not found", collector_id: {collector_id}, source_id: {source_id}''' 

            existing_fields = sv['source']['fields']
            new_fields = existing_fields.copy()
            new_fields.update(fields)

            sv['source']['fields'] = new_fields
            resp = self.sumologic_cli.update_source(collector_id, sv, etag)
            data = resp.json()['source']
            print("Added fields to collector: "+collector_id+ " and source: "+ source_id)
            return f'''source_name: {data["name"]}, source_id: {source_id}'''
        else:
            return f'''Status: "Collector, source not found", collector_id: {collector_id}, source_id: {source_id}'''

    #This function can delete fields of source
    def delete_fields_of_source(self, collector_id, source_id, fields_to_delete):
        if collector_id and source_id:
            try:
                sv, etag = self.sumologic_cli.source(collector_id, source_id)
            except:
                return f'''Status: "Collector, source not found", collector_id: {collector_id}, source_id: {source_id}''' 

            existing_fields = sv['source']['fields']
            new_fields = existing_fields.copy()
            
            for field in fields_to_delete:
                try:
                    new_fields.pop(field)
                except KeyError as ex:
                    pass

            sv['source']['fields'] = new_fields

            resp = self.sumologic_cli.update_source(collector_id, sv, etag)
            data = resp.json()['source']
            print("Deleted fields from collector: "+collector_id+ " and source: "+ source_id)
            return f'''source_name: {data["name"]}, source_id: {source_id}'''
        else:
            return f'''Status: "Collector, source not found", collector_id: {collector_id}, source_id: {source_id}'''

    # Finds the existing Host Metrics sources reporting for the given EC2 instances, using a
    # Sumo Logic metrics query over the last hour. Returns a list of unique (collector_id, source_id)
    # pairs as decimal strings. Instances are queried in batches of HOST_METRICS_BATCH_SIZE.
    def find_host_metrics_sources(self, instance_ids):
        found = []
        for idx in range(0, len(instance_ids), HOST_METRICS_BATCH_SIZE):
            query = " or ".join(f"InstanceId={i}" for i in instance_ids[idx:idx + HOST_METRICS_BATCH_SIZE])
            now = int(time.time() * 1000)
            content = {
                "query": [{
                    "query": f"_contentType=HostMetrics ({query}) | count by _sourceId, _collectorId",
                    "rowId": "A"
                }],
                "startTime": now - HOST_METRICS_QUERY_WINDOW_MS,
                "endTime": now,
                "desiredQuantizationInSecs": 600,
                "requestedDataPoints": 1
            }
            output = self.sumologic_cli.fetch_metric_data_points(content)
            for response in json.loads(output.text).get("response") or []:
                for result in response.get("results", []):
                    ids = {}
                    for dimension in result.get("metric", {}).get("dimensions", []):
                        if dimension["key"] in ("_collectorId", "_sourceId"):
                            ids[dimension["key"]] = str(int(dimension["value"], 16))
                    pair = (ids.get("_collectorId"), ids.get("_sourceId"))
                    if all(pair) and pair not in found:
                        found.append(pair)
        return found

    # Adds (action="add") or removes (action="remove") the fields on every existing Host Metrics
    # source of the given EC2 instances. Raises if a discovered source can not be updated.
    def update_host_metrics_sources(self, instance_ids, fields, action="add"):
        if not instance_ids:
            print("No EC2 instances found - nothing to do.")
            return 0
        sources = self.find_host_metrics_sources(instance_ids)
        for collector_id, source_id in sources:
            if action == "remove":
                result = self.delete_fields_of_source(collector_id, source_id, list(fields))
            else:
                result = self.add_fields_to_source(collector_id, source_id, fields)
            if result.startswith("Status:"):
                raise RuntimeError(f"Could not update Host Metrics source: {result}")
        if sources:
            print(f"Action '{action}' applied to {len(sources)} Host Metrics source(s).")
        else:
            print(f"No Host Metrics sources found for the {len(instance_ids)} EC2 instance(s) in the last hour.")
        return len(sources)


def main():
    env_vars = os.environ
    prop = {
        "SumoAccessID": env_vars.get("SumoAccessID"),
        "SumoAccessKey": env_vars.get("SumoAccessKey"),
        "SumoDeployment": env_vars.get("SumoDeployment"),
    }
    fields = json.loads(env_vars.get("Fields"))
    source = SumoLogicUpdateFields(prop)

    # Host Metrics sources: discovered through the EC2 instance IDs (no source URL is known).
    if env_vars.get("InstanceIds") is not None:
        action = env_vars.get("Action", "add")
        if action not in ("add", "remove"):
            sys.exit(f"Action must be 'add' or 'remove', got '{action}'")
        source.update_host_metrics_sources(json.loads(env_vars.get("InstanceIds")), fields, action)
        return

    # Any other existing source: the source API URL is known.
    source_url = env_vars.get("SourceApiUrl")
    source_id = source_url.split("/")[-1]
    collector_id = source_url.split("/")[-3]

    fields_to_delete = [] # Please add keys of the fields that needs to be removed on terraform destroy e.g ['FieldKeyName']
    source.add_fields_to_source(collector_id,source_id,fields)


if __name__ == "__main__":
    main()
