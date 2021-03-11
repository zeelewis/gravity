echo "setting up RBAC for Kubernetes"
/usr/local/bin/kubectl --kubeconfig=/etc/kubernetes/scheduler.kubeconfig apply -f /var/lib/gravity/resources/resources.yaml
echo "setting up RBAC for ETCD"
/usr/local/bin/etcdctl --cert-file /var/state/etcd.cert --key-file /var/state/etcd.key user add flannel
/usr/local/bin/etcdctl --cert-file /var/state/etcd.cert --key-file /var/state/etcd.key role add flannel_readwrite_role
/usr/local/bin/etcdctl --cert-file /var/state/etcd.cert --key-file /var/state/etcd.key role grant flannel_readwrite_role --readwrite --path /coreos.com/network
/usr/local/bin/etcdctl --cert-file /var/state/etcd.cert --key-file /var/state/etcd.key user grant --roles flannel_readwrite_role flannel
        