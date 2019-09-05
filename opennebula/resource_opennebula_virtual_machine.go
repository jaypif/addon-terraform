package opennebula

import (
	"bytes"
	"fmt"
	"github.com/fatih/structs"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/OpenNebula/one/src/oca/go/src/goca"
	"github.com/OpenNebula/one/src/oca/go/src/goca/schemas/vm"
)

func resourceOpennebulaVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create:        resourceOpennebulaVirtualMachineCreate,
		Read:          resourceOpennebulaVirtualMachineRead,
		Exists:        resourceOpennebulaVirtualMachineExists,
		Update:        resourceOpennebulaVirtualMachineUpdate,
		Delete:        resourceOpennebulaVirtualMachineDelete,
		CustomizeDiff: resourceVMCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the VM. If empty, defaults to 'templatename-<vmid>'",
			},
			"instance": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Final name of the VM instance",
			},
			"template_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Id of the VM template to use",
			},
			"pending": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Pending state of the VM during its creation, by default it is set to false",
			},
			"permissions": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Permissions for the template (in Unix format, owner-group-other, use-manage-admin)",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)

					if len(value) != 3 {
						errors = append(errors, fmt.Errorf("%q has specify 3 permission sets: owner-group-other", k))
					}

					all := true
					for _, c := range strings.Split(value, "") {
						if c < "0" || c > "7" {
							all = false
						}
					}
					if !all {
						errors = append(errors, fmt.Errorf("Each character in %q should specify a Unix-like permission set with a number from 0 to 7", k))
					}

					return
				},
			},

			"uid": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "ID of the user that will own the VM",
			},
			"gid": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "ID of the group that will own the VM",
			},
			"uname": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the user that will own the VM",
			},
			"gname": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the group that will own the VM",
			},
			"state": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Current state of the VM",
			},
			"lcmstate": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Current LCM state of the VM",
			},
			"cpu": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: "Amount of CPU quota assigned to the virtual machine",
			},
			"vcpu": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of virtual CPUs assigned to the virtual machine",
			},
			"memory": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Amount of memory (RAM) in MB assigned to the virtual machine",
			},
			"context": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Context variables",
			},
			"disk": {
				Type:     schema.TypeSet,
				Optional: true,
				//Computed:    true,
				MinItems:    1,
				MaxItems:    8,
				Description: "Definition of disks assigned to the Virtual Machine",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"image_id": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"target": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"driver": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"graphics": {
				Type:     schema.TypeSet,
				Optional: true,
				//Computed:    true,
				MinItems:    1,
				MaxItems:    1,
				Description: "Definition of graphics adapter assigned to the Virtual Machine",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"listen": {
							Type:     schema.TypeString,
							Required: true,
						},
						"port": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"keymap": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "en-us",
						},
					},
				},
			},
			"nic": {
				Type:     schema.TypeSet,
				Optional: true,
				//Computed:    true,
				MinItems:    1,
				MaxItems:    8,
				Description: "Definition of network adapter(s) assigned to the Virtual Machine",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"mac": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"model": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"network_id": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"network": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"physical_device": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"security_groups": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeInt,
							},
						},
						"nic_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
				Set: resourceVMNicHash,
			},
			"os": {
				Type:     schema.TypeSet,
				Optional: true,
				//Computed:    true,
				MinItems:    1,
				MaxItems:    1,
				Description: "Definition of OS boot and type for the Virtual Machine",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"arch": {
							Type:     schema.TypeString,
							Required: true,
						},
						"boot": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Primary IP address assigned by OpenNebula",
			},
			"group": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"gid"},
				Description:   "Name of the Group that onws the VM, If empty, it uses caller group",
			},
		},
	}
}

func getVirtualMachineController(d *schema.ResourceData, meta interface{}, args ...int) (*goca.VMController, error) {
	controller := meta.(*goca.Controller)
	var vmc *goca.VMController

	// Try to find the VM by ID, if specified
	if d.Id() != "" {
		id, err := strconv.ParseUint(d.Id(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("VM Id (%s) is not an integer", d.Id())
		}
		vmc = controller.VM(int(id))
	}

	// Otherwise, try to find the VM by name as the de facto compound primary key
	if d.Id() == "" {
		gid, err := controller.VMs().ByName(d.Get("name").(string), args...)
		if err != nil {
			d.SetId("")
			return nil, fmt.Errorf("Could not find VM with name %s", d.Get("name").(string))
		}
		vmc = controller.VM(gid)
	}

	return vmc, nil
}

func changeVmGroup(d *schema.ResourceData, meta interface{}) error {
	controller := meta.(*goca.Controller)
	var gid int

	vmc, err := getVirtualMachineController(d, meta)
	if err != nil {
		return err
	}

	if d.Get("group") != "" {
		gid, err = controller.Groups().ByName(d.Get("group").(string))
		if err != nil {
			return err
		}
	} else {
		gid = d.Get("gid").(int)
	}

	err = vmc.Chown(-1, gid)
	if err != nil {
		return err
	}

	return nil
}

func resourceOpennebulaVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	controller := meta.(*goca.Controller)

	//Call one.template.instantiate only if template_id is defined
	//otherwise use one.vm.allocate
	var err error
	var vmID int

	if v, ok := d.GetOk("template_id"); ok {
		// if template id is set, instantiate a VM from this template
		tc := controller.Template(v.(int))

		// customize template except for memory and cpu.
		vmxml, xmlerr := generateVmXML(d)
		if xmlerr != nil {
			return xmlerr
		}

		// Instantiate template without creating a persistent copy of the template
		// Note that the new VM is not pending
		vmID, err = tc.Instantiate(d.Get("name").(string), d.Get("pending").(bool), vmxml, false)
	} else {
		if _, ok := d.GetOk("cpu"); !ok {
			return fmt.Errorf("cpu is mandatory as template_id is not used")
		}
		if _, ok := d.GetOk("memory"); !ok {
			return fmt.Errorf("memory is mandatory as template_id is not used")
		}

		vmxml, xmlerr := generateVmXML(d)
		if xmlerr != nil {
			return xmlerr
		}

		// Create VM not in pending state
		vmID, err = controller.VMs().Create(vmxml, d.Get("pending").(bool))
	}

	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%v", vmID))
	vmc := controller.VM(vmID)

	_, err = waitForVmState(d, meta, "running")
	if err != nil {
		return fmt.Errorf(
			"Error waiting for virtual machine (%s) to be in state RUNNING: %s", d.Id(), err)
	}

	// Rename the VM with its real name
	if d.Get("name") != nil {
		err := vmc.Rename(d.Get("name").(string))
		if err != nil {
			return err
		}
	}

	//Set the permissions on the VM if it was defined, otherwise use the UMASK in OpenNebula
	if perms, ok := d.GetOk("permissions"); ok {
		err = vmc.Chmod(permissionUnix(perms.(string)))
		if err != nil {
			log.Printf("[ERROR] template permissions change failed, error: %s", err)
			return err
		}
	}

	if d.Get("group") != "" || d.Get("gid") != "" {
		err = changeVmGroup(d, meta)
		if err != nil {
			return err
		}
	}

	return resourceOpennebulaVirtualMachineRead(d, meta)
}

func resourceOpennebulaVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	vmc, err := getVirtualMachineController(d, meta, -2, -1, -1)
	if err != nil {
		return err
	}

	vm, err := vmc.Info()
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%v", vm.ID))
	d.Set("instance", vm.Name)
	d.Set("name", vm.Name)
	d.Set("uid", vm.UID)
	d.Set("gid", vm.GID)
	d.Set("uname", vm.UName)
	d.Set("gname", vm.GName)
	d.Set("state", vm.StateRaw)
	d.Set("lcmstate", vm.LCMStateRaw)
	//TODO fix this:
	//d.Set("ip", vm.VmTemplate.Context.IP)
	d.Set("permissions", permissionsUnixString(vm.Permissions))

	//Pull in NIC config from OpenNebula into schema
	if vm.Template.NICs != nil {
		d.Set("nic", generateNicMapFromStructs(vm.Template.NICs))
		d.Set("ip", &vm.Template.NICs[0].IP)
	}

	if vm.Template.Disks != nil {
		d.Set("disk", generateDiskMapFromStructs(vm.Template.Disks))
	}

	if vm.Template.OS != nil {
		d.Set("os", generateOskMapFromStructs(*vm.Template.OS))
	}

	if vm.Template.Graphics != nil {
		d.Set("graphics", generateGraphicskMapFromStructs(*vm.Template.Graphics))
	}
	return nil
}

func generateGraphicskMapFromStructs(graph vm.Graphics) []map[string]interface{} {

	graphmap := make([]map[string]interface{}, 0)

	graphmap = append(graphmap, structs.Map(graph))

	return graphmap
}

func generateOskMapFromStructs(os vm.OS) []map[string]interface{} {

	osmap := make([]map[string]interface{}, 0)

	osmap = append(osmap, structs.Map(os))

	return osmap
}

func generateDiskMapFromStructs(slice []vm.Disk) []map[string]interface{} {

	diskmap := make([]map[string]interface{}, 0)

	for i := 0; i < len(slice); i++ {
		diskmap = append(diskmap, structs.Map(slice[i]))
	}

	return diskmap
}

func generateNicMapFromStructs(slice []vm.Nic) []map[string]interface{} {

	nicmap := make([]map[string]interface{}, 0)

	for i := 0; i < len(slice); i++ {
		nicmap = append(nicmap, structs.Map(slice[i]))
	}

	return nicmap
}

func resourceOpennebulaVirtualMachineExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	err := resourceOpennebulaVirtualMachineRead(d, meta)
	// a terminated VM is in state 6 (DONE)
	if err != nil || d.Id() == "" || d.Get("state").(int) == 6 {
		return false, err
	}

	return true, nil
}

func resourceOpennebulaVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {

	// Enable partial state mode
	d.Partial(true)

	//Get VM
	vmc, err := getVirtualMachineController(d, meta)
	if err != nil {
		return err
	}

	vm, err := vmc.Info()
	if err != nil {
		return err
	}

	if d.HasChange("name") {
		err := vmc.Rename(d.Get("name").(string))
		if err != nil {
			return err
		}
		vm, err := vmc.Info()
		d.SetPartial("name")
		log.Printf("[INFO] Successfully updated name (%s) for VM ID %x\n", vm.Name, vm.ID)
	}

	if d.HasChange("permissions") && d.Get("permissions") != "" {
		if perms, ok := d.GetOk("permissions"); ok {
			err = vmc.Chmod(permissionUnix(perms.(string)))
			if err != nil {
				return err
			}
		}
		d.SetPartial("permissions")
		log.Printf("[INFO] Successfully updated Permissions VM %s\n", vm.Name)
	}

	if d.HasChange("group") {
		err := changeVmGroup(d, meta)
		if err != nil {
			return err
		}
		log.Printf("[INFO] Successfully updated group for VM %s\n", vm.Name)
	}

	// We succeeded, disable partial mode. This causes Terraform to save
	// save all fields again.
	d.Partial(false)

	return nil
}

func resourceOpennebulaVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	b, err := resourceOpennebulaVirtualMachineExists(d, meta)
	if err != nil {
		return err
	}
	if !b {
		// VM already deleted
		return nil
	}

	//Get VM
	vmc, err := getVirtualMachineController(d, meta)
	if err != nil {
		return err
	}

	if err = vmc.TerminateHard(); err != nil {
		return err
	}

	_, err = waitForVmState(d, meta, "done")
	if err != nil {
		return fmt.Errorf(
			"Error waiting for virtual machine (%s) to be in state DONE: %s", d.Id(), err)
	}

	log.Printf("[INFO] Successfully terminated VM\n")
	return nil
}

func waitForVmState(d *schema.ResourceData, meta interface{}, state string) (interface{}, error) {
	var vm *vm.VM
	var err error
	//Get VM controller
	vmc, err := getVirtualMachineController(d, meta)
	if err != nil {
		return vm, err
	}

	log.Printf("Waiting for VM (%s) to be in state Done", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending: []string{"anythingelse"}, Target: []string{state},
		Refresh: func() (interface{}, string, error) {
			log.Println("Refreshing VM state...")
			//Get VM controller
			vmc, err = getVirtualMachineController(d, meta)
			if err != nil {
				return vm, "", fmt.Errorf("Could not find VM by ID %s", d.Id())
			}
			vm, err = vmc.Info()
			if err != nil {
				if strings.Contains(err.Error(), "Error getting") {
					return vm, "notfound", nil
				}
				return vm, "error getting info", err
			}
			vmState, vmLcmState, err := vm.State()
			if err != nil {
				if strings.Contains(err.Error(), "Error getting") {
					return vm, "notfound", nil
				}
				return vm, "", err
			}
			log.Printf("VM %v is currently in state %v and in LCM state %v", vm.ID, vmState, vmLcmState)
			if vmState == 3 && vmLcmState == 3 {
				return vm, "running", nil
			} else if vmState == 6 {
				return vm, "done", nil
			} else if vmState == 3 && vmLcmState == 36 {
				return vm, "boot_failure", fmt.Errorf("VM ID %s entered fail state, error message: %s", d.Id(), vm.UserTemplate.Error)
			} else {
				return vm, "anythingelse", nil
			}
		},
		Timeout:    3 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	return stateConf.WaitForState()
}

func generateVmXML(d *schema.ResourceData) (string, error) {

	//Generate CONTEXT definition
	//context := d.Get("context").(*schema.Set).List()
	context := d.Get("context").(map[string]interface{})
	log.Printf("Number of CONTEXT vars: %d", len(context))
	log.Printf("CONTEXT Map: %s", context)

	vmcontext := "CONTEXT = [\n"
	conlen := len(context)
	for key, value := range context {
		//contextvar = v.(map[string]interface{})
		if conlen == 1 {
			vmcontext = fmt.Sprintf("%s%s = %s\n", vmcontext, key, value)
		} else {
			vmcontext = fmt.Sprintf("%s%s = %s,\n", vmcontext, key, value)
		}
		conlen--
	}
	vmcontext = fmt.Sprintf("%s]", vmcontext)

	//Generate NIC definition
	nics := d.Get("nic").(*schema.Set).List()
	log.Printf("Number of NICs: %d", len(nics))
	var vmnics string
	for i := 0; i < len(nics); i++ {
		nicconfig := nics[i].(map[string]interface{})
		nicip := nicconfig["ip"].(string)
		nicmac := nicconfig["mac"].(string)
		nicmodel := nicconfig["model"].(string)
		nicsecgroups := ArrayToString(nicconfig["security_groups"].([]interface{}), ",")
		nicphydev := nicconfig["physical_device"].(string)
		nicnetworkid := nicconfig["network_id"].(int)
		vmnics = fmt.Sprintf("%s\nNIC = [\n"+
			"NETWORK_ID = %d",
			vmnics,
			nicnetworkid)
		if nicphydev != "" {
			vmnics = fmt.Sprintf("%s,PHYDEV = %s", vmnics, nicphydev)
		}
		if nicip != "" {
			vmnics = fmt.Sprintf("%s,\nIP = %s", vmnics, nicip)
		}
		if nicmac != "" {
			vmnics = fmt.Sprintf("%s,\nMAC = %s", vmnics, nicmac)
		}
		if nicmodel != "" {
			vmnics = fmt.Sprintf("%s,\nMODEL = %s", vmnics, nicmodel)
		}
		if nicsecgroups != "" {
			vmnics = fmt.Sprintf("%s,\nSECURITY_GROUPS = %s", vmnics, nicsecgroups)
		}
		vmnics = fmt.Sprintf("%s]", vmnics)
	}

	//Generate DISK definition
	disks := d.Get("disk").(*schema.Set).List()
	log.Printf("Number of disks: %d", len(disks))
	vmdisks := ""
	for i := 0; i < len(disks); i++ {
		diskconfig := disks[i].(map[string]interface{})
		diskimage := diskconfig["image_id"].(int)
		disksize := diskconfig["size"].(int)
		disktarget := diskconfig["target"].(string)
		diskdriver := diskconfig["driver"].(string)
		vmdisks = fmt.Sprintf("%s\nDISK = [\n", vmdisks)
		if disksize > 0 {
			vmdisks = fmt.Sprintf("%sSIZE = %d,\n", vmdisks, disksize)
		}
		if disktarget != "" {
			vmdisks = fmt.Sprintf("%sTARGET = %s,\n", vmdisks, disktarget)
		}
		if diskdriver != "" {
			vmdisks = fmt.Sprintf("%sDRIVER = %s,\n", vmdisks, diskdriver)
		}
		vmdisks = fmt.Sprintf("%sIMAGE_ID = %d]", vmdisks, diskimage)
	}

	//Generate GRAPHICS definition
	var vmgraphics string
	if g, ok := d.GetOk("graphics"); ok {
		graphics := g.(*schema.Set).List()
		graphicsconfig := graphics[0].(map[string]interface{})
		conflen := len(graphicsconfig)
		listen := graphicsconfig["listen"].(string)
		gtype := graphicsconfig["type"].(string)
		port := graphicsconfig["port"].(string)
		keymap := graphicsconfig["keymap"].(string)

		vmgraphics = "GRAPHICS = ["
		if listen != "" {
			if conflen == 1 {
				vmgraphics = fmt.Sprintf("%s\nLISTEN = %s\n", vmgraphics, listen)
			} else {
				vmgraphics = fmt.Sprintf("%s\nLISTEN = %s,\n", vmgraphics, listen)
			}
			conflen--
		}
		if gtype != "" {
			if conflen == 1 {
				vmgraphics = fmt.Sprintf("%sTYPE = %s\n", vmgraphics, gtype)
			} else {
				vmgraphics = fmt.Sprintf("%sTYPE = %s,\n", vmgraphics, gtype)
			}
			conflen--
		}
		if port != "" {
			if conflen == 1 {
				vmgraphics = fmt.Sprintf("%sPORT = %s\n", vmgraphics, port)
			} else {
				vmgraphics = fmt.Sprintf("%sPORT = %s,\n", vmgraphics, port)
			}
			conflen--
		}
		if keymap != "" {
			vmgraphics = fmt.Sprintf("%sKEYMAP = %s\n", vmgraphics, keymap)
		}
		vmgraphics = fmt.Sprintf("%s]", vmgraphics)
	}

	//Generate OS definition
	var vmos string
	if o, ok := d.GetOk("os"); ok {
		os := o.(*schema.Set).List()
		osconfig := os[0].(map[string]interface{})
		arch := osconfig["arch"].(string)
		boot := osconfig["boot"].(string)
		vmos = "OS = [\n"
		if arch != "" {
			vmos = fmt.Sprintf("%sARCH = %s,\n", vmos, arch)
		}
		vmos = fmt.Sprintf("%sBOOT = \"%s\"]", vmos, boot)
	}

	//Pull all the bits together into the main VM template
	var vmvcpu interface{}
	var vmcpu interface{}
	var vmmemory interface{}
	vmtpl := ""
	var ok bool
	if vmcpu, ok = d.GetOk("cpu"); ok {
		if vmmemory, ok = d.GetOk("memory"); ok {
			if vmvcpu, ok = d.GetOk("vcpu"); ok {
				vmtpl = fmt.Sprintf("VCPU = %d\n"+
					"CPU = %f\n"+
					"MEMORY = %d\n"+
					"%s\n"+
					"%s\n"+
					"%s\n"+
					"%s\n"+
					"%s",
					vmvcpu.(int),
					vmcpu.(float64),
					vmmemory.(int),
					vmcontext,
					vmos,
					vmgraphics,
					vmnics,
					vmdisks)
			} else {
				vmtpl = fmt.Sprintf("CPU = %f\n"+
					"MEMORY = %d\n"+
					"%s\n"+
					"%s\n"+
					"%s\n"+
					"%s\n"+
					"%s",
					vmcpu.(float64),
					vmmemory.(int),
					vmcontext,
					vmos,
					vmgraphics,
					vmnics,
					vmdisks)
			}
		} else {
			vmtpl = fmt.Sprintf("CPU = %f\n"+
				"%s\n"+
				"%s\n"+
				"%s\n"+
				"%s\n"+
				"%s",
				vmcpu.(float64),
				vmcontext,
				vmos,
				vmgraphics,
				vmnics,
				vmdisks)
		}
	} else {
		vmtpl = fmt.Sprintf("%s\n"+
			"%s\n"+
			"%s\n"+
			"%s\n"+
			"%s",
			vmnics,
			vmos,
			vmgraphics,
			vmcontext,
			vmdisks)
	}

	log.Printf("VM XML: %s", vmtpl)
	return vmtpl, nil

}

func resourceVMNicHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["model"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["network_id"].(int)))
	return hashcode.String(buf.String())
}

func resourceVMCustomizeDiff(diff *schema.ResourceDiff, v interface{}) error {
	// If the VM is in error state, force the VM to be recreated
	if diff.Get("lcmstate") == 36 {
		log.Printf("[INFO] VM is in error state, forcing recreate.")
		diff.SetNew("lcmstate", 3)
		if err := diff.ForceNew("lcmstate"); err != nil {
			return err
		}
	}

	return nil
}
