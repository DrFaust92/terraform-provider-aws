package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/fsx/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/fsx/waiter"
)

func resourceAwsFsxWindowsFileSystem() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsFsxWindowsFileSystemCreate,
		Read:   resourceAwsFsxWindowsFileSystemRead,
		Update: resourceAwsFsxWindowsFileSystemUpdate,
		Delete: resourceAwsFsxWindowsFileSystemDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("skip_final_backup", false)

				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(45 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"active_directory_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"self_managed_active_directory"},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"automatic_backup_retention_days": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      7,
				ValidateFunc: validation.IntBetween(0, 90),
			},
			"copy_tags_to_backups": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"daily_automatic_backup_start_time": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(5, 5),
					validation.StringMatch(regexp.MustCompile(`^([01]\d|2[0-3]):?([0-5]\d)$`), "must be in the format HH:MM"),
				),
			},
			"aliases": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 50,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringLenBetween(4, 253),
				},
			},
			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"network_interface_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 50,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"self_managed_active_directory": {
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"active_directory_id"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dns_ips": {
							Type:     schema.TypeSet,
							Required: true,
							MinItems: 1,
							MaxItems: 2,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsIPAddress,
							},
						},
						"domain_name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 255),
						},
						"file_system_administrators_group": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "Domain Admins",
							ValidateFunc: validation.StringLenBetween(1, 256),
						},
						"organizational_unit_distinguished_name": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringLenBetween(1, 2000),
						},
						"password": {
							Type:         schema.TypeString,
							Required:     true,
							Sensitive:    true,
							ValidateFunc: validation.StringLenBetween(1, 256),
						},
						"username": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 256),
						},
					},
				},
			},
			"skip_final_backup": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"storage_capacity": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(32, 65536),
			},
			"subnet_ids": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MinItems: 1,
				MaxItems: 50,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tags": tagsSchema(),
			"throughput_capacity": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(8, 2048),
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"weekly_maintenance_start_time": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(7, 7),
					validation.StringMatch(regexp.MustCompile(`^[1-7]:([01]\d|2[0-3]):?([0-5]\d)$`), "must be in the format d:HH:MM"),
				),
			},
			"deployment_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      fsx.WindowsDeploymentTypeSingleAz1,
				ValidateFunc: validation.StringInSlice(fsx.WindowsDeploymentType_Values(), false),
			},
			"preferred_subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"preferred_file_server_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"remote_administration_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      fsx.StorageTypeSsd,
				ValidateFunc: validation.StringInSlice(fsx.StorageType_Values(), false),
			},
		},
	}
}

func resourceAwsFsxWindowsFileSystemCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).fsxconn

	input := &fsx.CreateFileSystemInput{
		ClientRequestToken: aws.String(resource.UniqueId()),
		FileSystemType:     aws.String(fsx.FileSystemTypeWindows),
		StorageCapacity:    aws.Int64(int64(d.Get("storage_capacity").(int))),
		SubnetIds:          expandStringList(d.Get("subnet_ids").([]interface{})),
		WindowsConfiguration: &fsx.CreateFileSystemWindowsConfiguration{
			AutomaticBackupRetentionDays: aws.Int64(int64(d.Get("automatic_backup_retention_days").(int))),
			CopyTagsToBackups:            aws.Bool(d.Get("copy_tags_to_backups").(bool)),
			ThroughputCapacity:           aws.Int64(int64(d.Get("throughput_capacity").(int))),
		},
	}

	if v, ok := d.GetOk("active_directory_id"); ok {
		input.WindowsConfiguration.ActiveDirectoryId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("deployment_type"); ok {
		input.WindowsConfiguration.DeploymentType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("preferred_subnet_id"); ok {
		input.WindowsConfiguration.PreferredSubnetId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("daily_automatic_backup_start_time"); ok {
		input.WindowsConfiguration.DailyAutomaticBackupStartTime = aws.String(v.(string))
	}

	if v, ok := d.GetOk("aliases"); ok {
		input.WindowsConfiguration.Aliases = expandStringSet(v.(*schema.Set))
	}

	if v, ok := d.GetOk("kms_key_id"); ok {
		input.KmsKeyId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("security_group_ids"); ok {
		input.SecurityGroupIds = expandStringSet(v.(*schema.Set))
	}

	if v, ok := d.GetOk("self_managed_active_directory"); ok {
		input.WindowsConfiguration.SelfManagedActiveDirectoryConfiguration = expandFsxSelfManagedActiveDirectoryConfigurationCreate(v.([]interface{}))
	}

	if v, ok := d.GetOk("tags"); ok {
		input.Tags = keyvaluetags.New(v.(map[string]interface{})).IgnoreAws().FsxTags()
	}

	if v, ok := d.GetOk("weekly_maintenance_start_time"); ok {
		input.WindowsConfiguration.WeeklyMaintenanceStartTime = aws.String(v.(string))
	}

	if v, ok := d.GetOk("storage_type"); ok {
		input.StorageType = aws.String(v.(string))
	}

	result, err := conn.CreateFileSystem(input)
	if err != nil {
		return fmt.Errorf("Error creating FSx filesystem: %w", err)
	}

	d.SetId(aws.StringValue(result.FileSystem.FileSystemId))

	log.Println("[DEBUG] Waiting for filesystem to become available")

	if _, err := waiter.FileSystemAvailable(conn, d.Id(), d.Timeout(schema.TimeoutCreate)); err != nil {
		return fmt.Errorf("error waiting for filesystem (%s) to become available: %w", d.Id(), err)
	}

	return resourceAwsFsxWindowsFileSystemRead(d, meta)
}

func resourceAwsFsxWindowsFileSystemUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).fsxconn

	if d.HasChange("tags") {
		o, n := d.GetChange("tags")

		if err := keyvaluetags.FsxUpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating FSx Windows File System (%s) tags: %s", d.Get("arn").(string), err)
		}
	}

	requestUpdate := false
	input := &fsx.UpdateFileSystemInput{
		ClientRequestToken:   aws.String(resource.UniqueId()),
		FileSystemId:         aws.String(d.Id()),
		WindowsConfiguration: &fsx.UpdateFileSystemWindowsConfiguration{},
	}

	if d.HasChange("automatic_backup_retention_days") {
		input.WindowsConfiguration.AutomaticBackupRetentionDays = aws.Int64(int64(d.Get("automatic_backup_retention_days").(int)))
		requestUpdate = true
	}

	if d.HasChange("throughput_capacity") {
		input.WindowsConfiguration.ThroughputCapacity = aws.Int64(int64(d.Get("throughput_capacity").(int)))
		requestUpdate = true
	}

	if d.HasChange("storage_capacity") {
		input.StorageCapacity = aws.Int64(int64(d.Get("storage_capacity").(int)))
		requestUpdate = true
	}

	if d.HasChange("daily_automatic_backup_start_time") {
		input.WindowsConfiguration.DailyAutomaticBackupStartTime = aws.String(d.Get("daily_automatic_backup_start_time").(string))
		requestUpdate = true
	}

	if d.HasChange("self_managed_active_directory") {
		input.WindowsConfiguration.SelfManagedActiveDirectoryConfiguration = expandFsxSelfManagedActiveDirectoryConfigurationUpdate(d.Get("self_managed_active_directory").([]interface{}))
		requestUpdate = true
	}

	if d.HasChange("weekly_maintenance_start_time") {
		input.WindowsConfiguration.WeeklyMaintenanceStartTime = aws.String(d.Get("weekly_maintenance_start_time").(string))
		requestUpdate = true
	}

	if requestUpdate {
		_, err := conn.UpdateFileSystem(input)

		if err != nil {
			return fmt.Errorf("error updating FSx Windows File System (%s): %w", d.Id(), err)
		}

		if _, err := waiter.FileSystemAdministrativeActionsCompletedOrOptimizing(conn, d.Id(), d.Timeout(schema.TimeoutCreate)); err != nil {
			return fmt.Errorf("error waiting for filesystem (%s) to update: %w", d.Id(), err)
		}
	}

	if d.HasChange("aliases") {
		o, n := d.GetChange("aliases")

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		additionList := ns.Difference(os)
		if additionList.Len() > 0 {
			input := &fsx.AssociateFileSystemAliasesInput{
				FileSystemId: aws.String(d.Id()),
				Aliases:      expandStringSet(additionList),
			}

			_, err := conn.AssociateFileSystemAliases(input)

			if err != nil {
				return fmt.Errorf("error Associating FSx Windows File System aliases (%s): %w", d.Id(), err)
			}

			for _, aliasID := range additionList.List() {
				if _, err := waiter.FileSystemWindowsAliasAvailable(conn, d.Id(), aliasID.(string)); err != nil {
					return fmt.Errorf("error waiting for filesystem alias (%s) to be Available: %w", d.Id(), err)
				}
			}
		}

		removalList := os.Difference(ns)
		if removalList.Len() > 0 {
			input := &fsx.DisassociateFileSystemAliasesInput{
				FileSystemId: aws.String(d.Id()),
				Aliases:      expandStringSet(additionList),
			}

			_, err := conn.DisassociateFileSystemAliases(input)

			if err != nil {
				return fmt.Errorf("error disassociating FSx Windows File System aliases (%s): %w", d.Id(), err)
			}

			for _, aliasID := range removalList.List() {
				if _, err := waiter.FileSystemWindowsAliasDeleted(conn, d.Id(), aliasID.(string)); err != nil {
					return fmt.Errorf("error waiting for filesystem alias (%s) to be Deleted: %w", d.Id(), err)
				}
			}
		}
	}

	return resourceAwsFsxWindowsFileSystemRead(d, meta)
}

func resourceAwsFsxWindowsFileSystemRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).fsxconn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	filesystem, err := finder.FileSystemByID(conn, d.Id())

	if isAWSErr(err, fsx.ErrCodeFileSystemNotFound, "") {
		log.Printf("[WARN] FSx File System (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error reading FSx File System (%s): %s", d.Id(), err)
	}

	if filesystem == nil {
		log.Printf("[WARN] FSx File System (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	winConfig := filesystem.WindowsConfiguration

	if filesystem.LustreConfiguration != nil {
		return fmt.Errorf("expected FSx Windows File System, found FSx Lustre File System: %s", d.Id())
	}

	if winConfig == nil {
		return fmt.Errorf("error describing FSx Windows File System (%s): empty Windows configuration", d.Id())
	}

	d.Set("active_directory_id", winConfig.ActiveDirectoryId)
	d.Set("arn", filesystem.ResourceARN)
	d.Set("automatic_backup_retention_days", winConfig.AutomaticBackupRetentionDays)
	d.Set("copy_tags_to_backups", winConfig.CopyTagsToBackups)
	d.Set("daily_automatic_backup_start_time", winConfig.DailyAutomaticBackupStartTime)
	d.Set("deployment_type", winConfig.DeploymentType)
	d.Set("preferred_subnet_id", winConfig.PreferredSubnetId)
	d.Set("preferred_file_server_ip", winConfig.PreferredFileServerIp)
	d.Set("remote_administration_endpoint", winConfig.RemoteAdministrationEndpoint)
	d.Set("remote_administration_endpoint", winConfig.RemoteAdministrationEndpoint)
	d.Set("dns_name", filesystem.DNSName)
	d.Set("kms_key_id", filesystem.KmsKeyId)
	d.Set("storage_type", filesystem.StorageType)

	if err := d.Set("network_interface_ids", aws.StringValueSlice(filesystem.NetworkInterfaceIds)); err != nil {
		return fmt.Errorf("error setting network_interface_ids: %w", err)
	}

	d.Set("owner_id", filesystem.OwnerId)

	if err := d.Set("self_managed_active_directory", flattenFsxSelfManagedActiveDirectoryConfiguration(d, winConfig.SelfManagedActiveDirectoryConfiguration)); err != nil {
		return fmt.Errorf("error setting self_managed_active_directory: %w", err)
	}

	d.Set("storage_capacity", filesystem.StorageCapacity)

	if err := d.Set("subnet_ids", aws.StringValueSlice(filesystem.SubnetIds)); err != nil {
		return fmt.Errorf("error setting subnet_ids: %w", err)
	}

	if err := d.Set("tags", keyvaluetags.FsxKeyValueTags(filesystem.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	d.Set("throughput_capacity", winConfig.ThroughputCapacity)
	d.Set("vpc_id", filesystem.VpcId)
	d.Set("weekly_maintenance_start_time", winConfig.WeeklyMaintenanceStartTime)

	if winConfig.Aliases != nil {
		var aliases []*string
		for _, i := range winConfig.Aliases {
			aliases = append(aliases, i.Name)
		}
		if err := d.Set("aliases", flattenStringSet(aliases)); err != nil {
			return fmt.Errorf("error setting aliases: %w", err)
		}
	}

	return nil
}

func resourceAwsFsxWindowsFileSystemDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).fsxconn

	input := &fsx.DeleteFileSystemInput{
		ClientRequestToken: aws.String(resource.UniqueId()),
		FileSystemId:       aws.String(d.Id()),
		WindowsConfiguration: &fsx.DeleteFileSystemWindowsConfiguration{
			SkipFinalBackup: aws.Bool(d.Get("skip_final_backup").(bool)),
		},
	}

	_, err := conn.DeleteFileSystem(input)

	if isAWSErr(err, fsx.ErrCodeFileSystemNotFound, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error deleting FSx filesystem: %w", err)
	}

	log.Println("[DEBUG] Waiting for filesystem to delete")

	if _, err := waiter.FileSystemDeleted(conn, d.Id(), d.Timeout(schema.TimeoutCreate)); err != nil {
		return fmt.Errorf("error waiting for filesystem (%s) to be deleted: %w", d.Id(), err)
	}

	return nil
}

func expandFsxSelfManagedActiveDirectoryConfigurationCreate(l []interface{}) *fsx.SelfManagedActiveDirectoryConfiguration {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	data := l[0].(map[string]interface{})
	req := &fsx.SelfManagedActiveDirectoryConfiguration{
		DomainName: aws.String(data["domain_name"].(string)),
		DnsIps:     expandStringSet(data["dns_ips"].(*schema.Set)),
		Password:   aws.String(data["password"].(string)),
		UserName:   aws.String(data["username"].(string)),
	}

	if v, ok := data["file_system_administrators_group"]; ok && v.(string) != "" {
		req.FileSystemAdministratorsGroup = aws.String(v.(string))
	}

	if v, ok := data["organizational_unit_distinguished_name"]; ok && v.(string) != "" {
		req.OrganizationalUnitDistinguishedName = aws.String(v.(string))
	}

	return req
}

func expandFsxSelfManagedActiveDirectoryConfigurationUpdate(l []interface{}) *fsx.SelfManagedActiveDirectoryConfigurationUpdates {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	data := l[0].(map[string]interface{})
	req := &fsx.SelfManagedActiveDirectoryConfigurationUpdates{
		DnsIps:   expandStringSet(data["dns_ips"].(*schema.Set)),
		Password: aws.String(data["password"].(string)),
		UserName: aws.String(data["username"].(string)),
	}

	return req
}

func flattenFsxSelfManagedActiveDirectoryConfiguration(d *schema.ResourceData, adopts *fsx.SelfManagedActiveDirectoryAttributes) []map[string]interface{} {
	if adopts == nil {
		return []map[string]interface{}{}
	}

	// Since we are in a configuration block and the FSx API does not return
	// the password, we need to set the value if we can or Terraform will
	// show a difference for the argument from empty string to the value.
	// This is not a pattern that should be used normally.
	// See also: flattenEmrKerberosAttributes

	m := map[string]interface{}{
		"dns_ips":                                aws.StringValueSlice(adopts.DnsIps),
		"domain_name":                            aws.StringValue(adopts.DomainName),
		"file_system_administrators_group":       aws.StringValue(adopts.FileSystemAdministratorsGroup),
		"organizational_unit_distinguished_name": aws.StringValue(adopts.OrganizationalUnitDistinguishedName),
		"password":                               d.Get("self_managed_active_directory.0.password").(string),
		"username":                               aws.StringValue(adopts.UserName),
	}

	return []map[string]interface{}{m}
}
