package cloudflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

func resourceCloudflareBYOIPPrefix() *schema.Resource {
	return &schema.Resource{
		Schema: resourceCloudflareBYOIPPrefixSchema(),
		Create: resourceCloudflareBYOIPPrefixCreate,
		Read:   resourceCloudflareBYOIPPrefixRead,
		Update: resourceCloudflareBYOIPPrefixUpdate,
		Delete: resourceCloudflareBYOIPPrefixDelete,
		Importer: &schema.ResourceImporter{
			State: resourceCloudflareBYOIPPrefixImport,
		},
	}
}

func resourceCloudflareBYOIPPrefixCreate(d *schema.ResourceData, meta interface{}) error {
	prefixID := d.Get("prefix_id")
	d.SetId(prefixID.(string))

	if err := resourceCloudflareBYOIPPrefixUpdate(d, meta); err != nil {
		return err
	}

	return resourceCloudflareBYOIPPrefixRead(d, meta)
}

func resourceCloudflareBYOIPPrefixImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	prefixID := d.Id()
	d.Set("prefix_id", prefixID)

	resourceCloudflareBYOIPPrefixRead(d, meta)

	return []*schema.ResourceData{d}, nil
}

func resourceCloudflareBYOIPPrefixRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	accountID := d.Get("account_id").(string)

	prefix, err := client.GetPrefix(context.Background(), accountID, d.Id())
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error reading IP prefix information for %q", d.Id()))
	}

	d.Set("description", prefix.Description)

	advertisementStatus, err := client.GetAdvertisementStatus(context.Background(), accountID, d.Id())
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error reading advertisement status of IP prefix for %q", d.Id()))
	}

	d.Set("advertisement", stringFromBool(advertisementStatus.Advertised))

	return nil
}

func resourceCloudflareBYOIPPrefixUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	accountID := d.Get("account_id").(string)

	if _, ok := d.GetOk("description"); ok && d.HasChange("description") {
		if _, err := client.UpdatePrefixDescription(context.Background(), accountID, d.Id(), d.Get("description").(string)); err != nil {
			return errors.Wrap(err, fmt.Sprintf("cannot update prefix description for %q", d.Id()))
		}
	}

	if _, ok := d.GetOk("advertisement"); ok && d.HasChange("advertisement") {
		if _, err := client.UpdateAdvertisementStatus(context.Background(), accountID, d.Id(), boolFromString(d.Get("advertisement").(string))); err != nil {
			return errors.Wrap(err, fmt.Sprintf("cannot update prefix advertisement status for %q", d.Id()))
		}
	}

	return nil
}

// Deletion of prefixes is not really supported, so we keep this as a dummy
func resourceCloudflareBYOIPPrefixDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
