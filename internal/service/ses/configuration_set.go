// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ses

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/enum"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKResource("aws_ses_configuration_set")
func ResourceConfigurationSet() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceConfigurationSetCreate,
		ReadWithoutTimeout:   resourceConfigurationSetRead,
		UpdateWithoutTimeout: resourceConfigurationSetUpdate,
		DeleteWithoutTimeout: resourceConfigurationSetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			names.AttrARN: {
				Type:     schema.TypeString,
				Computed: true,
			},
			"delivery_options": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"tls_policy": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          awstypes.TlsPolicyOptional,
							ValidateDiagFunc: enum.Validate[awstypes.TlsPolicy](),
						},
					},
				},
			},
			"last_fresh_start": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"reputation_metrics_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"sending_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			names.AttrName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 64),
			},
			"tracking_options": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"custom_redirect_domain": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringDoesNotMatch(regexache.MustCompile(`\.$`), "cannot end with a period"),
						},
					},
				},
			},
		},
	}
}

func resourceConfigurationSetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).SESClient(ctx)

	configurationSetName := d.Get(names.AttrName).(string)

	createOpts := &ses.CreateConfigurationSetInput{
		ConfigurationSet: &awstypes.ConfigurationSet{
			Name: aws.String(configurationSetName),
		},
	}

	_, err := conn.CreateConfigurationSet(ctx, createOpts)
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating SES configuration set (%s): %s", configurationSetName, err)
	}

	d.SetId(configurationSetName)

	if v, ok := d.GetOk("delivery_options"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input := &ses.PutConfigurationSetDeliveryOptionsInput{
			ConfigurationSetName: aws.String(configurationSetName),
			DeliveryOptions:      expandConfigurationSetDeliveryOptions(v.([]interface{})),
		}

		_, err := conn.PutConfigurationSetDeliveryOptions(ctx, input)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "adding SES configuration set (%s) delivery options: %s", configurationSetName, err)
		}
	}

	if v := d.Get("reputation_metrics_enabled"); v.(bool) {
		input := &ses.UpdateConfigurationSetReputationMetricsEnabledInput{
			ConfigurationSetName: aws.String(configurationSetName),
			Enabled:              v.(bool),
		}

		_, err := conn.UpdateConfigurationSetReputationMetricsEnabled(ctx, input)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "adding SES configuration set (%s) reputation metrics enabled: %s", configurationSetName, err)
		}
	}

	if v := d.Get("sending_enabled"); !v.(bool) {
		input := &ses.UpdateConfigurationSetSendingEnabledInput{
			ConfigurationSetName: aws.String(configurationSetName),
			Enabled:              v.(bool),
		}

		_, err := conn.UpdateConfigurationSetSendingEnabled(ctx, input)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "adding SES configuration set (%s) sending enabled: %s", configurationSetName, err)
		}
	}

	if v, ok := d.GetOk("tracking_options"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input := &ses.CreateConfigurationSetTrackingOptionsInput{
			ConfigurationSetName: aws.String(configurationSetName),
			TrackingOptions:      expandConfigurationSetTrackingOptions(v.([]interface{})),
		}

		_, err := conn.CreateConfigurationSetTrackingOptions(ctx, input)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "adding SES configuration set (%s) tracking options: %s", configurationSetName, err)
		}
	}

	return append(diags, resourceConfigurationSetRead(ctx, d, meta)...)
}

func resourceConfigurationSetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).SESClient(ctx)

	configSetInput := &ses.DescribeConfigurationSetInput{
		ConfigurationSetName: aws.String(d.Id()),
		ConfigurationSetAttributeNames: []awstypes.ConfigurationSetAttribute{
			awstypes.ConfigurationSetAttributeDeliveryOptions,
			awstypes.ConfigurationSetAttributeReputationOptions,
			awstypes.ConfigurationSetAttributeTrackingOptions,
		},
	}

	response, err := conn.DescribeConfigurationSet(ctx, configSetInput)

	if !d.IsNewResource() && errs.IsA[*awstypes.ConfigurationSetDoesNotExistException](err) {
		log.Printf("[WARN] SES Configuration Set (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading SES Configuration Set (%s): %s", d.Id(), err)
	}

	if err := d.Set("delivery_options", flattenConfigurationSetDeliveryOptions(response.DeliveryOptions)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting delivery_options: %s", err)
	}

	if err := d.Set("tracking_options", flattenConfigurationSetTrackingOptions(response.TrackingOptions)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting tracking_options: %s", err)
	}

	d.Set(names.AttrName, response.ConfigurationSet.Name)

	repOpts := response.ReputationOptions
	if repOpts != nil {
		d.Set("reputation_metrics_enabled", repOpts.ReputationMetricsEnabled)
		d.Set("sending_enabled", repOpts.SendingEnabled)
		d.Set("last_fresh_start", aws.ToTime(repOpts.LastFreshStart).Format(time.RFC3339))
	}

	arn := arn.ARN{
		Partition: meta.(*conns.AWSClient).Partition,
		Service:   "ses",
		Region:    meta.(*conns.AWSClient).Region,
		AccountID: meta.(*conns.AWSClient).AccountID,
		Resource:  fmt.Sprintf("configuration-set/%s", d.Id()),
	}.String()
	d.Set(names.AttrARN, arn)

	return diags
}

func resourceConfigurationSetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).SESClient(ctx)

	if d.HasChange("delivery_options") {
		input := &ses.PutConfigurationSetDeliveryOptionsInput{
			ConfigurationSetName: aws.String(d.Id()),
			DeliveryOptions:      expandConfigurationSetDeliveryOptions(d.Get("delivery_options").([]interface{})),
		}

		_, err := conn.PutConfigurationSetDeliveryOptions(ctx, input)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating SES configuration set (%s) delivery options: %s", d.Id(), err)
		}
	}

	if d.HasChange("reputation_metrics_enabled") {
		input := &ses.UpdateConfigurationSetReputationMetricsEnabledInput{
			ConfigurationSetName: aws.String(d.Id()),
			Enabled:              d.Get("reputation_metrics_enabled").(bool),
		}

		_, err := conn.UpdateConfigurationSetReputationMetricsEnabled(ctx, input)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating SES configuration set (%s) reputation metrics enabled: %s", d.Id(), err)
		}
	}

	if d.HasChange("sending_enabled") {
		input := &ses.UpdateConfigurationSetSendingEnabledInput{
			ConfigurationSetName: aws.String(d.Id()),
			Enabled:              d.Get("sending_enabled").(bool),
		}

		_, err := conn.UpdateConfigurationSetSendingEnabled(ctx, input)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating SES configuration set (%s) reputation metrics enabled: %s", d.Id(), err)
		}
	}

	if d.HasChange("tracking_options") {
		input := &ses.UpdateConfigurationSetTrackingOptionsInput{
			ConfigurationSetName: aws.String(d.Id()),
			TrackingOptions:      expandConfigurationSetTrackingOptions(d.Get("tracking_options").([]interface{})),
		}

		_, err := conn.UpdateConfigurationSetTrackingOptions(ctx, input)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating SES configuration set (%s) tracking options: %s", d.Id(), err)
		}
	}

	return append(diags, resourceConfigurationSetRead(ctx, d, meta)...)
}

func resourceConfigurationSetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).SESClient(ctx)

	log.Printf("[DEBUG] SES Delete Configuration Rule Set: %s", d.Id())
	input := &ses.DeleteConfigurationSetInput{
		ConfigurationSetName: aws.String(d.Id()),
	}

	if _, err := conn.DeleteConfigurationSet(ctx, input); err != nil {
		if !errs.IsA[*awstypes.ConfigurationSetDoesNotExistException](err) {
			return sdkdiag.AppendErrorf(diags, "deleting SES Configuration Set (%s): %s", d.Id(), err)
		}
	}

	return diags
}

func expandConfigurationSetDeliveryOptions(l []interface{}) *awstypes.DeliveryOptions {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}

	options := &awstypes.DeliveryOptions{}

	if v, ok := tfMap["tls_policy"].(string); ok && v != "" {
		options.TlsPolicy = awstypes.TlsPolicy(v)
	}

	return options
}

func flattenConfigurationSetDeliveryOptions(options *awstypes.DeliveryOptions) []interface{} {
	if options == nil {
		return nil
	}

	m := map[string]interface{}{
		"tls_policy": string(options.TlsPolicy),
	}

	return []interface{}{m}
}

func expandConfigurationSetTrackingOptions(l []interface{}) *awstypes.TrackingOptions {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}

	options := &awstypes.TrackingOptions{}

	if v, ok := tfMap["custom_redirect_domain"].(string); ok && v != "" {
		options.CustomRedirectDomain = aws.String(v)
	}

	return options
}

func flattenConfigurationSetTrackingOptions(options *awstypes.TrackingOptions) []interface{} {
	if options == nil {
		return nil
	}

	m := map[string]interface{}{
		"custom_redirect_domain": aws.ToString(options.CustomRedirectDomain),
	}

	return []interface{}{m}
}
