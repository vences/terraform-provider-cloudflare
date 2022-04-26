package cloudflare

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudflareAccessPolicy() *schema.Resource {
	return &schema.Resource{
		Schema: resourceCloudflareAccessPolicySchema(),
		Create: resourceCloudflareAccessPolicyCreate,
		Read:   resourceCloudflareAccessPolicyRead,
		Update: resourceCloudflareAccessPolicyUpdate,
		Delete: resourceCloudflareAccessPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: resourceCloudflareAccessPolicyImport,
		},
	}
}

func apiAccessPolicyApprovalGroupToSchema(approvalGroup cloudflare.AccessApprovalGroup) map[string]interface{} {
	data := make(map[string]interface{})
	data["approvals_needed"] = approvalGroup.ApprovalsNeeded

	if approvalGroup.EmailAddresses != nil {
		data["email_addresses"] = approvalGroup.EmailAddresses
	}

	if approvalGroup.EmailListUuid != "" {
		data["email_list_uuid"] = approvalGroup.EmailListUuid
	}
	return data
}

func schemaAccessPolicyApprovalGroupToAPI(data map[string]interface{}) cloudflare.AccessApprovalGroup {
	var approvalGroup cloudflare.AccessApprovalGroup

	approvalGroup.ApprovalsNeeded, _ = data["approvals_needed"].(int)
	approvalGroup.EmailListUuid, _ = data["email_list_uuid"].(string)

	if emailAddresses, ok := data["email_addresses"].([]interface{}); ok {
		approvalGroup.EmailAddresses = expandInterfaceToStringList(emailAddresses)
	}

	return approvalGroup
}

func resourceCloudflareAccessPolicyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	appID := d.Get("application_id").(string)

	identifier, err := initIdentifier(d)
	if err != nil {
		return err
	}

	var accessPolicy cloudflare.AccessPolicy
	if identifier.Type == AccountType {
		accessPolicy, err = client.AccessPolicy(context.Background(), identifier.Value, appID, d.Id())
	} else {
		accessPolicy, err = client.ZoneLevelAccessPolicy(context.Background(), identifier.Value, appID, d.Id())
	}
	if err != nil {
		if strings.Contains(err.Error(), "HTTP status 404") {
			log.Printf("[INFO] Access Policy %s no longer exists", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error finding Access Policy %q: %s", d.Id(), err)
	}

	d.Set("name", accessPolicy.Name)
	d.Set("decision", accessPolicy.Decision)
	d.Set("precedence", accessPolicy.Precedence)

	if err := d.Set("require", TransformAccessGroupForSchema(accessPolicy.Require)); err != nil {
		return fmt.Errorf("failed to set require attribute: %s", err)
	}

	if err := d.Set("exclude", TransformAccessGroupForSchema(accessPolicy.Exclude)); err != nil {
		return fmt.Errorf("failed to set exclude attribute: %s", err)
	}

	if err := d.Set("include", TransformAccessGroupForSchema(accessPolicy.Include)); err != nil {
		return fmt.Errorf("failed to set include attribute: %s", err)
	}

	if accessPolicy.PurposeJustificationRequired != nil {
		d.Set("purpose_justification_required", accessPolicy.PurposeJustificationRequired)
	}

	if accessPolicy.PurposeJustificationPrompt != nil {
		d.Set("purpose_justification_prompt", accessPolicy.PurposeJustificationPrompt)
	}

	if accessPolicy.ApprovalRequired != nil {
		d.Set("approval_required", accessPolicy.ApprovalRequired)
	}

	if len(accessPolicy.ApprovalGroups) != 0 {
		approvalGroups := make([]map[string]interface{}, 0, len(accessPolicy.ApprovalGroups))
		for _, apiApprovalGroup := range accessPolicy.ApprovalGroups {
			approvalGroups = append(approvalGroups, apiAccessPolicyApprovalGroupToSchema(apiApprovalGroup))
		}
		if err := d.Set("approval_group", approvalGroups); err != nil {
			return fmt.Errorf("failed to set approval_group attribute: %s", err)
		}
	}

	return nil
}

func resourceCloudflareAccessPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	appID := d.Get("application_id").(string)
	newAccessPolicy := cloudflare.AccessPolicy{
		Name:       d.Get("name").(string),
		Precedence: d.Get("precedence").(int),
		Decision:   d.Get("decision").(string),
	}

	newAccessPolicy = appendConditionalAccessPolicyFields(newAccessPolicy, d)

	log.Printf("[DEBUG] Creating Cloudflare Access Policy from struct: %+v", newAccessPolicy)

	identifier, err := initIdentifier(d)
	if err != nil {
		return err
	}

	var accessPolicy cloudflare.AccessPolicy
	if identifier.Type == AccountType {
		accessPolicy, err = client.CreateAccessPolicy(context.Background(), identifier.Value, appID, newAccessPolicy)
	} else {
		accessPolicy, err = client.CreateZoneLevelAccessPolicy(context.Background(), identifier.Value, appID, newAccessPolicy)
	}
	if err != nil {
		return fmt.Errorf("error creating Access Policy for ID %q: %s", accessPolicy.ID, err)
	}

	d.SetId(accessPolicy.ID)

	return nil
}

func resourceCloudflareAccessPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	appID := d.Get("application_id").(string)
	updatedAccessPolicy := cloudflare.AccessPolicy{
		Name:       d.Get("name").(string),
		Precedence: d.Get("precedence").(int),
		Decision:   d.Get("decision").(string),
		ID:         d.Id(),
	}

	updatedAccessPolicy = appendConditionalAccessPolicyFields(updatedAccessPolicy, d)

	log.Printf("[DEBUG] Updating Cloudflare Access Policy from struct: %+v", updatedAccessPolicy)

	identifier, err := initIdentifier(d)
	if err != nil {
		return err
	}

	var accessPolicy cloudflare.AccessPolicy
	if identifier.Type == AccountType {
		accessPolicy, err = client.UpdateAccessPolicy(context.Background(), identifier.Value, appID, updatedAccessPolicy)
	} else {
		accessPolicy, err = client.UpdateZoneLevelAccessPolicy(context.Background(), identifier.Value, appID, updatedAccessPolicy)
	}
	if err != nil {
		return fmt.Errorf("error updating Access Policy for ID %q: %s", d.Id(), err)
	}

	if accessPolicy.ID == "" {
		return fmt.Errorf("failed to find Access Policy ID in update response; resource was empty")
	}

	return resourceCloudflareAccessPolicyRead(d, meta)
}

func resourceCloudflareAccessPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	appID := d.Get("application_id").(string)

	log.Printf("[DEBUG] Deleting Cloudflare Access Policy using ID: %s", d.Id())

	identifier, err := initIdentifier(d)
	if err != nil {
		return err
	}

	if identifier.Type == AccountType {
		err = client.DeleteAccessPolicy(context.Background(), identifier.Value, appID, d.Id())
	} else {
		err = client.DeleteZoneLevelAccessPolicy(context.Background(), identifier.Value, appID, d.Id())
	}
	if err != nil {
		return fmt.Errorf("error deleting Access Policy for ID %q: %s", d.Id(), err)
	}

	resourceCloudflareAccessPolicyRead(d, meta)

	return nil
}

func resourceCloudflareAccessPolicyImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	attributes := strings.SplitN(d.Id(), "/", 4)

	if len(attributes) != 4 {
		return nil, fmt.Errorf(
			"invalid id (%q) specified, should be in format %q or %q",
			d.Id(),
			"account/accountID/accessApplicationID/accessPolicyID",
			"zone/zoneID/accessApplicationID/accessPolicyID",
		)
	}

	identifierType, identifierID, accessAppID, accessPolicyID := attributes[0], attributes[1], attributes[2], attributes[3]

	log.Printf("[DEBUG] Importing Cloudflare Access Policy: %s %q, appID %q, accessPolicyID %q", identifierType, identifierID, accessAppID, accessPolicyID)

	//lintignore:R001
	d.Set(fmt.Sprintf("%s_id", identifierType), identifierID)
	d.Set("application_id", accessAppID)
	d.SetId(accessPolicyID)

	resourceCloudflareAccessPolicyRead(d, meta)

	return []*schema.ResourceData{d}, nil
}

// appendConditionalAccessPolicyFields determines which of the
// conditional policy enforcement fields it should append to the
// AccessPolicy by iterating over the provided values and generating the
// correct structs.
func appendConditionalAccessPolicyFields(policy cloudflare.AccessPolicy, d *schema.ResourceData) cloudflare.AccessPolicy {
	exclude := d.Get("exclude").([]interface{})
	for _, value := range exclude {
		if value != nil {
			policy.Exclude = BuildAccessGroupCondition(value.(map[string]interface{}))
		}
	}

	require := d.Get("require").([]interface{})
	for _, value := range require {
		if value != nil {
			policy.Require = BuildAccessGroupCondition(value.(map[string]interface{}))
		}
	}

	include := d.Get("include").([]interface{})
	for _, value := range include {
		if value != nil {
			policy.Include = BuildAccessGroupCondition(value.(map[string]interface{}))
		}
	}

	purposeJustificationRequired := d.Get("purpose_justification_required").(bool)
	policy.PurposeJustificationRequired = &purposeJustificationRequired

	purposeJustificationPrompt := d.Get("purpose_justification_prompt").(string)
	policy.PurposeJustificationPrompt = &purposeJustificationPrompt

	approvalRequired := d.Get("approval_required").(bool)
	policy.ApprovalRequired = &approvalRequired

	approvalGroups := d.Get("approval_group").([]interface{})
	for _, approvalGroup := range approvalGroups {
		approvalGroupAsMap := approvalGroup.(map[string]interface{})
		policy.ApprovalGroups = append(policy.ApprovalGroups, schemaAccessPolicyApprovalGroupToAPI(approvalGroupAsMap))
	}

	return policy
}
