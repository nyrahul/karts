# KubeArmor Automated Tests

## Prerequisites
1. Install ginkgo `go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo`

## How to add a new testsuite?

```
mkdir $BASE/newsuite && cd $BASE/newsuite
ginkgo bootstrap # This adds the skeleton for a new testcase
```

## Adding new testcases
Refer to `$BASE/smoke` folder.

For making assertions in testspec using gomega, [check this](https://onsi.github.io/gomega/#making-assertions).

## Testsuite execution

1. Execute all testsuites using `ginkgo -r`
2. Execute specific testsuite using `ginkgo --focus "Smoke"`
