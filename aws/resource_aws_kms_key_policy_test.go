package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// func init() {
// 	resource.AddTestSweepers("aws_kms_key", &resource.Sweeper{
// 		Name: "aws_kms_key",
// 		F:    testSweepKmsKeys,
// 	})
// }

// func testSweepKmsKeys(region string) error {
// 	client, err := sharedClientForRegion(region)
// 	if err != nil {
// 		return fmt.Errorf("error getting client: %w", err)
// 	}
// 	conn := client.(*AWSClient).kmsconn

// 	err = conn.ListKeysPages(&kms.ListKeysInput{Limit: aws.Int64(int64(1000))}, func(out *kms.ListKeysOutput, lastPage bool) bool {
// 		for _, k := range out.Keys {
// 			kKeyId := aws.StringValue(k.KeyId)
// 			kOut, err := conn.DescribeKey(&kms.DescribeKeyInput{
// 				KeyId: k.KeyId,
// 			})
// 			if err != nil {
// 				log.Printf("Error: Failed to describe key %q: %s", kKeyId, err)
// 				return false
// 			}
// 			if aws.StringValue(kOut.KeyMetadata.KeyManager) == kms.KeyManagerTypeAws {
// 				// Skip (default) keys which are managed by AWS
// 				continue
// 			}
// 			if aws.StringValue(kOut.KeyMetadata.KeyState) == kms.KeyStatePendingDeletion {
// 				// Skip keys which are already scheduled for deletion
// 				continue
// 			}

// 			r := resourceAwsKmsKey()
// 			d := r.Data(nil)
// 			d.SetId(kKeyId)
// 			d.Set("key_id", kKeyId)
// 			d.Set("deletion_window_in_days", "7")
// 			err = r.Delete(d, client)
// 			if err != nil {
// 				log.Printf("Error: Failed to schedule key %q for deletion: %s", kKeyId, err)
// 				return false
// 			}
// 		}
// 		return !lastPage
// 	})
// 	if err != nil {
// 		if testSweepSkipSweepError(err) {
// 			log.Printf("[WARN] Skipping KMS Key sweep for %s: %s", region, err)
// 			return nil
// 		}
// 		return fmt.Errorf("Error describing KMS keys: %w", err)
// 	}

// 	return nil
// }

func TestAccAWSKmsKeyPolicy_disappears(t *testing.T) {
	var policy kms.GetKeyPolicyOutput
	rName := fmt.Sprintf("tf-testacc-kms-key-%s", acctest.RandString(13))
	resourceName := "aws_kms_key_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, kms.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: nil,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsKey(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyPolicyExists(resourceName, &policy),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsKmsKeyPolicy(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSKmsKeyPolicy_basic(t *testing.T) {
	var policy kms.GetKeyPolicyOutput
	rName := fmt.Sprintf("tf-testacc-kms-key-%s", acctest.RandString(13))
	resourceName := "aws_kms_key_policy.test"
	expectedPolicyText := `{"Version":"2012-10-17","Id":"kms-tf-1","Statement":[{"Sid":"Enable IAM User Permissions","Effect":"Allow","Principal":{"AWS":"*"},"Action":"kms:*","Resource":"*"}]}`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, kms.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: nil,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsKeyPolicyBasicConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyPolicyExists(resourceName, &policy),
					testAccCheckAWSKmsKeyHasPolicy(resourceName, expectedPolicyText),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// func TestAccAWSKmsKeyPolicy_Policy_IamRole(t *testing.T) {
// 	var key kms.KeyMetadata
// 	rName := acctest.RandomWithPrefix("tf-acc-test")
// 	resourceName := "aws_kms_key.test"

// 	resource.ParallelTest(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		ErrorCheck:   testAccErrorCheck(t, kms.EndpointsID),
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccCheckAWSKmsKeyPolicyDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccAWSKmsKeyConfigPolicyIamRole(rName),
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccCheckAWSKmsKeyPolicyExists(resourceName, &key),
// 				),
// 			},
// 			{
// 				ResourceName:            resourceName,
// 				ImportState:             true,
// 				ImportStateVerify:       true,
// 				ImportStateVerifyIgnore: []string{"deletion_window_in_days"},
// 			},
// 		},
// 	})
// }

// // Reference: https://github.com/hashicorp/terraform-provider-aws/issues/7646
// func TestAccAWSKmsKeyPolicy_Policy_IamServiceLinkedRole(t *testing.T) {
// 	var key kms.KeyMetadata
// 	rName := acctest.RandomWithPrefix("tf-acc-test")
// 	resourceName := "aws_kms_key.test"

// 	resource.ParallelTest(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		ErrorCheck:   testAccErrorCheck(t, kms.EndpointsID),
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccCheckAWSKmsKeyPolicyDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccAWSKmsKeyConfigPolicyIamServiceLinkedRole(rName),
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccCheckAWSKmsKeyPolicyExists(resourceName, &key),
// 				),
// 			},
// 			{
// 				ResourceName:            resourceName,
// 				ImportState:             true,
// 				ImportStateVerify:       true,
// 				ImportStateVerifyIgnore: []string{"deletion_window_in_days"},
// 			},
// 		},
// 	})
// }

// func testAccCheckAWSKmsKeyPolicyDestroy(s *terraform.State) error {
// 	conn := testAccProvider.Meta().(*AWSClient).kmsconn

// 	for _, rs := range s.RootModule().Resources {
// 		if rs.Type != "aws_kms_key_policy" {
// 			continue
// 		}

// 		out, err := conn.DescribeKey(&kms.DescribeKeyInput{
// 			KeyId: aws.String(rs.Primary.ID),
// 		})

// 		if err != nil {
// 			return err
// 		}

// 		if *out.KeyMetadata.KeyState == "PendingDeletion" {
// 			return nil
// 		}

// 		return fmt.Errorf("KMS key still exists:\n%#v", out.KeyMetadata)
// 	}

// 	return nil
// }

func testAccCheckAWSKmsKeyPolicyExists(name string, policy *kms.GetKeyPolicyOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No KMS Key ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).kmsconn
		out, err := conn.GetKeyPolicy(&kms.GetKeyPolicyInput{
			KeyId:      aws.String(rs.Primary.ID),
			PolicyName: aws.String("default"),
		})
		if err != nil {
			return err
		}

		*policy = *out

		return nil
	}
}

func testAccAWSKmsKeyPolicyBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "test" {
  description             = %[1]q
  deletion_window_in_days = 7
}

resource "aws_kms_key_policy" "test" {
  key_id             = aws_kms_key.test.arn
  
  policy = jsonencode({
	"Version": "2012-10-17",
	"Id": "kms-tf-1",
	"Statement": [
	  {
		"Sid": "Enable IAM User Permissions",
		"Effect": "Allow",
		"Principal": {
		  "AWS": "*"
		},
		"Action": "kms:*",
		"Resource": "*"
	  }
	]
  })
}
`, rName)
}

// func testAccAWSKmsKeyConfigPolicyIamRole(rName string) string {
// 	return fmt.Sprintf(`
// data "aws_partition" "current" {}

// resource "aws_iam_role" "test" {
//   name = %[1]q

//   assume_role_policy = jsonencode({
//     Statement = [{
//       Action = "sts:AssumeRole"
//       Effect = "Allow"
//       Principal = {
//         Service = "ec2.${data.aws_partition.current.dns_suffix}"
//       }
//     }]
//     Version = "2012-10-17"
//   })
// }

// resource "aws_kms_key" "test" {
//   description             = %[1]q
//   deletion_window_in_days = 7

//   policy = jsonencode({
//     Id = "kms-tf-1"
//     Statement = [
//       {
//         Action = "kms:*"
//         Effect = "Allow"
//         Principal = {
//           AWS = "*"
//         }

//         Resource = "*"
//         Sid      = "Enable IAM User Permissions"
//       },
//       {
//         Action = [
//           "kms:Encrypt",
//           "kms:Decrypt",
//           "kms:ReEncrypt*",
//           "kms:GenerateDataKey*",
//           "kms:DescribeKey",
//         ]
//         Effect = "Allow"
//         Principal = {
//           AWS = [aws_iam_role.test.arn]
//         }

//         Resource = "*"
//         Sid      = "Enable IAM User Permissions"
//       },
//     ]
//     Version = "2012-10-17"
//   })
// }
// `, rName)
// }

// func testAccAWSKmsKeyConfigPolicyIamServiceLinkedRole(rName string) string {
// 	return fmt.Sprintf(`
// data "aws_partition" "current" {}

// resource "aws_iam_service_linked_role" "test" {
//   aws_service_name = "autoscaling.${data.aws_partition.current.dns_suffix}"
//   custom_suffix    = %[1]q
// }

// resource "aws_kms_key" "test" {
//   description             = %[1]q
//   deletion_window_in_days = 7

//   policy = jsonencode({
//     Id = "kms-tf-1"
//     Statement = [
//       {
//         Action = "kms:*"
//         Effect = "Allow"
//         Principal = {
//           AWS = "*"
//         }

//         Resource = "*"
//         Sid      = "Enable IAM User Permissions"
//       },
//       {
//         Action = [
//           "kms:Encrypt",
//           "kms:Decrypt",
//           "kms:ReEncrypt*",
//           "kms:GenerateDataKey*",
//           "kms:DescribeKey",
//         ]
//         Effect = "Allow"
//         Principal = {
//           AWS = [aws_iam_service_linked_role.test.arn]
//         }

//         Resource = "*"
//         Sid      = "Enable IAM User Permissions"
//       },
//     ]
//     Version = "2012-10-17"
//   })
// }
// `, rName)
// }
