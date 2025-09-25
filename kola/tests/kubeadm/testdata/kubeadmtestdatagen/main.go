package main

import (
	"fmt"
	"os"

	"github.com/flatcar/mantle/kola/tests/kubeadm"
)

func main() {
	outputDirectory := "."
	if len(os.Args) > 1 {
		outputDirectory = os.Args[1]
	}
	for _, CNI := range kubeadm.CNIs {
		renderToFile(
			fmt.Sprintf("master script for %s", CNI),
			kubeadm.GetMasterScript(),
			kubeadm.GetTestMasterScriptRenderParams(CNI),
			fmt.Sprintf("%s/master-%s-script.sh", outputDirectory, CNI),
		)
	}
	for _, arch := range kubeadm.TestArchitectures {
		renderToFile(
			fmt.Sprintf("master cilium config for %s", arch),
			kubeadm.GetMasterConfig(),
			kubeadm.GetTestMasterConfigRenderParams(arch),
			fmt.Sprintf("%s/master-cilium-%s-config.yml", outputDirectory, arch),
		)
	}
}

func renderToFile(what, template string, parameters map[string]interface{}, outputPath string) {
	res, err := kubeadm.Render(template, parameters, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to render %s: %v\n", what, err)
		os.Exit(1)
	}
	err = os.WriteFile(outputPath, res.Bytes(), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s for %s: %v\n", outputPath, what, err)
		os.Exit(1)
	}
}
