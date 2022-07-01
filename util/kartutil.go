package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	kins "github.com/kubearmor/kubearmor-client/install"
	kcli "github.com/kubearmor/kubearmor-client/k8s"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

var k8sClient *kcli.Client
var stopChan chan struct{}

func isK8sEnv() bool {
	if k8sClient != nil {
		return true
	}
	cli, err := kcli.ConnectK8sClient()
	if err != nil {
		return false
	}
	k8sClient = cli
	return true
}

func getOptions() kins.Options {
	return kins.Options{
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
	err := kins.K8sInstaller(k8sClient, getOptions())
	if err != nil {
		log.Error("failed to install kubearmor err=%s", err)
		return err
	}
	return nil
}

func k8sUninstallKubearmor() {
	if !isK8sEnv() {
		log.Error("could not find k8s env to uninstall kubearmor")
		return
	}
	err := kins.K8sUninstaller(k8sClient, getOptions())
	if err != nil {
		log.Error("failed to install kubearmor err=%s", err)
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
			log.Errorf("could not get daemonsets error:%s", err)
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
			log.Errorf("could not get deployment. error:%s", err)
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

func K8sGetPods(podstr string, ns string, timeout int) ([]string, error) {
	podList, err := k8sClient.K8sClientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("k8s list pods failed. error=%s", err)
		return nil, err
	}
	pods := []string{}
	for t := 0; t <= timeout; t++ {
		for _, p := range podList.Items {
			if p.Status.Phase != v1.PodRunning {
				continue
			}
			if strings.HasPrefix(p.ObjectMeta.Name, podstr) {
				log.Printf("prefix match prefix=%s podName=%s", podstr, p.ObjectMeta.Name)
				pods = append(pods, p.ObjectMeta.Name)
			} else if match, _ := regexp.MatchString(podstr, p.ObjectMeta.Name); match {
				log.Printf("regexp match prefix=%s podName=%s", podstr, p.ObjectMeta.Name)
				pods = append(pods, p.ObjectMeta.Name)
			}
		}
		if timeout == 0 || len(pods) > 0 {
			break
		}
		time.Sleep(1 * time.Second)
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
	if k8sMode {
		log.Println("starting kubearmor")
		err := k8sInstallKubearmor()
		if err != nil {
			log.Errorf("start kubearmor failed error=%s", err)
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
	cmdf := []string{"--request-timeout", "20s"}
	cmdf = append(cmdf, strings.Fields(cmdstr)...)
	cmd := exec.Command("kubectl", cmdf...)
	sout, err := cmd.Output()
	return string(sout), err
}

// K8sApply execute
func K8sApply(files []string) error {
	for _, f := range files {
		_, err := Kubectl(fmt.Sprintf("apply -f %s", f))
		if err != nil {
			return err
		}
	}
	return nil
}

// K8sDelete
func K8sDelete(files []string) error {
	for _, f := range files {
		_, err := Kubectl(fmt.Sprintf("delete -f %s", f))
		if err != nil {
			return err
		}
	}
	return nil
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
