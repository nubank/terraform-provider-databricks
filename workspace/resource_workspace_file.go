// file: workspace/resource_workspace_file.go     
package workspace

import (
	"bytes"
	"context"
	"log"
	"path/filepath"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/databricks/terraform-provider-databricks/common"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func workspaceFileUploadOptionFunc(i *workspace.Import) {
	i.Format = "RAW"
	i.Overwrite = true
	i.ForceSendFields = []string{"Content"}
}


func ResourceWorkspaceFile() common.Resource {
	s := FileContentSchema(map[string]*schema.Schema{
		"url": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"object_id": {
			Type:     schema.TypeInt,
			Optional: true,
			Computed: true,
		},
		"workspace_path": {
			Type:     schema.TypeString,
			Computed: true,
		},
	})
	common.AddNamespaceInSchema(s)
	common.NamespaceCustomizeSchemaMap(s)
	return common.Resource{
		Schema:        s,
		SchemaVersion: 1,
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, c *common.DatabricksClient) error {
			return common.NamespaceCustomizeDiff(ctx, d, c)
		},
		Create: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			content, err := ReadContent(d)
			if err != nil {
				return err
			}
			client, err := c.WorkspaceClientUnifiedProvider(ctx, d)
			if err != nil {
				return err
			}
			path := d.Get("path").(string)
			
			
			err = client.Workspace.Upload(ctx, path, bytes.NewReader(content), workspaceFileUploadOptionFunc)
			if err != nil {
				parent := filepath.ToSlash(filepath.Dir(path))
				
				
				_, errStatus := client.Workspace.GetStatusByPath(ctx, parent)
				if errStatus != nil {
					log.Printf("[DEBUG] Parent folder '%s' check failed, executing concurrent safe mkdirs...", parent)
					
					
					errMkdir := client.Workspace.Mkdirs(ctx, workspace.Mkdirs{Path: parent})
					if errMkdir != nil {
						errStr := strings.ToUpper(errMkdir.Error())
						if !strings.Contains(errStr, "RESOURCE_ALREADY_EXISTS") && 
						   !strings.Contains(errStr, "ALREADY_EXISTS") {
							return errMkdir
						}
					}
					
					err = client.Workspace.Upload(ctx, path, bytes.NewReader(content), workspaceFileUploadOptionFunc)
					if err != nil {
						return err
					}
				} else {
					
					return err
				}
			}
			
			d.SetId(path)   
			return nil
		},
		Read: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			client, err := c.WorkspaceClientUnifiedProvider(ctx, d)
			if err != nil {
				return err
			}
			objectStatus, err := common.RetryOnTimeout(ctx, func(ctx context.Context) (*workspace.ObjectInfo, error) {
				return client.Workspace.GetStatusByPath(ctx, d.Id())
			})
			if err != nil {
				return err
			}
			SetWorkspaceObjectComputedProperties(d, c)
			return common.StructToData(objectStatus, s, d)
		},
		Update: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			client, err := c.WorkspaceClientUnifiedProvider(ctx, d)
			if err != nil {
				return err
			}
			content, err := ReadContent(d)
			if err != nil {
				return err
			}
			return client.Workspace.Upload(ctx, d.Id(), bytes.NewReader(content), workspaceFileUploadOptionFunc)
		},
		Delete: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			client, err := c.WorkspaceClientUnifiedProvider(ctx, d)
			if err != nil {
				return err
			}
			return client.Workspace.Delete(ctx, workspace.Delete{Path: d.Id(), Recursive: false})
		},
	}
}