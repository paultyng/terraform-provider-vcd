* Add support to the metadata that gets automatically created on `vcd_vapp_vm` and `vcd_vm` when they are created by a VM from a vApp Template,
  with the new `inherited_metadata` computed map. Example of metadata entries of this kind: `vm.origin.id`, `vm.origin.name`, `vm.origin.type` [GH-1146]
* Add support to the metadata that gets automatically created on `vcd_vapp` when it is created by a vApp Template or another vApp,
  with the new `inherited_metadata` computed map. Example of metadata entries of this kind: `vapp.origin.id`, `vapp.origin.name`, `vapp.origin.type` [GH-1146]
