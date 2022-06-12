package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudflareWAFOverride() *schema.Resource {
	return &schema.Resource{
		Schema:        resourceCloudflareWAFOverrideSchema(),
		CreateContext: resourceCloudflareWAFOverrideCreate,
		ReadContext:   resourceCloudflareWAFOverrideRead,
		UpdateContext: resourceCloudflareWAFOverrideUpdate,
		DeleteContext: resourceCloudflareWAFOverrideDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceCloudflareWAFOverrideImport,
		},
	}
}

func resourceCloudflareWAFOverrideRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	zoneID := d.Get("zone_id").(string)

	override, err := client.WAFOverride(ctx, zoneID, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "wafuriconfig.api.not_found") {
			tflog.Info(ctx, fmt.Sprintf("WAF override %s no longer exists", d.Id()))
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to find WAF override %s: %w", d.Id(), err))
	}

	d.Set("zone_id", zoneID)
	d.Set("urls", override.URLs)
	d.Set("paused", override.Paused)
	d.Set("description", override.Description)
	d.Set("priority", override.Priority)

	if len(override.Rules) != 0 {
		d.Set("rules", override.Rules)
	}

	if len(override.Groups) != 0 {
		d.Set("groups", override.Groups)
	}

	if len(override.RewriteAction) != 0 {
		d.Set("rewrite_action", override.RewriteAction)
	}

	d.Set("override_id", override.ID)

	return nil
}

func resourceCloudflareWAFOverrideCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	zoneID := d.Get("zone_id").(string)
	newOverride, _ := buildWAFOverride(d)

	override, err := client.CreateWAFOverride(ctx, zoneID, newOverride)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create WAF override: %w", err))
	}

	d.SetId(override.ID)

	return resourceCloudflareWAFOverrideRead(ctx, d, meta)
}

func resourceCloudflareWAFOverrideUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	zoneID := d.Get("zone_id").(string)
	overrideID := d.Get("override_id").(string)
	updatedOverride, _ := buildWAFOverride(d)

	_, err := client.UpdateWAFOverride(ctx, zoneID, overrideID, updatedOverride)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update WAF override: %w", err))
	}

	return resourceCloudflareWAFOverrideRead(ctx, d, meta)
}

func resourceCloudflareWAFOverrideDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	overrideID := d.Get("override_id").(string)
	zoneID := d.Get("zone_id").(string)

	err := client.DeleteWAFOverride(ctx, zoneID, overrideID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete WAF override ID %s: %w", overrideID, err))
	}

	return resourceCloudflareWAFOverrideRead(ctx, d, meta)
}

func resourceCloudflareWAFOverrideImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	idAttr := strings.SplitN(d.Id(), "/", 2)

	if len(idAttr) != 2 {
		return nil, fmt.Errorf("invalid id (\"%s\") specified, should be in format \"zoneID/WAFOverrideID\"", d.Id())
	}

	zoneID, WAFOverrideID := idAttr[0], idAttr[1]

	tflog.Debug(ctx, fmt.Sprintf("Importing WAF override: id %s for zone %s", WAFOverrideID, zoneID))

	d.Set("zone_id", zoneID)
	d.Set("override_id", WAFOverrideID)
	d.SetId(WAFOverrideID)

	resourceCloudflareWAFOverrideRead(ctx, d, meta)

	return []*schema.ResourceData{d}, nil
}

// buildWAFOverride centralises the creation of a WAFOverride struct which can
// be reused between Create and Update methods to ensure consistent building of
// the values.
func buildWAFOverride(d *schema.ResourceData) (cloudflare.WAFOverride, error) {
	builtOverride := cloudflare.WAFOverride{}

	urls := d.Get("urls").([]interface{})
	for _, url := range urls {
		builtOverride.URLs = append(builtOverride.URLs, url.(string))
	}

	if rules, ok := d.GetOk("rules"); ok {
		rulesMap := make(map[string]string)
		for ruleID, state := range rules.(map[string]interface{}) {
			rulesMap[ruleID] = state.(string)
		}
		builtOverride.Rules = rulesMap
	}

	if pausedValue, ok := d.GetOk("paused"); ok {
		builtOverride.Paused = pausedValue.(bool)
	}

	if descriptionValue, ok := d.GetOk("description"); ok {
		builtOverride.Description = descriptionValue.(string)
	}

	if priorityValue, ok := d.GetOk("priority"); ok {
		builtOverride.Priority = priorityValue.(int)
	}

	if groupsValue, ok := d.GetOk("groups"); ok {
		groupsMap := make(map[string]string)
		for groupID, state := range groupsValue.(map[string]interface{}) {
			groupsMap[groupID] = state.(string)
		}
		builtOverride.Groups = groupsMap
	}

	if rewriteActionValue, ok := d.GetOk("rewrite_action"); ok {
		rewriteActions := make(map[string]string)
		for rewriteOriginal, rewriteWant := range rewriteActionValue.(map[string]interface{}) {
			rewriteActions[rewriteOriginal] = rewriteWant.(string)
		}
		builtOverride.RewriteAction = rewriteActions
	}

	return builtOverride, nil
}
