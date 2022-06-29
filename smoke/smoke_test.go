package smoke

import (
	"fmt"
	"time"

	. "github.com/kubearmor/karts/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	// start kubearmor
	err := StartKubearmor(true)
	Expect(err).To(BeNil())

	// install wordpress-mysql app
	err = K8sApply([]string{"wordpress-mysql-deployment.yaml"})
	Expect(err).To(BeNil())

	// enable kubearmor port forwarding
	err = KubearmorPortForward()
	Expect(err).To(BeNil())

})

var _ = AfterSuite(func() {
	KubearmorPortForwardStop()
})

var _ = Describe("Smoke", func() {

	BeforeEach(func() {
	})

	AfterEach(func() {
		KarmorLogStop()
		KspDeleteAll()
	})

	Describe("Policy Apply", func() {
		It("can block execution of cmd as part of parent process", func() {
			// Get wordpress pod
			pods, err := K8sGetPods("wordpress", "wordpress-mysql")
			Expect(err).To(BeNil())
			Expect(len(pods)).To(Equal(1))

			// Apply policy
			err = K8sApply([]string{"ksp-wordpress-block-process.yaml"})
			Expect(err).To(BeNil())

			// Start Kubearmor Logs
			err = KarmorLogStart("policy", "wordpress-mysql", "Process", pods[0])
			Expect(err).To(BeNil())

			actCnt := 3
			// exec command in pod
			for i := 0; i < actCnt; i++ {
				sout, _, err := K8sExecInPod(pods[0], "wordpress-mysql", []string{"bash", "-c", "apt"})
				Expect(err).To(BeNil())
				fmt.Println(sout)
				Expect(sout).To(ContainSubstring("Permission denied"))
				time.Sleep(10 * time.Millisecond)
			}

			// check policy violation alert
			logs, alerts, err := KarmorGetLogs(5*time.Second, actCnt)
			Expect(err).To(BeNil())
			Expect(len(logs)).To(BeNumerically("==", 0))
			Expect(len(alerts)).To(BeNumerically(">=", 1))
			fmt.Printf("Lost %d events from kubearmor\n", actCnt-len(alerts))
			Expect(alerts[0].PolicyName).To(Equal("ksp-wordpress-block-process"))
			Expect(alerts[0].Severity).To(Equal("3"))
		})
	})

})
