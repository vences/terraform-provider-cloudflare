package cloudflare

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudflareWorkerCronTrigger() *schema.Resource {
	return &schema.Resource{
		Schema: resourceCloudflareWorkerCronTriggerSchema(),
		Create: resourceCloudflareWorkerCronTriggerUpdate,
		Read:   resourceCloudflareWorkerCronTriggerRead,
		Update: resourceCloudflareWorkerCronTriggerUpdate,
		Delete: resourceCloudflareWorkerCronTriggerDelete,
		Importer: &schema.ResourceImporter{
			State: resourceCloudflareWorkerCronTriggerImport,
		},
	}
}

// resourceCloudflareWorkerCronTriggerUpdate is used for creation and updates of
// Worker Cron Triggers as the remote API endpoint is shared uses HTTP PUT.
func resourceCloudflareWorkerCronTriggerUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	accountID := d.Get("account_id").(string)

	scriptName := d.Get("script_name").(string)

	_, err := client.UpdateWorkerCronTriggers(context.Background(), accountID, scriptName, transformSchemaToWorkerCronTriggerStruct(d))
	if err != nil {
		return fmt.Errorf("failed to update Worker Cron Trigger: %s", err)
	}

	d.SetId(stringChecksum(scriptName))

	return nil
}

func resourceCloudflareWorkerCronTriggerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	scriptName := d.Get("script_name").(string)
	accountID := d.Get("account_id").(string)

	s, err := client.ListWorkerCronTriggers(context.Background(), accountID, scriptName)
	if err != nil {
		// If the script is removed, we also need to remove the triggers.
		if strings.Contains(err.Error(), "workers.api.error.script_not_found") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("failed to read Worker Cron Trigger: %s", err)
	}

	if err := d.Set("schedules", transformWorkerCronTriggerStructToSet(s)); err != nil {
		return fmt.Errorf("failed to set schedules attribute: %s", err)
	}

	return nil
}

func resourceCloudflareWorkerCronTriggerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	scriptName := d.Get("script_name").(string)
	accountID := d.Get("account_id").(string)

	client.UpdateWorkerCronTriggers(context.Background(), accountID, scriptName, []cloudflare.WorkerCronTrigger{})

	d.SetId("")

	return nil
}

func resourceCloudflareWorkerCronTriggerImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.SetId(stringChecksum(d.Id()))

	resourceCloudflareWorkerCronTriggerRead(d, meta)

	return []*schema.ResourceData{d}, nil
}

func transformWorkerCronTriggerStructToSet(triggers []cloudflare.WorkerCronTrigger) *schema.Set {
	returnSet := schema.NewSet(schema.HashString, []interface{}{})

	for _, trigger := range triggers {
		returnSet.Add(trigger.Cron)
	}

	return returnSet
}

func transformSchemaToWorkerCronTriggerStruct(d *schema.ResourceData) []cloudflare.WorkerCronTrigger {
	triggers := []cloudflare.WorkerCronTrigger{}
	schedules := d.Get("schedules").(*schema.Set).List()

	for _, schedule := range schedules {
		triggers = append(triggers, cloudflare.WorkerCronTrigger{Cron: schedule.(string)})
	}

	return triggers
}
