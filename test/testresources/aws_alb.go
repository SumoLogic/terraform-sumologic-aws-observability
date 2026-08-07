package testresources

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// AWSALB creates/deletes an internet-facing Application Load Balancer as a test prerequisite.
// Creates a security group (HTTP/80 open) and a fixed-response listener so HTTP requests
// actually generate access logs. Mirrors the CF test PreRequisitesInfra pattern.
type AWSALB struct {
	Cfg  Config
	Name string
	arn  string
	sgID string
	dns  string
}

func (a *AWSALB) Create(t *testing.T) string {
	region := a.Cfg.region()

	vpcID := shellOutput(fmt.Sprintf(
		`aws ec2 describe-vpcs --region %s --filters "Name=isDefault,Values=true" --query 'Vpcs[0].VpcId' --output text`,
		region,
	))
	if vpcID == "" || vpcID == "None" {
		// No default VPC — fall back to the first available VPC in the region
		vpcID = shellOutput(fmt.Sprintf(
			`aws ec2 describe-vpcs --region %s --query 'Vpcs[0].VpcId' --output text`,
			region,
		))
		if vpcID == "" || vpcID == "None" {
			t.Fatalf("[testresources] AWSALB: no VPC found in %s", region)
		}
		t.Logf("[testresources] AWSALB: no default VPC, using %s", vpcID)
	}

	subnets := shellOutput(fmt.Sprintf(
		`aws ec2 describe-subnets --region %s --filters "Name=default-for-az,Values=true" --query 'Subnets[*].SubnetId' --output text`,
		region,
	))
	if subnets == "" {
		// No default subnets — fall back to any subnets in the resolved VPC
		subnets = shellOutput(fmt.Sprintf(
			`aws ec2 describe-subnets --region %s --filters "Name=vpc-id,Values=%s" --query 'Subnets[*].SubnetId' --output text`,
			region, vpcID,
		))
	}
	if subnets == "" {
		t.Fatalf("[testresources] AWSALB: no subnets found in VPC %s (%s)", vpcID, region)
	}
	subnetArgs := ""
	for _, s := range strings.Fields(subnets) {
		subnetArgs += " " + s
	}

	sgName := a.Name + "-sg"
	a.sgID = shellOutput(fmt.Sprintf(
		`aws ec2 create-security-group --group-name "%s" --description "ALB test SG" --vpc-id "%s" --region %s --query 'GroupId' --output text`,
		sgName, vpcID, region,
	))
	if a.sgID == "" || a.sgID == "None" {
		t.Fatalf("[testresources] AWSALB: failed to create SG %s", sgName)
	}
	shellOutput(fmt.Sprintf(
		`aws ec2 authorize-security-group-ingress --group-id "%s" --protocol tcp --port 80 --cidr 0.0.0.0/0 --region %s`,
		a.sgID, region,
	))

	a.arn = shellOutput(fmt.Sprintf(
		`aws elbv2 create-load-balancer --name "%s" --subnets%s --security-groups "%s" --scheme internet-facing --type application --region %s --query 'LoadBalancers[0].LoadBalancerArn' --output text`,
		a.Name, subnetArgs, a.sgID, region,
	))
	if a.arn == "" || a.arn == "None" {
		t.Fatalf("[testresources] AWSALB: failed to create ALB %s", a.Name)
	}
	t.Logf("[testresources] Created ALB %s (%s), waiting for active...", a.Name, a.arn)

	for i := 0; i < 20; i++ {
		state := shellOutput(fmt.Sprintf(
			`aws elbv2 describe-load-balancers --load-balancer-arns "%s" --region %s --query 'LoadBalancers[0].State.Code' --output text`,
			a.arn, region,
		))
		if state == "active" {
			break
		}
		time.Sleep(15 * time.Second)
	}

	a.dns = shellOutput(fmt.Sprintf(
		`aws elbv2 describe-load-balancers --load-balancer-arns "%s" --region %s --query 'LoadBalancers[0].DNSName' --output text`,
		a.arn, region,
	))

	shellOutput(fmt.Sprintf(
		`aws elbv2 create-listener --load-balancer-arn "%s" --protocol HTTP --port 80 --default-actions Type=fixed-response,FixedResponseConfig='{StatusCode="200",MessageBody="OK",ContentType="text/plain"}' --region %s`,
		a.arn, region,
	))
	t.Logf("[testresources] ALB %s active (DNS: %s)", a.Name, a.dns)
	return a.arn
}

func (a *AWSALB) Delete(t *testing.T) {
	region := a.Cfg.region()
	if a.arn != "" {
		shellOutput(fmt.Sprintf(
			`aws elbv2 delete-load-balancer --load-balancer-arn "%s" --region %s 2>&1`,
			a.arn, region,
		))
		t.Logf("[testresources] Deleted ALB %s", a.Name)
		time.Sleep(30 * time.Second)
	}
	if a.sgID != "" {
		shellOutput(fmt.Sprintf(
			`aws ec2 delete-security-group --group-id "%s" --region %s 2>&1`,
			a.sgID, region,
		))
	}
}

func (a *AWSALB) ID() string  { return a.arn }
func (a *AWSALB) DNS() string { return a.dns }
