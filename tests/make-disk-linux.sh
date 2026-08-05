#!/bin/bash
# Creates a phony RHEL 9.2 disk image for testing kc-prepare/kc-finalize.
# Usage: make-disk-linux.sh <output.img>
set -e

export LIBGUESTFS_BACKEND=direct

IMG="${1:?Usage: make-disk-linux.sh <output.img>}"
rm -f "$IMG"

guestfish <<EOF
disk-create $IMG raw 512M
add $IMG
run

part-init /dev/sda mbr
part-add /dev/sda p 2048 1048575

mkfs ext4 /dev/sda1
mount /dev/sda1 /

mkdir-p /etc/modprobe.d
write /etc/os-release "NAME=\"Red Hat Enterprise Linux\"\nVERSION=\"9.2 (Plow)\"\nID=\"rhel\"\nID_LIKE=\"fedora\"\nVERSION_ID=\"9.2\"\nPRETTY_NAME=\"Red Hat Enterprise Linux 9.2 (Plow)\"\n"
write /etc/fstab "/dev/sda1 / ext4 defaults 0 1\n"

mkdir-p /boot/grub2
write /boot/grub2/grub.cfg "menuentry 'RHEL 9' {\n    linux /boot/vmlinuz-5.14.0-284 root=/dev/sda1\n    initrd /boot/initramfs-5.14.0-284.img\n}\n"
touch /boot/vmlinuz-5.14.0-284
touch /boot/initramfs-5.14.0-284.img

mkdir-p /usr/lib/modules/5.14.0-284/kernel/drivers/virtio
touch /usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_blk.ko
touch /usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_net.ko
touch /usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_scsi.ko
touch /usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_pci.ko

umount-all
EOF

echo "Created $IMG"
