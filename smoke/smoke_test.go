package smoke

import (
	"fmt"

	util "github.com/kubearmor/karts/kartutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = BeforeSuite(func() {
	log.Println("====================[ BeforeSuite ]========================")
	err := util.StartKubearmor(true)
	Expect(err).To(BeNil())
})

var _ = AfterSuite(func() {
	log.Println("====================[ AfterSuite ]========================")
})

var _ = Describe("Smoke", func() {

	BeforeEach(func() {
		log.Println("--------------[ BeforeEach ]------------------")
	})

	AfterEach(func() {
		log.Println("--------------[ AfterEach ]------------------")
		util.KspDeleteAll()
	})

	Describe("Policy Apply", func() {
		It("can block execution of cmd as part of parent process", func() {
			// Get wordpress pod
			pods, err := util.K8sGetPods("wordpress", "wordpress-mysql")
			Expect(err).To(BeNil())
			Expect(len(pods)).To(Equal(1))

			// Apply policy
			sout, err := util.Kubectl("apply -f ksp-wordpress-block-process.yaml")
			Expect(err).To(BeNil())
			fmt.Println(sout)

			// exec command in pod
			sout, _, err = util.K8sExecInPod(pods[0], "wordpress-mysql", []string{"bash", "-c", "apt"})
			Expect(err).To(BeNil())
			fmt.Println(sout)
			Expect(sout).To(ContainSubstring("Permission denied"))

			// Validate alert event
		})
	})

})
