package kartutil

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	karmorins "github.com/kubearmor/kubearmor-client/install"
	karmorcli "github.com/kubearmor/kubearmor-client/k8s"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

var k8sClient *karmorcli.Client

func isK8sEnv() bool {
	if k8sClient != nil {
		return true
	}
	cli, err := karmorcli.ConnectK8sClient()
	if err != nil {
		return false
	}
	k8sClient = cli
	return true
}

func getOptions() karmorins.Options {
	return karmorins.Options{
		"kube-system",
		"kubearmor/kubearmor:stable",
		"",
		false,
	}
}

func k8sInstallKubearmor() error {
	if !isK8sEnv() {
		return errors.New("could not find k8s env")
	}
	err := karmorins.K8sInstaller(k8sClient, getOptions())
	if err != nil {
		log.Error("failed to install kubearmor")
		return err
	}
	return nil
}

func k8sUninstallKubearmor() {
	if !isK8sEnv() {
		log.Error("could not find k8s env to uninstall kubearmor")
		return
	}
	err := karmorins.K8sUninstaller(k8sClient, getOptions())
	if err != nil {
		log.Error("failed to install kubearmor")
		return
	}
}

func K8sDaemonSetCheck(dsname string, ns string, timeout int) (string, error) {
	if !isK8sEnv() {
		log.Error("could not find k8s env dscheck")
		return "", errors.New("no k8s env")
	}
	status := ""
	for t := 0; t <= timeout; t++ {
		dsset, err := k8sClient.K8sClientset.AppsV1().DaemonSets(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Errorf("could not get daemonsets error:%s", err.Error())
			return "", err
		}
		for _, ds := range dsset.Items {
			if dsname == ds.ObjectMeta.Name {
				if ds.Status.NumberReady > 0 {
					return "ready", nil
				} else {
					status = "not-ready"
				}
			}
		}
		if timeout == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if status != "" {
		return status, errors.New("daemonset not ready")
	}
	return "", errors.New("daemonset not found")
}

func K8sDeploymentCheck(depname string, ns string, timeout int) (string, error) {
	if !isK8sEnv() {
		log.Error("could not find k8s env dscheck")
		return "", errors.New("no k8s env")
	}
	status := ""
	for t := 0; t <= timeout; t++ {
		depset, err := k8sClient.K8sClientset.AppsV1().Deployments(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Errorf("could not get deployment. error:%s", err.Error())
			return "", err
		}
		for _, dep := range depset.Items {
			if depname == dep.ObjectMeta.Name {
				if dep.Status.ReadyReplicas == dep.Status.Replicas {
					return "ready", nil
				} else {
					status = "not-ready"
				}
			}
		}
		if timeout == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if status != "" {
		return status, errors.New("deployment not ready")
	}
	return "", errors.New("deployment not found")
}

func K8sGetPods(podPrefix string, ns string) ([]string, error) {
	podList, err := k8sClient.K8sClientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("k8s list pods failed. error=%s", err.Error())
		return nil, err
	}
	pods := []string{}
	for _, p := range podList.Items {
		if strings.HasPrefix(p.ObjectMeta.Name, podPrefix) {
			pods = append(pods, p.ObjectMeta.Name)
		}
	}
	if len(pods) == 0 {
		return nil, errors.New("pod not found")
	}
	return pods, nil
}

// K8sExecInPod Exec into the pod. Output: stdout, stderr, err
func K8sExecInPod(pod string, ns string, cmd []string) (string, string, error) {
	req := k8sClient.K8sClientset.CoreV1().RESTClient().Post().Resource("pods").Name(pod).Namespace(ns).SubResource("exec")
	option := &v1.PodExecOptions{
		Command: cmd,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(k8sClient.Config, "POST", req.URL())
	if err != nil {
		return "", "", err
	}
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	return buf.String(), errBuf.String(), nil
}

func StartKubearmor(k8sMode bool) error {
	if !isK8sEnv() {
		log.Error("could not find k8s env")
		return errors.New("no k8s env")
	}
	return nil //XXX REMOVE THIS
	if k8sMode {
		log.Println("starting kubearmor")
		err := k8sInstallKubearmor()
		if err != nil {
			log.Errorf("start kubearmor failed error=%s", err.Error())
			return err
		}
	} else {
		return errors.New("unknown mode, systemd mode not supported yet")
	}
	status, err := K8sDaemonSetCheck("kubearmor", "kube-system", 20)
	if status == "ready" && err == nil {
		return nil
	}
	return nil
}

// Kubectl execute
func Kubectl(cmdstr string) (string, error) {
	cmdf := strings.Fields(cmdstr)
	cmd := exec.Command("kubectl", cmdf...)
	sout, err := cmd.Output()
	return string(sout), err
}

func KspDeleteAll() {
	sout, err := Kubectl("get ksp -A --no-headers -o custom-columns=:metadata.name,:metadata.namespace")
	if err != nil {
		return
	}
	lines := strings.Split(sout, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		field := strings.Fields(line)
		Kubectl("delete ksp " + field[0] + " -n " + field[1])
	}
}
