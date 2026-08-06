package testresources

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// AWSClassicLB creates/deletes an internet-facing Classic Load Balancer as a test prerequisite.
type AWSClassicLB struct {
	Cfg  Config
	Name string // max 32 chars for Classic LB
	dns  string
	sgID string
}

func (c *AWSClassicLB) Create(t *testing.T) string {
	region := c.Cfg.region()

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
			t.Fatalf("[testresources] AWSClassicLB: no VPC found in %s", region)
		}
		t.Logf("[testresources] AWSClassicLB: no default VPC, using %s", vpcID)
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
	subnetList := strings.Fields(subnets)
	if len(subnetList) < 2 {
		t.Fatalf("[testresources] AWSClassicLB: need at least 2 subnets in VPC %s, found %d", vpcID, len(subnetList))
	}

	sgName := c.Name + "-sg"
	c.sgID = shellOutput(fmt.Sprintf(
		`aws ec2 create-security-group --group-name "%s" --description "CLB test SG" --vpc-id "%s" --region %s --query 'GroupId' --output text`,
		sgName, vpcID, region,
	))
	if c.sgID == "" || c.sgID == "None" {
		t.Fatalf("[testresources] AWSClassicLB: failed to create SG %s", sgName)
	}
	shellOutput(fmt.Sprintf(
		`aws ec2 authorize-security-group-ingress --group-id "%s" --protocol tcp --port 80 --cidr 0.0.0.0/0 --region %s`,
		c.sgID, region,
	))

	c.dns = shellOutput(fmt.Sprintf(
		`aws elb create-load-balancer --load-balancer-name "%s" --listeners "Protocol=HTTP,LoadBalancerPort=80,InstanceProtocol=HTTP,InstancePort=80" --subnets %s %s --security-groups "%s" --scheme internet-facing --region %s --query 'DNSName' --output text`,
		c.Name, subnetList[0], subnetList[1], c.sgID, region,
	))
	if c.dns == "" || c.dns == "None" {
		t.Fatalf("[testresources] AWSClassicLB: failed to create CLB %s", c.Name)
	}

	shellOutput(fmt.Sprintf(
		`aws elb configure-health-check --load-balancer-name "%s" --health-check Target=HTTP:80/,Interval=30,UnhealthyThreshold=2,HealthyThreshold=2,Timeout=5 --region %s`,
		c.Name, region,
	))
	t.Logf("[testresources] Created Classic LB %s (DNS: %s)", c.Name, c.dns)
	return c.dns
}

func (c *AWSClassicLB) Delete(t *testing.T) {
	region := c.Cfg.region()
	if c.Name != "" {
		shellOutput(fmt.Sprintf(
			`aws elb delete-load-balancer --load-balancer-name "%s" --region %s 2>&1`,
			c.Name, region,
		))
		t.Logf("[testresources] Deleted Classic LB %s", c.Name)
		time.Sleep(15 * time.Second)
	}
	if c.sgID != "" {
		shellOutput(fmt.Sprintf(
			`aws ec2 delete-security-group --group-id "%s" --region %s 2>&1`,
			c.sgID, region,
		))
	}
}

func (c *AWSClassicLB) ID() string  { return c.Name }
func (c *AWSClassicLB) DNS() string { return c.dns }
