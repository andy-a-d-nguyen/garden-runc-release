package main_test

import (
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/grootfs/commands/config"
	yaml "gopkg.in/yaml.v2"
)

var _ = Describe("Thresholder", func() {
	var (
		reservedSpace       string
		thresholderCmd      *exec.Cmd
		pathToDisk          string
		pathToGrootfsConfig string
	)

	exitsNonZeroWithMessage := func(message string) {
		It("prints an informative error message", func() {
			session := gexecStartAndWait(thresholderCmd, GinkgoWriter, GinkgoWriter)
			Expect(string(session.Out.Contents())).To(ContainSubstring(message))
		})

		It("exits non zero", func() {
			session := gexecStartAndWait(thresholderCmd, GinkgoWriter, GinkgoWriter)
			Expect(session.ExitCode()).NotTo(BeZero())
		})
	}

	resultingConfig := func() *config.Config {
		gexecStartAndWait(thresholderCmd, GinkgoWriter, GinkgoWriter)
		return configFromFile(pathToGrootfsConfig)
	}

	BeforeEach(func() {
		reservedSpace = "3000"
		pathToDisk = diskMountPath
		pathToGrootfsConfigAsset := filepath.Join("testassets", "grootfs.yml")
		pathToGrootfsConfig = copyFileToTempFile(pathToGrootfsConfigAsset)
	})

	JustBeforeEach(func() {
		thresholderCmd = exec.Command(thresholderBin, reservedSpace, pathToDisk, pathToGrootfsConfig)
	})

	AfterEach(func() {
		os.Remove(pathToGrootfsConfig)
	})

	It("sets clean.threshold_bytes", func() {
		reservedSpaceInt, err := strconv.ParseInt(reservedSpace, 10, 64)
		Expect(err).NotTo(HaveOccurred())
		Expect(resultingConfig().Clean.ThresholdBytes).To(Equal(diskSize - megabytesToBytes(reservedSpaceInt)))
	})

	It("sets init.store_size_bytes", func() {
		reservedSpaceInt, err := strconv.ParseInt(reservedSpace, 10, 64)
		Expect(err).NotTo(HaveOccurred())
		Expect(resultingConfig().Init.StoreSizeBytes).To(Equal(diskSize - megabytesToBytes(reservedSpaceInt)))
	})

	It("sets create.with_clean", func() {
		Expect(resultingConfig().Create.WithClean).To(BeTrue())
	})

	When("the store path doesn't exist", func() {
		BeforeEach(func() {
			pathToDisk = "/path/to/foo/bar"
			Expect(pathToDisk).NotTo(BeADirectory())
		})

		exitsNonZeroWithMessage(pathToDisk)
	})

	Describe("Parameters validation", func() {
		Context("when not all input args are provided", func() {
			JustBeforeEach(func() {
				thresholderCmd = exec.Command(thresholderBin, "1", "2", "3", "4")
			})

			exitsNonZeroWithMessage("Not all input arguments provided (Expected: 3)")
		})

		Context("when reserved space parameter cannot be parsed", func() {
			BeforeEach(func() {
				reservedSpace = "abc"
			})

			exitsNonZeroWithMessage("Reserved space parameter must be a number")
		})

		Context("when grootfs configfile does not exist", func() {
			BeforeEach(func() {
				pathToGrootfsConfig = "not/a/path"
			})

			exitsNonZeroWithMessage("Grootfs config parameter must be path to valid grootfs config file")
		})

		Context("when grootfs configfile does not contain valid grootfs config", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(pathToGrootfsConfig, []byte("not yaml"), 0600)).To(Succeed())
			})

			exitsNonZeroWithMessage("Grootfs config parameter must be path to valid grootfs config file")
		})
	})
})

func copyFileToTempFile(src string) string {
	fileContents, err := ioutil.ReadFile(src)
	Expect(err).NotTo(HaveOccurred())

	tempFile, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, bytes.NewReader(fileContents))
	Expect(err).NotTo(HaveOccurred())

	return tempFile.Name()
}

func configFromFile(path string) *config.Config {
	conf, err := ioutil.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())

	var c config.Config
	Expect(yaml.Unmarshal(conf, &c)).To(Succeed())

	return &c
}

func megabytesToBytes(megabytes int64) int64 {
	return int64(megabytes * 1024 * 1024)
}
