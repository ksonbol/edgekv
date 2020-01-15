#!/bin/sh

# Usage: ./disk_prep.sh DISK MOUNT_POINT
# create one partition, the size of the disk, e.g., /dev/sdb1 from /dev/sdb
echo 'type=83' | sudo sfdisk $1

# needed sleep for mkfs to see the /dev/sdX file
sleep 1
# make a file system on the crated partion e.g., /dev/sdb1
mkfs.ext4 -F "${1}1"

# added another sleep just in case!
sleep 1
# mount the created partition on the mount point
mount "${1}1" $2
