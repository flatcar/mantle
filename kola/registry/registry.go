package registry

// Tests imported for registration side effects. These make up the OS test suite and is explicitly imported from the main package.
import (
	_ "github.com/flatcar-linux/mantle/kola/tests/coretest"
	_ "github.com/flatcar-linux/mantle/kola/tests/crio"
	_ "github.com/flatcar-linux/mantle/kola/tests/docker"
	_ "github.com/flatcar-linux/mantle/kola/tests/etcd"
	_ "github.com/flatcar-linux/mantle/kola/tests/flannel"
	_ "github.com/flatcar-linux/mantle/kola/tests/ignition"
	_ "github.com/flatcar-linux/mantle/kola/tests/kubeadm"
	_ "github.com/flatcar-linux/mantle/kola/tests/kubernetes"
	_ "github.com/flatcar-linux/mantle/kola/tests/locksmith"
	_ "github.com/flatcar-linux/mantle/kola/tests/metadata"
	_ "github.com/flatcar-linux/mantle/kola/tests/misc"
	_ "github.com/flatcar-linux/mantle/kola/tests/ostree"
	_ "github.com/flatcar-linux/mantle/kola/tests/packages"
	_ "github.com/flatcar-linux/mantle/kola/tests/podman"
	_ "github.com/flatcar-linux/mantle/kola/tests/rkt"
	_ "github.com/flatcar-linux/mantle/kola/tests/rpmostree"
	_ "github.com/flatcar-linux/mantle/kola/tests/systemd"
	_ "github.com/flatcar-linux/mantle/kola/tests/torcx"
	_ "github.com/flatcar-linux/mantle/kola/tests/update"
)
