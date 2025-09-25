package kubeadm

import "bytes"

var (
	TestArchitectures = []string{"amd64", "arm64"}
)

func GetTestMasterScriptRenderParams(cni string) map[string]interface{} {
	return map[string]interface{}{
		"HelmVersion":    "1.2.3",
		"CiliumVersion":  "v0.11.1",
		"FlannelVersion": "v0.14.0",
		"CNI":            cni,
		"Endpoints":      []string{"http://1.2.3.4:2379"},
		"Params":         "amd64",
		"DownloadDir":    "/opt/bin",
		"PodSubnet":      "192.168.0.0/17",
	}
}

func GetTestMasterConfigRenderParams(arch string) map[string]interface{} {
	return map[string]interface{}{
		"HelmVersion":      "1.2.3",
		"CiliumVersion":    "v0.11.1",
		"CNI":              "cilium",
		"CiliumCLIVersion": "v0.9.0",
		"Endpoints":        []string{"http://1.2.3.4:2379"},
		"Arch":             arch,
		"DownloadDir":      "/opt/bin",
		"PodSubnet":        "192.168.0.0/17",
		"Release":          "v1.29.2",
	}
}

func GetMasterScript() string {
	return masterScript
}

func GetMasterConfig() string {
	return masterConfig
}

func Render(s string, p map[string]interface{}, b bool) (*bytes.Buffer, error) {
	return render(s, p, b)
}
