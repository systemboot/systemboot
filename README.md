# systemboot

[![Build Status](https://travis-ci.org/systemboot/systemboot.svg?branch=master)](https://travis-ci.org/systemboot/systemboot)
[![codecov](https://codecov.io/gh/systemboot/systemboot/branch/master/graph/badge.svg)](https://codecov.io/gh/systemboot/systemboot)
[![Go Report Card](https://goreportcard.com/badge/github.com/systemboot/systemboot)](https://goreportcard.com/report/github.com/systemboot/systemboot)


SystemBoot is a distribution for LinuxBoot to create a system firmware + bootloader. It is based on [u-root](https://github.com/u-root/u-root). The provided programs are:

* `netboot`: a network boot client that uses DHCP and HTTP to get a boot program based on Linux, and uses kexec to run it
* `localboot`: a tool that finds bootable kernel configurations on the local disks and boots them
* `stboot`: a network boot client that uses static ip configuration stored on disk to get a signed boot program based on 
* `uinit`: a wrapper around `netboot` and `localboot` that just mimicks a BIOS/UEFI BDS behaviour, by looping between network booting and local booting. The name `uinit` is necessary to be picked up as boot program by u-root.

This work is similar to the `pxeboot` and `boot` commands that are already part of u-root, but approach and implementation are slightly different. Thanks to Chris Koch and Jean-Marie Verdun for pioneering in this area.

This project started as a personal experiment under github.com/insomniacslk/systemboot but it is now an effort of a broader community and graduated to a real project for system firmwares.

The next sections go into further details.

## netboot

The `netboot` client has the duty of configuring the network, downloading a boot program, and kexec'ing it.
Optionally, the network configuration can be obtained via SLAAC and the boot program URL can be overridden to use a known endpoint.

In its DHCP-mode operation, `netboot` does the following:
* bring up the selected network interface (`eth0` by default)
* make a DHCPv6 transaction asking for network configuration, DNS, and a boot file URL
* extract network and DNS configuration from the DHCP reply and configure the interface
* extract the boot file URL from the DHCP reply and download it. The only supported scheme at the moment is HTTP. No TFTP, sorry, it's 2018 (but I accept pull requests)
* kexec the downloaded boot program

There is an additional mode that uses SLAAC and a known endpoint, that can be enabled with `-skip-dhcp`, `-netboot-url`, and a working SLAAC configuration.

## localboot

The `localboot` program looks for bootable kernels on attached storage and tries to boot them in order, until one succeeds.
In the future it will support a configurable boot order, but for that I need [Google VPD](https://chromium.googlesource.com/chromiumos/platform/vpd/) support, which will come soon.

In the current mode, `localboot` does the following:
* look for all the locally attached block devices
* try to mount them with all the available file systems
* look for a GRUB configuration on each mounted partition
* look for valid kernel configurations in each GRUB config
* try to boot (via kexec) each valid kernel/ramfs combination found above

In the future I will also support VPD, which will be used as a substitute for EFI variables, in this specific case to hold the boot order of the various boot entries.

## stboot
The `stboot` programm looks for configuration file named `netvars.json` on all available blockdevices. In this file a static IP configuration is defined as well as a bootstap URL to get the final boot files from and the pupblic key to verify the signature. These bootfiles are packed into a signed ZIP archive. `stboot` downloads This is done by using verified and measured boot process.

### IP Configuration

#### 1) Create netvars.json
There must be an `netvars.json` on the disk with the following structure:
```json
{
  {
  "host_ip":"10.0.2.15/24",
  "netmask":"",
  "gateway":"10.0.2.2/24",
  "dns":"",
  "host_priv_key":"",
  "host_pub_key":"",
  "bootstrap_url":"http://some.remotesource.io/bc.zip",
  "signature_pub_key":""
}

}
```
###### host_ip
The static IP address of the host. The IP musst be specified in CIDR notation like above.

###### netmask
The traditional notation of subnet mask. This field is ignored at the moment.

###### gateway
The static IP address of default gateway. The IP musst be specified in CIDR notation like above.

###### dns
With this field a dedicated DNS Server can be choosen.

###### host_priv_key
Fore later use.

###### host_pub_key
Fore later use.

###### bootstrap_url
This is the URL where the archive including the boot files resides.

###### signature_pub_key
The public key to verify the signature of the archive


#### 2) Add netvars.json to disk
The `netvars.json` has to be placed at `/` on any of the blockdevices in the system. `stboot` will check any device listed unter `/sys/class/blck` and searches the file at root level of the filesystem

### Boot Configuration

During boot phase systemboot tries to find the boot configuration file by using the NVRAM boot options. If it is found then it gets executed by systemboot as part of the boot process.

#### 1) Generate signing keys

Needed to sign the the boot configuration file

```bash
configtool genkeys --passphrase=thisisnotasecurepassword private_key.pem public_key.pem
```

#### 2) Create a manifest.json
This a unique identifier for the boot configuration. It can have multiple boot configurations in an array:

```json
{
  "configs": [
    {
      "kernel_args": "root=/dev/hdx",
      "initrd": "initrd/path/inside/zip-archive",
      "kernel": "initrd/path/inside/zip-archive",
      "name": "test stboot configuration"
    }
  ]
}
```

###### kernel_args
Is the kernel commandline for the linux kernel to boot.

###### initrd
Is the initramfs file name in the initrd directory.

###### kernel
Is the kernel file name in the kernel directory.

###### dt
Is the device-tree file name in the dt directory.

###### name
The name of the boot configuration



#### 3) Pack configuration + arbitrary files

Now we merging everything into one zip file and attaching a signature at the end. The output file must have an `.zip` extension. In the following example it is `bc.zip`:

```bash
configtool pack --passphrase=thisisnotasecurepassword --kernel-dir=kernelDir --initrd-dir=initrdDir --dt-dir=deviceTreeDir manifest.json bc.zip private_key.pem
```
Afterwards upload `bc.zip` to a server so that it maches the URL you specified in `netvars.json`.

## uinit

The `uinit` program just wraps `netboot` and `localboot` in a forever-loop logic, just like your BIOS/UEFI would do. At the moment it just loops between netboot and localboot in this order, but I plan to make this more flexible and configurable.

## How to build systemboot

* Install a recent version of Go, we recommend 1.10 or later
* make sure that your PATH points appropriately to wherever Go stores the
  go-get'ed executables
* Then build it with the `u-root` ramfs builder using the following commands:

```
go get -u github.com/u-root/u-root
go get -u github.com/systemboot/systemboot/{uinit,localboot,netboot,stboot}
u-root -build=bb core github.com/systemboot/systemboot/{uinit,localboot,netboot,stboot}
```

The initramfs will be located in `/tmp/initramfs_${platform}_${arch}.cpio`.

More detailed information about the build process for a full LinuxBoot firmware image
using u-root/systemboot and coreboot can be found in the [LinuxBoot book](https://github.com/linuxboot/book)
chapter 11, [LinuxBoot using coreboot, u-root and systemboot](https://github.com/linuxboot/book/blob/master/11.coreboot.u-root.systemboot/README.md).

## Example: LinuxBoot with coreboot

One of the ways to create a LinuxBoot system firmware is by using
[coreboot](https://coreboot.org) do the basic silicon and DRAM initialization,
and then run Linux as payload, with u-root and systemboot as initramfs. See the
following diagram:

![LinuxBoot and coreboot](resources/LinuxBoot.png)
(images from coreboot.org and wikipedia.org, diagram generated with draw.io)

## Build and run as a fully open source bootloader in Qemu

Systemboot is one of the parts of a bigger picture: running Linux as firmware.
We call this [LinuxBoot](https://linuxboot.org), and it can be achieved in various
ways. One of these is by combining [coreboot](https://coreboot.org), [Linux](https://kernel.org),
[u-root](https://u-root.tk) and `systemboot`. Check out the instructions on the 
[LinuxBoot using coreboot, u-root and systemboot](https://github.com/linuxboot/book/tree/master/11.coreboot.u-root.systemboot)
chapter of the [LinuxBoot Book](https://github.com/linuxboot/book).

## TODO

* verified and measured boot
* a proper GRUB config parser
* backwards compatibility with BIOS-style partitions
