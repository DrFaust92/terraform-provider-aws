package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
	tfec2 "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/ec2"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/ec2/waiter"
)

func resourceAwsEc2ClientVpnEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2ClientVpnEndpointCreate,
		Read:   resourceAwsEc2ClientVpnEndpointRead,
		Delete: resourceAwsEc2ClientVpnEndpointDelete,
		Update: resourceAwsEc2ClientVpnEndpointUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		CustomizeDiff: SetTagsDiff,

		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"client_cidr_block": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsCIDR,
			},
			"dns_servers": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"server_certificate_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},
			"split_tunnel": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"transport_protocol": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      ec2.TransportProtocolUdp,
				ValidateFunc: validation.StringInSlice(ec2.TransportProtocol_Values(), false),
			},
			"authentication_options": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 2,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice(ec2.ClientVpnAuthenticationType_Values(), false),
						},
						"saml_provider_arn": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateArn,
						},
						"active_directory_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"root_certificate_chain_arn": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},
			"connection_log_options": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cloudwatch_log_group": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"cloudwatch_log_stream": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsEc2ClientVpnEndpointCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	req := &ec2.CreateClientVpnEndpointInput{
		ClientCidrBlock:      aws.String(d.Get("client_cidr_block").(string)),
		ServerCertificateArn: aws.String(d.Get("server_certificate_arn").(string)),
		TransportProtocol:    aws.String(d.Get("transport_protocol").(string)),
		SplitTunnel:          aws.Bool(d.Get("split_tunnel").(bool)),
		TagSpecifications:    ec2TagSpecificationsFromKeyValueTags(tags, ec2.ResourceTypeClientVpnEndpoint),
	}

	if v, ok := d.GetOk("description"); ok {
		req.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("dns_servers"); ok {
		req.DnsServers = expandStringSet(v.(*schema.Set))
	}

	if v, ok := d.GetOk("authentication_options"); ok {
		authOptions := v.([]interface{})
		authRequests := make([]*ec2.ClientVpnAuthenticationRequest, 0, len(authOptions))

		for _, authOpt := range authOptions {
			auth := authOpt.(map[string]interface{})

			authReq := expandEc2ClientVpnAuthenticationRequest(auth)
			authRequests = append(authRequests, authReq)
		}
		req.AuthenticationOptions = authRequests
	}

	if v, ok := d.GetOk("connection_log_options"); ok {
		req.ConnectionLogOptions = expandEc2ClientVpnEndpointConnectionLogOptions(v.([]interface{}))
	}

	if v, ok := d.GetOk("vpc_id"); ok {
		req.VpcId = aws.String(v.(string))
	}

	resp, err := conn.CreateClientVpnEndpoint(req)

	if err != nil {
		return fmt.Errorf("Error creating Client VPN endpoint: %w", err)
	}

	d.SetId(aws.StringValue(resp.ClientVpnEndpointId))

	return resourceAwsEc2ClientVpnEndpointRead(d, meta)
}

func resourceAwsEc2ClientVpnEndpointRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	result, err := conn.DescribeClientVpnEndpoints(&ec2.DescribeClientVpnEndpointsInput{
		ClientVpnEndpointIds: []*string{aws.String(d.Id())},
	})

	if isAWSErr(err, tfec2.ErrCodeClientVpnAssociationIdNotFound, "") || isAWSErr(err, tfec2.ErrCodeClientVpnEndpointIdNotFound, "") {
		log.Printf("[WARN] EC2 Client VPN Endpoint (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error reading Client VPN endpoint: %w", err)
	}

	vpnEndpoints := result.ClientVpnEndpoints
	if result == nil || len(vpnEndpoints) == 0 || vpnEndpoints[0] == nil {
		log.Printf("[WARN] EC2 Client VPN Endpoint (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	vpnEndpoint := vpnEndpoints[0]
	vpnEndpointStatus := vpnEndpoint.Status
	if vpnEndpointStatus != nil && aws.StringValue(vpnEndpointStatus.Code) == ec2.ClientVpnEndpointStatusCodeDeleted {
		log.Printf("[WARN] EC2 Client VPN Endpoint (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("description", vpnEndpoint.Description)
	d.Set("client_cidr_block", vpnEndpoint.ClientCidrBlock)
	d.Set("server_certificate_arn", vpnEndpoint.ServerCertificateArn)
	d.Set("transport_protocol", vpnEndpoint.TransportProtocol)
	d.Set("dns_name", vpnEndpoint.DnsName)
	d.Set("dns_servers", flattenStringSet(vpnEndpoint.DnsServers))
	d.Set("vpc_id", vpnEndpoint.VpcId)

	if vpnEndpointStatus != nil {
		d.Set("status", vpnEndpointStatus.Code)
	}
	d.Set("split_tunnel", vpnEndpoint.SplitTunnel)

	err = d.Set("authentication_options", flattenAuthOptsConfig(vpnEndpoint.AuthenticationOptions))
	if err != nil {
		return fmt.Errorf("error setting authentication_options: %w", err)
	}

	err = d.Set("connection_log_options", flattenConnLoggingConfig(vpnEndpoint.ConnectionLogOptions))
	if err != nil {
		return fmt.Errorf("error setting connection_log_options: %w", err)
	}

	tags := keyvaluetags.Ec2KeyValueTags(vpnEndpoint.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   ec2.ServiceName,
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("client-vpn-endpoint/%s", d.Id()),
	}.String()
	d.Set("arn", arn)

	return nil
}

func resourceAwsEc2ClientVpnEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	err := deleteClientVpnEndpoint(conn, d.Id())
	if err != nil {
		return fmt.Errorf("error deleting Client VPN endpoint: %w", err)
	}

	return nil
}

func resourceAwsEc2ClientVpnEndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.ModifyClientVpnEndpointInput{
		ClientVpnEndpointId: aws.String(d.Id()),
	}

	if d.HasChange("description") {
		req.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("dns_servers") {
		dnsValue := expandStringSet(d.Get("dns_servers").(*schema.Set))
		var enabledValue *bool

		if len(dnsValue) > 0 {
			enabledValue = aws.Bool(true)
		} else {
			enabledValue = aws.Bool(false)
		}

		dnsMod := &ec2.DnsServersOptionsModifyStructure{
			CustomDnsServers: dnsValue,
			Enabled:          enabledValue,
		}
		req.DnsServers = dnsMod
	}

	if d.HasChange("server_certificate_arn") {
		req.ServerCertificateArn = aws.String(d.Get("server_certificate_arn").(string))
	}

	if d.HasChange("split_tunnel") {
		req.SplitTunnel = aws.Bool(d.Get("split_tunnel").(bool))
	}

	if d.HasChange("vpc_id") {
		req.VpcId = aws.String(d.Get("vpc_id").(string))
	}

	if d.HasChange("connection_log_options") {
		if v, ok := d.GetOk("connection_log_options"); ok {
			req.ConnectionLogOptions = expandEc2ClientVpnEndpointConnectionLogOptions(v.([]interface{}))
		}
	}

	if _, err := conn.ModifyClientVpnEndpoint(req); err != nil {
		return fmt.Errorf("Error modifying Client VPN endpoint: %w", err)
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")
		if err := keyvaluetags.Ec2UpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating EC2 Client VPN Endpoint (%s) tags: %w", d.Id(), err)
		}
	}

	return resourceAwsEc2ClientVpnEndpointRead(d, meta)
}

func flattenConnLoggingConfig(lopts *ec2.ConnectionLogResponseOptions) []map[string]interface{} {
	m := make(map[string]interface{})
	if lopts.CloudwatchLogGroup != nil {
		m["cloudwatch_log_group"] = aws.StringValue(lopts.CloudwatchLogGroup)
	}
	if lopts.CloudwatchLogStream != nil {
		m["cloudwatch_log_stream"] = aws.StringValue(lopts.CloudwatchLogStream)
	}
	m["enabled"] = aws.BoolValue(lopts.Enabled)
	return []map[string]interface{}{m}
}

func flattenAuthOptsConfig(aopts []*ec2.ClientVpnAuthentication) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(aopts))
	for _, aopt := range aopts {
		r := map[string]interface{}{
			"type": aws.StringValue(aopt.Type),
		}
		if aopt.MutualAuthentication != nil {
			r["root_certificate_chain_arn"] = aws.StringValue(aopt.MutualAuthentication.ClientRootCertificateChain)
		}
		if aopt.FederatedAuthentication != nil {
			r["saml_provider_arn"] = aws.StringValue(aopt.FederatedAuthentication.SamlProviderArn)
		}
		if aopt.ActiveDirectory != nil {
			r["active_directory_id"] = aws.StringValue(aopt.ActiveDirectory.DirectoryId)
		}
		result = append([]map[string]interface{}{r}, result...)
	}
	return result
}

func expandEc2ClientVpnEndpointConnectionLogOptions(l []interface{}) *ec2.ConnectionLogOptions {
	if len(l) == 0 || l[0] == nil {
		return &ec2.ConnectionLogOptions{}
	}

	m := l[0].(map[string]interface{})
	enabled := m["enabled"].(bool)
	connLogReq := &ec2.ConnectionLogOptions{
		Enabled: aws.Bool(enabled),
	}

	if enabled && m["cloudwatch_log_group"].(string) != "" {
		connLogReq.CloudwatchLogGroup = aws.String(m["cloudwatch_log_group"].(string))
	}

	if enabled && m["cloudwatch_log_stream"].(string) != "" {
		connLogReq.CloudwatchLogStream = aws.String(m["cloudwatch_log_stream"].(string))
	}

	return connLogReq
}

func expandEc2ClientVpnAuthenticationRequest(data map[string]interface{}) *ec2.ClientVpnAuthenticationRequest {
	req := &ec2.ClientVpnAuthenticationRequest{
		Type: aws.String(data["type"].(string)),
	}

	if data["type"].(string) == ec2.ClientVpnAuthenticationTypeCertificateAuthentication {
		req.MutualAuthentication = &ec2.CertificateAuthenticationRequest{
			ClientRootCertificateChainArn: aws.String(data["root_certificate_chain_arn"].(string)),
		}
	}

	if data["type"].(string) == ec2.ClientVpnAuthenticationTypeDirectoryServiceAuthentication {
		req.ActiveDirectory = &ec2.DirectoryServiceAuthenticationRequest{
			DirectoryId: aws.String(data["active_directory_id"].(string)),
		}
	}

	if data["type"].(string) == ec2.ClientVpnAuthenticationTypeFederatedAuthentication {
		req.FederatedAuthentication = &ec2.FederatedAuthenticationRequest{
			SAMLProviderArn: aws.String(data["saml_provider_arn"].(string)),
		}
	}

	return req
}

func deleteClientVpnEndpoint(conn *ec2.EC2, endpointID string) error {
	_, err := conn.DeleteClientVpnEndpoint(&ec2.DeleteClientVpnEndpointInput{
		ClientVpnEndpointId: aws.String(endpointID),
	})
	if isAWSErr(err, tfec2.ErrCodeClientVpnEndpointIdNotFound, "") {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = waiter.ClientVpnEndpointDeleted(conn, endpointID)

	return err
}
