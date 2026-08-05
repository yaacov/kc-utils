#!/bin/bash
# Creates a phony UEFI RHEL 9.2 disk image (GPT + ESP + ext4).
# Usage: make-disk-linux-uefi.sh <output.img>
set -e

export LIBGUESTFS_BACKEND=direct

IMG="${1:?Usage: make-disk-linux-uefi.sh <output.img>}"
rm -f "$IMG"

guestfish <<EOF
disk-create $IMG raw 512M
add $IMG
run

part-init /dev/sda gpt

part-add /dev/sda p 2048 411647
part-set-gpt-type /dev/sda 1 C12A7328-F81F-11D2-BA4B-00A0C93EC93B

part-add /dev/sda p 411648 -2048

mkfs vfat /dev/sda1
mkfs ext4 /dev/sda2

mount /dev/sda2 /

mkdir-p /etc/modprobe.d
write /etc/os-release "NAME=\"Red Hat Enterprise Linux\"\nVERSION=\"9.2 (Plow)\"\nID=\"rhel\"\nID_LIKE=\"fedora\"\nVERSION_ID=\"9.2\"\nPRETTY_NAME=\"Red Hat Enterprise Linux 9.2 (Plow)\"\n"
write /etc/fstab "/dev/sda2 / ext4 defaults 0 1\n/dev/sda1 /boot/efi vfat defaults 0 2\n"

mkdir-p /boot/grub2
write /boot/grub2/grub.cfg "menuentry 'RHEL 9' {\n    linux /boot/vmlinuz-5.14.0-284 root=/dev/sda2\n    initrd /boot/initramfs-5.14.0-284.img\n}\n"
touch /boot/vmlinuz-5.14.0-284
touch /boot/initramfs-5.14.0-284.img

mkdir-p /usr/lib/modules/5.14.0-284/kernel/drivers/virtio
touch /usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_blk.ko
touch /usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_net.ko
touch /usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_scsi.ko
touch /usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_pci.ko

mkdir-p /boot/efi/EFI/redhat
touch /boot/efi/EFI/redhat/grubx64.efi
touch /boot/efi/EFI/redhat/shimx64.efi

umount-all
EOF

echo "Created $IMG"
