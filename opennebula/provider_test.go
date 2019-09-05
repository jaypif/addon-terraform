package opennebula

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"strconv"
	"testing"

	"github.com/OpenNebula/one/src/oca/go/src/goca"
)

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]terraform.ResourceProvider{
		"opennebula": testAccProvider,
	}
}

func testAccPreCheck(t *testing.T) {
	testEnvIsSet("OPENNEBULA_ENDPOINT", t)
	testEnvIsSet("OPENNEBULA_USERNAME", t)
	testEnvIsSet("OPENNEBULA_PASSWORD", t)
}

func testEnvIsSet(k string, t *testing.T) {
	if v := os.Getenv(k); v == "" {
		t.Fatalf("%s must be set for acceptance tests", k)
	}
}

func testAccCheckDestroy(s *terraform.State) error {
	controller := testAccProvider.Meta().(*goca.Controller)

	for _, rs := range s.RootModule().Resources {
		ID, _ := strconv.ParseUint(rs.Primary.ID, 10, 64)
		if rs.Type == "opennebula_image" {
			ic := controller.Image(int(ID))
			// Get Image Info
			image, _ := ic.Info()
			if image != nil {
				return fmt.Errorf("Expected image %s to have been destroyed", rs.Primary.ID)
			}
		}
		if rs.Type == "opennebula_group" {
			gc := controller.Group(int(ID))
			// Get Group Info
			group, _ := gc.Info()
			if group != nil {
				return fmt.Errorf("Expected group %s to have been destroyed", rs.Primary.ID)
			}
		}
		if rs.Type == "opennebula_security_group" {
			sgc := controller.SecurityGroup(int(ID))
			// Get Security Group Info
			sg, _ := sgc.Info()
			if sg != nil {
				return fmt.Errorf("Expected security group %s to have been destroyed", rs.Primary.ID)
			}
		}
		if rs.Type == "opennebula_virtual_network" {
			vnc := controller.VirtualNetwork(int(ID))
			// Get Virtual Network Info
			vn, _ := vnc.Info()
			if vn != nil {
				return fmt.Errorf("Expected virtual network %s to have been destroyed", rs.Primary.ID)
			}
		}
		if rs.Type == "opennebula_virtual_machine" {
			vmc := controller.VM(int(ID))
			// Get Virtual Machine Info
			vm, err := vmc.Info()
			if err != nil {
				return err
			}
			// When a VM is destroyed, it is in "DONE" state
			vmState, _, err := vm.StateString()
			if err != nil {
				return err
			}
			if vmState != "DONE" {
				return fmt.Errorf("Expected virtual machine %s to have been destroyed, current state is: %s", rs.Primary.ID, vmState)
			}
		}
	}

	return nil
}
