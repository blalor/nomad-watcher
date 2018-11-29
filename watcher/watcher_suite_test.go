package watcher_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "github.com/onsi/ginkgo/reporters"
    "github.com/onsi/ginkgo/config"
    "testing"

    "fmt"
)

func TestSubmodule(t *testing.T) {
    RegisterFailHandler(Fail)
    junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("junit_%d.xml", config.GinkgoConfig.ParallelNode))

    RunSpecsWithDefaultAndCustomReporters(t, "Watcher Suite", []Reporter{junitReporter})
}
