# systemboot

SystemBoot is a distribution for LinuxBoot to create a system firmware + bootloader. It is based on u-root. The provided programs are:

-   `netboot`: a network boot client that uses DHCP and HTTP to get a boot program based on Linux, and uses kexec to run it
-   `localboot`: a tool that finds bootable kernel configurations on the local disks and boots them
-   `uinit`: a wrapper around `netboot` and `localboot` that just mimicks a BIOS/UEFI BDS behaviour, by looping between network booting and local booting. The name `uinit` is necessary to be picked up as boot program by u-root.

This work is similar to the `pxeboot` and `boot` commands that are already part of u-root, but approach and implementation are slightly different. Thanks to Chris Koch and Jean-Marie Verdun for pioneering in this area.

This project started as a personal experiment under github.com/insomniacslk/systemboot but it is now an effort of a broader community and graduated to a real project for system firmwares.

The next sections go into further details.

## netboot

The `netboot` client has the duty of configuring the network, downloading a boot program, and kexec'ing it.
Optionally, the network configuration can be obtained via SLAAC and the boot program URL can be overridden to use a known endpoint.

In its DHCP-mode operation, `netboot` does the following:

-   bring up the selected network interface (`eth0` by default)
-   make a DHCPv6 transaction asking for network configuration, DNS, and a boot file URL
-   extract network and DNS configuration from the DHCP reply and configure the interface
-   extract the boot file URL from the DHCP reply and download it. The only supported scheme at the moment is HTTP. No TFTP, sorry, it's 2018 (but I accept pull requests)
-   kexec the downloaded boot program

There is an additional mode that uses SLAAC and a known endpoint, that can be enabled with `-skip-dhcp`, `-netboot-url`, and a working SLAAC configuration.

## localboot

The `localboot` program looks for bootable kernels on attached storage and tries to boot them in order, until one succeeds.
In the future it will support a configurable boot order, but for that I need [Google VPD](https://chromium.googlesource.com/chromiumos/platform/vpd/) support, which will come soon.

In the current mode, `localboot` does the following:

-   look for all the locally attached block devices
-   try to mount them with all the available file systems
-   look for a GRUB configuration on each mounted partition
-   look for valid kernel configurations in each GRUB config
-   try to boot (via kexec) each valid kernel/ramfs combination found above

In the future I will also support VPD, which will be used as a substitute for EFI variables, in this specific case to hold the boot order of the various boot entries.

## verifiedboot

Is a booter tool which loads a boot configuration by given NVRAM variables and executes into the OS in a secure manner. This is done by using verified and measured boot process.

### Boot options

Boot options are stored inside the firmware NVRAM and dynamically probed by systemboot.
Those options can be written as map where the key is the string identifier and the value is the responding json matching the booter (e.g. verifiedbooter):

```go
[ "Boot0000": {"device_path": "/dev/sda1", "bc_file": "/boot/bc.img", "bc_name": "test"} ]
```

Each key is unique and RO NVRAM precedes RW entries. So they overwrite them.
Boot order is in reverse order which leads to follong boot procedure.

```bash
Boot9999
...
Boot0003 ----|
Boot0002 <---|----|
Boot0001 <--------|
Boot0000 <- RO fallback
```

If newly written RW entries break the boot procedure the RO entry will give
the a fallback option for booting the system.

#### VPD

For coreboot firmware boot options are implemented through VPD.
Now if we want to write some boot options from the OS in order to
describe the boot setup.

##### Required Tools

-   [VPD tool](https://chromium.googlesource.com/chromiumos/platform/vpd/)
-   [cbfstool](https://github.com/coreboot/coreboot/tree/master/util/cbfstool)
-   [flashrom](https://review.coreboot.org/flashrom.git)

##### Set RO factory data

Set the boot device where the boot configuration file is located

###### Add RO VPD entries to file

```bash
vpd -f build/coreboot.rom -i RO_VPD -O -s "Boot0000"='{"device_path": "/dev/sda1", "bc_file": "/boot/bc.img", "bc_name": "test"}'
```

##### Set RW options inside the OS

Inside the running OS the RW VPD values can be easily written:

```bash
vpd -i "RW_VPD" -O -s "Boot0001"='{"device_path": "/dev/sda1", "bc_file": "/boot/bc.img", "bc_name": "test"}'
```

### Integration

#### Boot Configuration

During boot phase systemboot tries to find the boot configuration file by using the NVRAM boot options. If it is found then it gets executed by systemboot as part of the boot process.

##### 1) Generate signing keys

Needed to sign the the boot configuration file

```bash
configtool genkeys --passphrase=thisisnotasecurepassword private_key.pem public_key.pem
```

##### 2) Create a manifest.json

Which can have multiple boot configurations in an array. It offers the following config items:

###### commandline

Is the kernel commandline for the linux kernel to boot.

###### initrd

Is the initramfs file name in the initrd directory.

###### kernel

Is the kernel file name in the kernel directory.

###### dt

Is the device-tree file name in the dt directory.

###### name

Is a unique identifier for the boot configuration.

```json
{
  "configs": [
    {
      "commandline": "root=/dev/mapper/sys-root ro root_trim=yes crypt_root=UUID=597ca453-ddb4-499b-8385-aa1383133249 keymap=de dolvm init=/lib/systemd/systemd net.ifnames=0 intel_iommu=igfx_off",
      "initrd": "initrd",
      "kernel": "kernel",
      "name": "test"
    }
  ]
}
```

##### 3) Pack configuration + arbitrary files

Now we merging everything into one zip file and attaching a signature at the end.

```bash
configtool pack --passphrase=thisisnotasecurepassword --kernel-dir=kernelDir --initrd-dir=initrdDir --dt-dir=deviceTreeDir manifest.json bc.file private_key.pem
```

Afterwards copy the boot file to the boot directory with the file path location stored inside the NVRAM vars.

#### Go Initramnfs

In order to setup systemboot properly it needs to be bundled into u-root.

##### 4) Download u-root

```bash
git clone https://github.com/u-root/u-root
```

##### 5) Compile u-root tool

```bash
go get ./...
```

```bash
go build u-root.go
```

##### 6) Generate initramfs

Select the right architecture via GOARCH environment variable.
Keep in mind to select the right kexec binary.

```bash
GOARCH=arm64 ./u-root -build bb -files "kexec-arm64:sbin/kexec-arm64" -files "public_key.pem:etc/security/key.pem" -format cpio -o initramfs.cpio cmds/init path/to/systemboot/verifiedboot/dir path/to/systemboot/uinit/dir
```

##### 7) Compress initramfs

In order to reduce the size we compress it with XZ.

```bash
xz -9 --check=crc32 --lzma2=dict=1MiB initramfs.cpio
```

#### Glue it together

Now we are throwing everything together into our coreboot image.

##### 8) Configure coreboot for LinuxBoot support

```bash
make menuconfig
```

##### 9) Overwrite the coreboot config var

```bash
echo 'CONFIG_PAYLOAD_USERSPACE="path/to/initramfs.cpio.xz"' > .config
```

##### 10) Build coreboot

```bash
make
```

##### 11) Add RO_VPD region to coreboot

```bash
./vpd -f build/coreboot.rom -O -i RO_VPD -s "Boot0000"='{"device_path": "/dev/sda1", "bc_file": "/boot/bc.img", "bc_name": "test"}'
```

## uinit

The `uinit` program just wraps `netboot` and `localboot` in a forever-loop logic, just like your BIOS/UEFI would do. At the moment it just loops between netboot and localboot in this order, but I plan to make this more flexible and configurable.

## Who uses systemboot?

Public projects that use it and that we are aware of:
* [OpenCellular](https://github.com/Telecominfraproject/OpenCellular/wiki/How-to-systemboot-(verifiedboot)

If you use systemboot in your project please let us know and we will add your
project to this list.

## TODO

-   DHCPv4 is under work
-   VPD
-   TPM support
-   verified and measured boot
-   a proper GRUB config parser
-   backwards compatibility with BIOS-style partitions
