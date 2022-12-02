package rbin_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rbin"
	"github.com/aws/aws-sdk-go-v2/service/rbin/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"

	tfrbin "github.com/hashicorp/terraform-provider-aws/internal/service/rbin"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccRBinRBinRule_basic(t *testing.T) {
	var rbinrule rbin.GetRuleOutput
	description := "my test description"
	resourceType := "EBS_SNAPSHOT"
	resourceName := "aws_rbin_rbin_rule.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckPartitionHasService(rbin.ServiceID, t)
			testAccPreCheck(t)
		},
		ErrorCheck:        acctest.ErrorCheck(t, rbin.ServiceID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckRBinRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRBinRuleConfig_basic(description, resourceType),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRBinRuleExists(resourceName, &rbinrule),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "resource_type", resourceType),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "retention_period.*", map[string]string{
						"retention_period_value": "10",
						"retention_period_unit":  "DAYS",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "resource_tags.*", map[string]string{
						"resource_tag_key":   "some_tag",
						"resource_tag_value": "",
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"apply_immediately", "user"},
			},
		},
	})
}

func TestAccRBinRBinRule_disappears(t *testing.T) {
	var rbinrule rbin.GetRuleOutput
	description := "my test description"
	resourceType := "EBS_SNAPSHOT"
	resourceName := "aws_rbin_rbin_rule.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckPartitionHasService(rbin.ServiceID, t)
			testAccPreCheck(t)
		},
		ErrorCheck:        acctest.ErrorCheck(t, rbin.ServiceID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckRBinRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRBinRuleConfig_basic(description, resourceType),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRBinRuleExists(resourceName, &rbinrule),
					acctest.CheckResourceDisappears(acctest.Provider, tfrbin.ResourceRBinRule(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckRBinRuleDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).RBinConn
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_rbin_rbin_rule" {
			continue
		}

		_, err := conn.GetRule(ctx, &rbin.GetRuleInput{
			Identifier: aws.String(rs.Primary.ID),
		})
		if err != nil {
			var nfe *types.ResourceNotFoundException
			if errors.As(err, &nfe) {
				return nil
			}
			return err
		}

		return names.Error(names.RBin, names.ErrActionCheckingDestroyed, tfrbin.ResNameRBinRule, rs.Primary.ID, errors.New("not destroyed"))
	}

	return nil
}

func testAccCheckRBinRuleExists(name string, rbinrule *rbin.GetRuleOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return names.Error(names.RBin, names.ErrActionCheckingExistence, tfrbin.ResNameRBinRule, name, errors.New("not found"))
		}

		if rs.Primary.ID == "" {
			return names.Error(names.RBin, names.ErrActionCheckingExistence, tfrbin.ResNameRBinRule, name, errors.New("not set"))
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).RBinConn
		ctx := context.Background()
		resp, err := conn.GetRule(ctx, &rbin.GetRuleInput{
			Identifier: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return names.Error(names.RBin, names.ErrActionCheckingExistence, tfrbin.ResNameRBinRule, rs.Primary.ID, err)
		}

		*rbinrule = *resp

		return nil
	}
}

func testAccPreCheck(t *testing.T) {
	conn := acctest.Provider.Meta().(*conns.AWSClient).RBinConn
	ctx := context.Background()

	input := &rbin.ListRulesInput{
		ResourceType: types.ResourceTypeEc2Image,
	}
	_, err := conn.ListRules(ctx, input)
	if acctest.PreCheckSkipError(err) {
		t.Skipf("skipping acceptance testing: %s", err)
	}

	input = &rbin.ListRulesInput{
		ResourceType: types.ResourceTypeEbsSnapshot,
	}
	_, err = conn.ListRules(ctx, input)
	if acctest.PreCheckSkipError(err) {
		t.Skipf("skipping acceptance testing: %s", err)
	}

	if err != nil {
		t.Fatalf("unexpected PreCheck error: %s", err)
	}
}

func testAccCheckRBinRuleNotRecreated(before, after *rbin.GetRuleOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if before, after := aws.ToString(before.Identifier), aws.ToString(after.Identifier); before != after {
			return names.Error(names.RBin, names.ErrActionCheckingNotRecreated, tfrbin.ResNameRBinRule, before, errors.New("recreated"))
		}

		return nil
	}
}

func testAccRBinRuleConfig_basic(description, resourceType string) string {
	return fmt.Sprintf(`
resource "aws_rbin_rbin_rule" "test" {
  description   = %[1]q
  resource_type = %[2]q
  resource_tags {
    resource_tag_key   = "some_tag"
    resource_tag_value = ""
  }

  retention_period {
    retention_period_value = 10
    retention_period_unit  = "DAYS"
  }
  
  tags = {
    "test_tag_key" = "test_tag_value"
  }
	
}
`, description, resourceType)
}
