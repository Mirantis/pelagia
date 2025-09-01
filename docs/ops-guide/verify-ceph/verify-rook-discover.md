<a id="verify-rook-discover-mira"></a>

# Verify rook-discover

To ensure that `rook-discover` is running properly, verify if the
`local-device` configmap has been created for each Ceph node specified
in the cluster configuration:

1. Obtain the list of local devices:
   ```bash
   kubectl get configmap -n rook-ceph | grep local-device
   ```

     Example of a system response:
     ```bash
     local-device-01      1      30m
     local-device-02      1      29m
     local-device-03      1      30m
     ```

2. Verify that each device from the list contains information about
   available devices for the Ceph node deployment:
   ```bash
   kubectl describe configmap local-device-01 -n rook-ceph
   ```

     Example of a positive system response:
     ```bash
     Name:         local-device-01
     Namespace:    rook-ceph
     Labels:       app=rook-discover
                   rook.io/node=01
     Annotations:  <none>

     Data
     ====
     devices:
     ----
     [{"name":"vdd","parent":"","hasChildren":false,"devLinks":"/dev/disk/by-id/virtio-41d72dac-c0ff-4f24-b /dev/disk/by-path/virtio-pci-0000:00:09.0","size":32212254720,"uuid":"27e9cf64-85f4-48e7-8862-faa7270202ed","serial":"41d72dac-c0ff-4f24-b","type":"disk","rotational":true,"readOnly":false,"Partitions":null,"filesystem":"","vendor":"","model":"","wwn":"","wwnVendorExtension":"","empty":true,"cephVolumeData":"{\"path\":\"/dev/vdd\",\"available\":true,\"rejected_reasons\":[],\"sys_api\":{\"size\":32212254720.0,\"scheduler_mode\":\"none\",\"rotational\":\"1\",\"vendor\":\"0x1af4\",\"human_readable_size\":\"30.00 GB\",\"sectors\":0,\"sas_device_handle\":\"\",\"rev\":\"\",\"sas_address\":\"\",\"locked\":0,\"sectorsize\":\"512\",\"removable\":\"0\",\"path\":\"/dev/vdd\",\"support_discard\":\"0\",\"model\":\"\",\"ro\":\"0\",\"nr_requests\":\"128\",\"partitions\":{}},\"lvs\":[]}","label":""},{"name":"vdb","parent":"","hasChildren":false,"devLinks":"/dev/disk/by-path/virtio-pci-0000:00:07.0","size":67108864,"uuid":"988692e5-94ac-4c9a-bc48-7b057dd94fa4","serial":"","type":"disk","rotational":true,"readOnly":false,"Partitions":null,"filesystem":"","vendor":"","model":"","wwn":"","wwnVendorExtension":"","empty":true,"cephVolumeData":"{\"path\":\"/dev/vdb\",\"available\":false,\"rejected_reasons\":[\"Insufficient space (\\u003c5GB)\"],\"sys_api\":{\"size\":67108864.0,\"scheduler_mode\":\"none\",\"rotational\":\"1\",\"vendor\":\"0x1af4\",\"human_readable_size\":\"64.00 MB\",\"sectors\":0,\"sas_device_handle\":\"\",\"rev\":\"\",\"sas_address\":\"\",\"locked\":0,\"sectorsize\":\"512\",\"removable\":\"0\",\"path\":\"/dev/vdb\",\"support_discard\":\"0\",\"model\":\"\",\"ro\":\"0\",\"nr_requests\":\"128\",\"partitions\":{}},\"lvs\":[]}","label":""},{"name":"vdc","parent":"","hasChildren":false,"devLinks":"/dev/disk/by-id/virtio-e8fdba13-e24b-41f0-9 /dev/disk/by-path/virtio-pci-0000:00:08.0","size":32212254720,"uuid":"190a50e7-bc79-43a9-a6e6-81b173cd2e0c","serial":"e8fdba13-e24b-41f0-9","type":"disk","rotational":true,"readOnly":false,"Partitions":null,"filesystem":"","vendor":"","model":"","wwn":"","wwnVendorExtension":"","empty":true,"cephVolumeData":"{\"path\":\"/dev/vdc\",\"available\":true,\"rejected_reasons\":[],\"sys_api\":{\"size\":32212254720.0,\"scheduler_mode\":\"none\",\"rotational\":\"1\",\"vendor\":\"0x1af4\",\"human_readable_size\":\"30.00 GB\",\"sectors\":0,\"sas_device_handle\":\"\",\"rev\":\"\",\"sas_address\":\"\",\"locked\":0,\"sectorsize\":\"512\",\"removable\":\"0\",\"path\":\"/dev/vdc\",\"support_discard\":\"0\",\"model\":\"\",\"ro\":\"0\",\"nr_requests\":\"128\",\"partitions\":{}},\"lvs\":[]}","label":""}]
     ```
