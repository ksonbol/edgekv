First do the same Grid'5000 stuff always required:
> frontend
oarsub -t deploy -l slash_22=1+nodes=3,walltime=00:50:00 -I
g5k-subnets -sp -> 10.144.4.0/22
kadeploy3 -f $OAR_NODE_FILE -e debian9-x64-nfs -k
distem-bootstrap --debian-version stretch
* note: store address returned by second step
===========
EXPERIMENT:
===========
# switch to coordinator node
frontend> 
ssh root@grisou-34

coord>
# create virtual network
distem --create-vnetwork vnetwork=vnetwork,address=10.144.4.0/22

# SKIP THIS IF YOU HAVE FILESYS IMG
###########
# go back to frontend
coord>
exit

#download system image (skip if done before)
frontend>
wget 'http://public.nancy.grid5000.fr/~amerlin/distem/distem-fs-jessie.tar.gz' -P ~/distem_img

# switch to coord
ssh root@grisou-34
##################
# create virtual nodes
> coord
distem --create-vnode vnode=node-1,pnode=grisou-34,\
rootfs=file:///home/ksonbol/distem_img/distem-fs-jessie.tar.gz,\
sshprivkey=/root/.ssh/id_rsa,sshpubkey=/root/.ssh/id_rsa.pub

distem --create-vnode vnode=node-2,pnode=grisou-34,\
rootfs=file:///home/ksonbol/distem_img/distem-fs-jessie.tar.gz,\
sshprivkey=/root/.ssh/id_rsa,sshpubkey=/root/.ssh/id_rsa.pub

# create network interfaces on each vnode
> coord
distem --create-viface vnode=node-1,iface=if0,vnetwork=vnetwork
distem --create-viface vnode=node-2,iface=if0,vnetwork=vnetwork

node1-address: 10.144.4.1/22
node2-address: 10.144.4.2/22

distem --start-vnode node-1
distem --start-vnode node-2



    connection in a node to get a shell (user:root, password:root):

    coord> distem --shell node-1

    This option uses lxc-console, so the exit key sequence (ctrl-a + q by default) can be modified if you are running inside a screen session.

    execution of a command:

    coord> distem --execute vnode=node-1,command="hostname"


