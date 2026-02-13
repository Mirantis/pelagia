<a id="increase-storage-size"></a>

# Increase Ceph cluster storage size

This section describes how to increase the overall storage size for all Ceph
pools of the same device class: `hdd`, `ssd`, or `nvme`.
The procedure presupposes adding a new Ceph OSD. The overall storage size for
the required device class automatically increases once the Ceph OSD becomes
available in the Ceph cluster.

**To increase the overall storage size for a device class:**

1. Identify the current storage size for the required device class:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph df
   ```

     Example of system response:
     ```bash
     --- RAW STORAGE ---
     CLASS  SIZE     AVAIL    USED    RAW USED  %RAW USED
     hdd    128 GiB  101 GiB  23 GiB    27 GiB      21.40
     TOTAL  128 GiB  101 GiB  23 GiB    27 GiB      21.40

     --- POOLS ---
     POOL                   ID  PGS  STORED  OBJECTS  USED    %USED  MAX AVAIL
     device_health_metrics   1    1     0 B        0     0 B      0     30 GiB
     kubernetes-hdd          2   32  12 GiB    3.13k  23 GiB  20.57     45 GiB
     ```

2. Identify the number of Ceph OSDs with the required device class:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph osd df <deviceClass>
   ```

     Substitute `<deviceClass>` with the required device class: `hdd`, `ssd`, or `nvme`.

     Example of system response for the `hdd` device class:
     ```bash
     ID  CLASS  WEIGHT   REWEIGHT  SIZE     RAW USE  DATA     OMAP      META      AVAIL    %USE   VAR   PGS  STATUS
      1    hdd  0.03119   1.00000   32 GiB  5.8 GiB  4.8 GiB   1.5 MiB  1023 MiB   26 GiB  18.22  0.85   14      up
      3    hdd  0.03119   1.00000   32 GiB  6.9 GiB  5.9 GiB   1.1 MiB  1023 MiB   25 GiB  21.64  1.01   17      up
      0    hdd  0.03119   0.84999   32 GiB  6.8 GiB  5.8 GiB  1013 KiB  1023 MiB   25 GiB  21.24  0.99   16      up
      2    hdd  0.03119   1.00000   32 GiB  7.9 GiB  6.9 GiB   1.2 MiB  1023 MiB   24 GiB  24.55  1.15   20      up
                            TOTAL  128 GiB   27 GiB   23 GiB   4.8 MiB   4.0 GiB  101 GiB  21.41
     MIN/MAX VAR: 0.85/1.15  STDDEV: 2.29
     ```

3. Follow [Add a Ceph OSD](./add-rm-ceph-osd.md#ceph-osd-add) to add a new device with a supported device class: `hdd`, `ssd`, or `nvme`.

4. Wait for the new Ceph OSD pod to start `Running`:
   ```bash
   kubectl -n rook-ceph get pod -l app=rook-ceph-osd
   ```

5. Verify that the new Ceph OSD has rebalanced and Ceph health is `HEALTH_OK`:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph -s
   ```

6. Verify that the new Ceph has been OSD added to the list of device class OSDs:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph osd df <deviceClass>
   ```

     Substitute `<deviceClass>` with the required device class: `hdd`, `ssd`, or `nvme`

     Example of system response for the `hdd` device class after adding a new Ceph OSD:
     ```bash
     ID  CLASS  WEIGHT   REWEIGHT  SIZE     RAW USE  DATA     OMAP      META      AVAIL    %USE   VAR   PGS  STATUS
      1    hdd  0.03119   1.00000   32 GiB  4.5 GiB  3.5 GiB   1.5 MiB  1023 MiB   28 GiB  13.93  0.78   10      up
      3    hdd  0.03119   1.00000   32 GiB  5.5 GiB  4.5 GiB   1.1 MiB  1023 MiB   26 GiB  17.22  0.96   13      up
      0    hdd  0.03119   0.84999   32 GiB  6.5 GiB  5.5 GiB  1013 KiB  1023 MiB   25 GiB  20.32  1.14   15      up
      2    hdd  0.03119   1.00000   32 GiB  7.5 GiB  6.5 GiB   1.2 MiB  1023 MiB   24 GiB  23.43  1.31   19      up
      4    hdd  0.03119   1.00000   32 GiB  4.6 GiB  3.6 GiB       0 B     1 GiB   27 GiB  14.45  0.81   10      up
                            TOTAL  160 GiB   29 GiB   24 GiB   4.8 MiB   5.0 GiB  131 GiB  17.87
     MIN/MAX VAR: 0.78/1.31  STDDEV: 3.62
     ```

7. Verify the total storage capacity increased for the entire device class:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph df
   ```

     Example of system response:
     ```bash
     --- RAW STORAGE ---
     CLASS  SIZE     AVAIL    USED    RAW USED  %RAW USED
     hdd    160 GiB  131 GiB  24 GiB    29 GiB      17.97
     TOTAL  160 GiB  131 GiB  24 GiB    29 GiB      17.97

     --- POOLS ---
     POOL                   ID  PGS  STORED  OBJECTS  USED    %USED  MAX AVAIL
     device_health_metrics   1    1     0 B        0     0 B      0     38 GiB
     kubernetes-hdd          2   32  12 GiB    3.18k  24 GiB  17.17     57 GiB
     ```
