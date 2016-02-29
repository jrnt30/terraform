package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"log"
	"testing"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceNameValueIDHash(t *testing.T) {
	cases := []struct {
		resource_id string
		name        string
		value       string
		hash        string
	}{
		{
			resource_id: "i-12343",
			name:        "no_value_test",
			hash:        "ec2tag-4037004924",
		},
		{"i-12345", "some_value", "test value", "ec2tag-1725353070"},
	}

	for _, tc := range cases {
		actual := resourceNameValueIDHash(tc.resource_id, tc.name, tc.value)
		if actual != tc.hash {
			t.Error("Hash %s for test case %s did not match %s", actual, tc.name, tc.hash)
		}
	}
}

func TestAccAWSEc2Tags(t *testing.T) {
	var tag ec2.Tag

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsEc2TagDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEc2Tags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEc2TagExists("aws_ec2_tag.test_tag", &tag),
				),
			},
		},
	})
}

func testAccCheckAWSEc2TagExists(n string, tag *ec2.Tag) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No computed hash set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		rid := rs.Primary.Attributes["resource_id"]
		log.Printf("[INFO] RID is: %v", rs.Primary.Attributes)
		req := &ec2.DescribeTagsInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("resource-id"),
					Values: []*string{
						aws.String(rid),
					},
				},
				{
					Name: aws.String("key"),
					Values: []*string{
						aws.String("Name"),
					},
				},
				{
					Name: aws.String("value"),
					Values: []*string{
						aws.String("test_tag"),
					},
				},
			},
		}

		resp, err := conn.DescribeTags(req)

		if err != nil {
			return fmt.Errorf("Error while trying to describe tags %s", err)
		}

		if len(resp.Tags) != 1 {
			return fmt.Errorf("Recieved the incorrect number for tags for %s (%d)", rid, len(resp.Tags))
		}

		return nil
	}
}

func testAccCheckAwsEc2TagDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ec2_tag" {
			continue
		}

		req := &ec2.DescribeTagsInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("resource-id"),
					Values: []*string{
						aws.String(rs.Primary.Attributes["resource_id"]),
					},
				},
				{
					Name: aws.String("key"),
					Values: []*string{
						aws.String("Name"),
					},
				},
				{
					Name: aws.String("value"),
					Values: []*string{
						aws.String("test_tag"),
					},
				},
			},
		}

		resp, err := conn.DescribeTags(req)

		if err != nil {
			return fmt.Errorf("Unexpected error on describe call %s", err)
		}

		if len(resp.Tags) > 0 {
			return fmt.Errorf("Tags still exist for VPC")
		}
	}
	return nil
}

var testAccAWSEc2Tags = `
resource "aws_vpc" "foo" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_ec2_tag" "test_tag" {
  resource_id = "${aws_vpc.foo.id}"
  name = "Name"
  value = "test_tag"
}
`
