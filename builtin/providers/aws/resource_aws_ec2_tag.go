package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEc2Tag() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2TagCreate,
		Read:   resourceAwsEc2TagRead,
		Delete: resourceAwsEc2TagDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 127 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 127 characters", k))
					}
					if len(value) > 3 && value[0:4] == "aws:" {
						errors = append(errors, fmt.Errorf(
							"%q cannot being with an aws: prefix", k))
					}
					return
				},
			},

			"value": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 255 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 255 characters", k))
					}
					return
				},
			},

			"resource_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					return
				},
			},
		},
	}
}

func resourceAwsEc2TagCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	resource_id := d.Get("resource_id").(string)

	awsMutexKV.Lock(resource_id)
	defer awsMutexKV.Unlock(resource_id)

	name := d.Get("name").(string)
	value := d.Get("value").(string)
	tag := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(resource_id),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(name),
				Value: aws.String(value),
			},
		},
	}

	var err error
	log.Printf("[DEBUG] Create Tags input: %#v", tag)
	_, err = conn.CreateTags(tag)

	if err != nil {
		return fmt.Errorf("Error creating Ec2 Tags: %s", err)
	}

	d.SetId(resourceNameValueIDHash(resource_id, name, value))
	return resourceAwsEc2TagRead(d, meta)
}

func resourceAwsEc2TagRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	resource_id := d.Get("resource_id").(string)
	name := d.Get("name").(string)
	value := d.Get("value").(string)

	req := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(resource_id),
				},
			},
			{
				Name: aws.String("key"),
				Values: []*string{
					aws.String(name),
				},
			},
			{
				Name: aws.String("value"),
				Values: []*string{
					aws.String(value),
				},
			},
		},
	}

	resp, err := conn.DescribeTags(req)

	if err != nil {
		log.Printf("[DEBUG] Error finding EC2 tag %s (%s) for resource (%s): %s", name, d.Id(), resource_id, err)
		d.SetId("")
		return nil
	}

	if len(resp.Tags) == 0 {
		log.Printf("[DEBUG] Unable to find EC2 tag %s (%s) for resource (%s) with value %s", name, d.Id(), resource_id, value)
		d.SetId("")
		return nil
	}

	return nil
}
func resourceAwsEc2TagDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	resource_id := d.Get("resource_id").(string)
	name := d.Get("name").(string)
	value := d.Get("value").(string)

	req := &ec2.DeleteTagsInput{
		Resources: []*string{
			aws.String(resource_id),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(name),
				Value: aws.String(value),
			},
		},
	}

	_, err := conn.DeleteTags(req)
	if err != nil {
		return fmt.Errorf("Error removing the EC2 tag %s with value %s on %s: %s", name, value, resource_id, err)
	}
	d.SetId("")

	return nil
}

func resourceNameValueIDHash(resource_id, name, value string) string {
	return fmt.Sprintf("ec2tag-%d", hashcode.String(fmt.Sprintf("%s-%s-%s", resource_id, name, value)))
}
