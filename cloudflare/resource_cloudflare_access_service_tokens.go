package cloudflare

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudflareAccessServiceToken() *schema.Resource {
	return &schema.Resource{
		Schema: resourceCloudflareAccessServiceTokenSchema(),
		Create: resourceCloudflareAccessServiceTokenCreate,
		Read:   resourceCloudflareAccessServiceTokenRead,
		Update: resourceCloudflareAccessServiceTokenUpdate,
		Delete: resourceCloudflareAccessServiceTokenDelete,
		Importer: &schema.ResourceImporter{
			State: resourceCloudflareAccessServiceTokenImport,
		},

		CustomizeDiff: customdiff.ComputedIf("expires_at", resourceCloudflareAccessServiceTokenExpireDiff),
	}
}

func resourceCloudflareAccessServiceTokenExpireDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) bool {
	mindays := d.Get("min_days_for_renewal").(int)
	if mindays > 0 {
		expires_at := d.Get("expires_at").(string)

		if expires_at != "" {
			expected_expiration_date, _ := time.Parse(time.RFC3339, expires_at)

			expiration_date := time.Now().Add(time.Duration(mindays) * 24 * time.Hour)

			if expiration_date.After(expected_expiration_date) {
				d.SetNewComputed("client_secret")
				return true
			}
		}
	}

	return false
}

func resourceCloudflareAccessServiceTokenRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	identifier, err := initIdentifier(d)
	if err != nil {
		return err
	}

	// The Cloudflare API doesn't support fetching a single service token
	// so instead we loop over all the service tokens and only continue
	// when we have a match.
	var serviceTokens []cloudflare.AccessServiceToken
	if identifier.Type == AccountType {
		serviceTokens, _, err = client.AccessServiceTokens(context.Background(), identifier.Value)
	} else {
		serviceTokens, _, err = client.ZoneLevelAccessServiceTokens(context.Background(), identifier.Value)
	}
	if err != nil {
		return fmt.Errorf("error fetching access service tokens: %s", err)
	}
	for _, token := range serviceTokens {
		if token.ID == d.Id() {
			d.Set("name", token.Name)
			d.Set("client_id", token.ClientID)
			d.Set("expires_at", token.ExpiresAt.Format(time.RFC3339))
		}
	}

	return nil
}

func resourceCloudflareAccessServiceTokenCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	tokenName := d.Get("name").(string)

	identifier, err := initIdentifier(d)
	if err != nil {
		return err
	}

	var serviceToken cloudflare.AccessServiceTokenCreateResponse
	if identifier.Type == AccountType {
		serviceToken, err = client.CreateAccessServiceToken(context.Background(), identifier.Value, tokenName)
	} else {
		serviceToken, err = client.CreateZoneLevelAccessServiceToken(context.Background(), identifier.Value, tokenName)
	}
	if err != nil {
		return fmt.Errorf("error creating access service token: %s", err)
	}

	d.SetId(serviceToken.ID)
	d.Set("name", serviceToken.Name)
	d.Set("client_id", serviceToken.ClientID)
	d.Set("client_secret", serviceToken.ClientSecret)
	d.Set("expires_at", serviceToken.ExpiresAt.Format(time.RFC3339))

	resourceCloudflareAccessServiceTokenRead(d, meta)

	return nil
}

func resourceCloudflareAccessServiceTokenUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	tokenName := d.Get("name").(string)

	identifier, err := initIdentifier(d)
	if err != nil {
		return err
	}

	var serviceToken cloudflare.AccessServiceTokenUpdateResponse
	if identifier.Type == AccountType {
		serviceToken, err = client.UpdateAccessServiceToken(context.Background(), identifier.Value, d.Id(), tokenName)
	} else {
		serviceToken, err = client.UpdateZoneLevelAccessServiceToken(context.Background(), identifier.Value, d.Id(), tokenName)
	}
	if err != nil {
		return fmt.Errorf("error updating access service token: %s", err)
	}

	d.Set("name", serviceToken.Name)

	return resourceCloudflareAccessServiceTokenRead(d, meta)
}

func resourceCloudflareAccessServiceTokenDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	identifier, err := initIdentifier(d)
	if err != nil {
		return err
	}

	if identifier.Type == AccountType {
		_, err = client.DeleteAccessServiceToken(context.Background(), identifier.Value, d.Id())
	} else {
		_, err = client.DeleteZoneLevelAccessServiceToken(context.Background(), identifier.Value, d.Id())
	}
	if err != nil {
		return fmt.Errorf("error deleting access service token: %s", err)
	}

	d.SetId("")

	return nil
}

func resourceCloudflareAccessServiceTokenImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	attributes := strings.SplitN(d.Id(), "/", 2)

	if len(attributes) != 2 {
		return nil, fmt.Errorf("invalid id (\"%s\") specified, should be in format \"accountID/serviceTokenID\"", d.Id())
	}

	d.Set("account_id", attributes[0])
	d.SetId(attributes[1])

	resourceCloudflareAccessServiceTokenRead(d, meta)

	return []*schema.ResourceData{d}, nil
}
