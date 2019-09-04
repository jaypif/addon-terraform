package opennebula

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"reflect"
	"strconv"
	"testing"

	"github.com/OpenNebula/one/src/oca/go/src/goca"
	"github.com/OpenNebula/one/src/oca/go/src/goca/schemas/shared"
)

func TestAccImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("opennebula_image.testimage", "name", "test-image-datablock"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "datastore_id", "1"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "persistent", "true"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "type", "DATABLOCK"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "size", "128"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "dev_prefix", "vd"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "driver", "qcow2"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "permissions", "742"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "group", "oneadmin"),
					resource.TestCheckResourceAttrSet("opennebula_image.testimage", "uid"),
					resource.TestCheckResourceAttrSet("opennebula_image.testimage", "gid"),
					resource.TestCheckResourceAttrSet("opennebula_image.testimage", "uname"),
					resource.TestCheckResourceAttrSet("opennebula_image.testimage", "gname"),
					testAccCheckImagePermissions(&shared.Permissions{
						OwnerU: 1,
						OwnerM: 1,
						OwnerA: 1,
						GroupU: 1,
						OtherM: 1,
					}, "test-image-datablock"),
				),
			},
			{
				Config:  testAccImageConfigUpdate,
				Destroy: false,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("opennebula_image.testimage", "name", "test-image-datablock-renamed"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "datastore_id", "1"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "persistent", "false"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "type", "OS"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "size", "128"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "dev_prefix", "vd"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "driver", "qcow2"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "permissions", "660"),
					resource.TestCheckResourceAttr("opennebula_image.testimage", "group", "oneadmin"),
					resource.TestCheckResourceAttrSet("opennebula_image.testimage", "uid"),
					resource.TestCheckResourceAttrSet("opennebula_image.testimage", "gid"),
					resource.TestCheckResourceAttrSet("opennebula_image.testimage", "uname"),
					resource.TestCheckResourceAttrSet("opennebula_image.testimage", "gname"),
					testAccCheckImagePermissions(&shared.Permissions{
						OwnerU: 1,
						OwnerM: 1,
						OwnerA: 0,
						GroupU: 1,
						GroupM: 1,
					}, "test-image-datablock-renamed"),
				),
			},
			{
				Config:  testAccImageConfigClone,
				Destroy: false,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("opennebula_image.clone", "name", "test-image-clone"),
					resource.TestCheckResourceAttr("opennebula_image.clone", "datastore_id", "1"),
					resource.TestCheckResourceAttr("opennebula_image.clone", "dev_prefix", "vd"),
					resource.TestCheckResourceAttr("opennebula_image.clone", "permissions", "660"),
					resource.TestCheckResourceAttr("opennebula_image.clone", "group", "iamgroup"),
					resource.TestCheckResourceAttrSet("opennebula_image.clone", "uid"),
					resource.TestCheckResourceAttrSet("opennebula_image.clone", "gid"),
					resource.TestCheckResourceAttrSet("opennebula_image.clone", "uname"),
					resource.TestCheckResourceAttrSet("opennebula_image.clone", "gname"),
					testAccCheckImagePermissions(&shared.Permissions{
						OwnerU: 1,
						OwnerM: 1,
						OwnerA: 0,
						GroupU: 1,
						GroupM: 1,
					}, "test-image-clone"),
				),
			},
		},
	})
}

func testAccCheckImagePermissions(expected *shared.Permissions, resourcename string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		controller := testAccProvider.Meta().(*goca.Controller)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "opennebula_image" {
				continue
			}
			imageID, _ := strconv.ParseUint(rs.Primary.ID, 10, 64)
			ic := controller.Image(int(imageID))
			// Get image Info
			image, _ := ic.Info()
			if image == nil {
				return fmt.Errorf("Expected image %s to exist when checking permissions", rs.Primary.ID)
			}
			if image.Name != resourcename {
				continue
			}

			if !reflect.DeepEqual(image.Permissions, expected) {
				return fmt.Errorf(
					"Permissions for image %s were expected to be %s. Instead, they were %s",
					rs.Primary.ID,
					permissionsUnixString(expected),
					permissionsUnixString(image.Permissions),
				)
			}
		}

		return nil
	}
}

var testAccImageConfigBasic = `
resource "opennebula_image" "testimage" {
   name = "test-image-datablock"
   description = "Terraform datablock"
   datastore_id = 1
   persistent = true
   type = "DATABLOCK"
   size = "128"
   dev_prefix = "vd"
   permissions = "742"
   driver = "qcow2"
   group = "oneadmin"
}
`

var testAccImageConfigUpdate = `
resource "opennebula_image" "testimage" {
   name = "test-image-datablock-renamed"
   description = "Terraform datablock"
   datastore_id = 1
   persistent = false
   type = "OS"
   size = "128"
   dev_prefix = "vd"
   permissions = "660"
   driver = "qcow2"
   group = "oneadmin"
}
`
var testAccImageConfigClone = `
resource "opennebula_group" "group" {
  name = "iamgroup"
  template = <<EOF
    SUNSTONE = [
      DEFAULT_VIEW = "cloud",
      GROUP_ADMIN_DEFAULT_VIEW = "groupadmin",
      GROUP_ADMIN_VIEWS = "groupadmin",
      VIEWS = "cloud"
    ]
    EOF
    delete_on_destruction = true
    quotas {
        vm {
            cpu = 4
            memory = 8192
        }
    }
}

resource "opennebula_image" "testimage" {
   name = "test-image-datablock-renamed"
   description = "Terraform datablock"
   datastore_id = 1
   persistent = false
   type = "OS"
   size = "128"
   dev_prefix = "vd"
   permissions = "660"
   driver = "qcow2"
   group = "oneadmin"
}

resource "opennebula_image" "clone" {
   name = "test-image-clone"
   description = "Terraform clone"
   clone_from_image = opennebula_image.testimage.id
   datastore_id = 1
   persistent = false
   dev_prefix = "vd"
   permissions = "660"
   group = "iamgroup"
}
`
