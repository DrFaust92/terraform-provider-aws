package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAwsKmsKeyPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKmsKeyPolicyPut,
		Read:   resourceAwsKmsKeyPolicyRead,
		Update: resourceAwsKmsKeyPolicyPut,
		Delete: resourceAwsKmsKeyPolicyDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"key_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
		},
	}
}

func resourceAwsKmsKeyPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	pOut, err := retryOnAwsCode(kms.ErrCodeNotFoundException, func() (interface{}, error) {
		return conn.GetKeyPolicy(&kms.GetKeyPolicyInput{
			KeyId:      aws.String(d.Id()),
			PolicyName: aws.String("default"),
		})
	})
	if err != nil {
		return err
	}

	p := pOut.(*kms.GetKeyPolicyOutput)
	policy, err := structure.NormalizeJsonString(*p.Policy)
	if err != nil {
		return fmt.Errorf("policy contains an invalid JSON: %w", err)
	}
	d.Set("policy", policy)
	d.Set("key_id", d.Id())

	return nil
}

func resourceAwsKmsKeyPolicyPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	policy, err := structure.NormalizeJsonString(d.Get("policy").(string))
	if err != nil {
		return fmt.Errorf("policy contains an invalid JSON: %w", err)
	}
	keyId := d.Get("key_id").(string)

	log.Printf("[DEBUG] KMS key: %s, update policy: %s", keyId, policy)

	req := &kms.PutKeyPolicyInput{
		KeyId:      aws.String(keyId),
		Policy:     aws.String(policy),
		PolicyName: aws.String("default"),
	}
	_, err = conn.PutKeyPolicy(req)
	if err != nil {
		return fmt.Errorf("error setting KMS Key policy (%s): %w", d.Id(), err)
	}

	d.SetId(keyId)

	return resourceAwsKmsKeyPolicyRead(d, meta)
}

func resourceAwsKmsKeyPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	req := &kms.PutKeyPolicyInput{
		KeyId:      aws.String(d.Id()),
		Policy:     aws.String("{\n  \"Version\" : \"2012-10-17\",\n  \"Id\" : \"key-default-1\",\n  \"Statement\" : [ {\n    \"Sid\" : \"Enable IAM User Permissions\",\n    \"Effect\" : \"Allow\",\n    \"Principal\" : {\n      \"AWS\" : \"arn:aws:iam::577328853911:root\"\n    },\n    \"Action\" : \"kms:*\",\n    \"Resource\" : \"*\"\n  } ]\n}"),
		PolicyName: aws.String("default"),
	}
	_, err := conn.PutKeyPolicy(req)
	if err != nil {
		return fmt.Errorf("error deleting KMS Key policy (%s): %w", d.Id(), err)
	}

	return nil
}
