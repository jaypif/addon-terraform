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

func TestAccVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualMachineTemplateConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "name", "test-virtual_machine"),
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "permissions", "642"),
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "memory", "1024"),
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "cpu", "0.5"),
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "group", "oneadmin"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "uid"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "gid"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "uname"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "gname"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "instance"),
					testAccCheckVirtualMachinePermissions(&shared.Permissions{
						OwnerU: 1,
						OwnerM: 1,
						GroupU: 1,
						OtherM: 1,
					}),
				),
			},
			{
				Config:             testAccVirtualMachineConfigUpdate,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "name", "test-virtual_machine-renamed"),
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "permissions", "660"),
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "group", "iamgoup"),
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "memory", "1024"),
					resource.TestCheckResourceAttr("opennebula_virtual_machine.test", "cpu", "0.5"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "uid"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "gid"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "uname"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "gname"),
					resource.TestCheckResourceAttrSet("opennebula_virtual_machine.test", "instance"),
					testAccCheckVirtualMachinePermissions(&shared.Permissions{
						OwnerU: 1,
						OwnerM: 1,
						OwnerA: 0,
						GroupU: 1,
						GroupM: 1,
					}),
				),
			},
		},
	})
}


func testAccCheckVirtualMachinePermissions(expected *shared.Permissions) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		controller := testAccProvider.Meta().(*goca.Controller)

		for _, rs := range s.RootModule().Resources {
			vmID, _ := strconv.ParseUint(rs.Primary.ID, 10, 64)
			vmc := controller.VM(int(vmID))
			// Get Virtual Machine Info
			vm, _ := vmc.Info()
			if vm == nil {
				return fmt.Errorf("Expected virtual_machine %s to exist when checking permissions", rs.Primary.ID)
			}

			if !reflect.DeepEqual(vm.Permissions, expected) {
				return fmt.Errorf(
					"Permissions for virtual_machine %s were expected to be %s. Instead, they were %s",
					rs.Primary.ID,
					permissionsUnixString(expected),
					permissionsUnixString(vm.Permissions),
				)
			}
		}

		return nil
	}
}

var testAccVirtualMachineTemplateConfigBasic = `
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

resource "opennebula_image" "image" {
   name = "image-datablock"
   description = "Terraform datablock"
   datastore_id = 1
   persistent = false
   type = "DATABLOCK"
   size = "128"
   dev_prefix = "vd"
   permissions = "660"
   driver = "qcow2"
   group = "oneadmin"
}

resource "opennebula_virtual_network" "network" {
  name = "test-virtual_network"
  type            = "bridge"
  mtu             = 1500
  ar {
    ar_type = "IP4"
    size    = 16
    ip4     = "172.16.100.110"
  }
  ar {
    ar_type = "IP4"
    size    = 12
    ip4     = "172.16.100.130"
  }
  permissions = "642"
  group = "oneadmin"
  security_groups = [0]
  clusters = [0]
}

resource "opennebula_virtual_machine" "test" {
  name        = "test-virtual_machine"
  group       = "oneadmin"
  permissions = "642"
  memory = 1024
  cpu = 0.5

  context = {
    NETWORK  = "YES"
    SET_HOSTNAME = "$NAME"
  }

  graphics {
    type   = "VNC"
    listen = "0.0.0.0"
    keymap = "en-us"
  }

  os {
    arch = "x86_64"
    boot = ""
  }

  disk {
    image_id = opennebula_image.image.id
    target = "vda"
  }

  nic {
    network_id = opennebula_virtual_network.network.id
    ip = "172.16.100.110"
  }
}
`

var testAccVirtualMachineConfigUpdate = `
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

resource "opennebula_image" "image" {
   name = "image-datablock"
   description = "Terraform datablock"
   datastore_id = 1
   persistent = false
   type = "DATABLOCK"
   size = "128"
   dev_prefix = "vd"
   permissions = "660"
   driver = "qcow2"
   group = "oneadmin"
}

resource "opennebula_virtual_network" "network" {
  name = "test-virtual_network"
  type            = "bridge"
  mtu             = 1500
  ar {
    ar_type = "IP4"
    size    = 16
    ip4     = "172.16.100.110"
  }
  ar {
    ar_type = "IP4"
    size    = 12
    ip4     = "172.16.100.130"
  }
  permissions = "642"
  group = "oneadmin"
  security_groups = [0]
  clusters = [0]
}

resource "opennebula_virtual_machine" "test" {
  name        = "test-virtual_machine-renamed"
  group       = "iamgroup"
  permissions = "660"
  memory = 1024
  cpu = 0.5

  context = {
    NETWORK  = "YES"
    SET_HOSTNAME = "$NAME"
  }

  graphics {
    type   = "VNC"
    listen = "0.0.0.0"
    keymap = "en-us"
  }

  os {
    arch = "x86_64"
    boot = ""
  }

  disk {
    image_id = opennebula_image.image.id
    target = "vda"
  }

  nic {
    network_id = opennebula_virtual_network.network.id
    ip = "172.16.100.110"
  }
}
`
